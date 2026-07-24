package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/XxKotfeJxX/netscope/internal/network"
	dnsprobe "github.com/XxKotfeJxX/netscope/internal/probe/dns"
	httpProbe "github.com/XxKotfeJxX/netscope/internal/probe/httpcheck"
	pingProbe "github.com/XxKotfeJxX/netscope/internal/probe/ping"
	tcpProbe "github.com/XxKotfeJxX/netscope/internal/probe/tcp"
	tlsProbe "github.com/XxKotfeJxX/netscope/internal/probe/tlscheck"
	traceProbe "github.com/XxKotfeJxX/netscope/internal/probe/traceroute"
	"github.com/XxKotfeJxX/netscope/internal/reports"
	"github.com/XxKotfeJxX/netscope/internal/storage/postgres"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/XxKotfeJxX/netscope/internal/transport/httpapi"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config    Config
	logger    *slog.Logger
	pool      *pgxpool.Pool
	manager   *diagnostics.Manager
	scheduler *monitoring.Scheduler
	server    *http.Server
}

func New(ctx context.Context, cfg Config, logger *slog.Logger) (*App, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	poolConfig.MaxConns = 10

	migrationCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := postgres.Migrate(migrationCtx, poolConfig); err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	repository := postgres.NewRunRepository(pool)
	events := diagnostics.NewHub()
	policy := target.Policy{
		Public:         cfg.NetworkPolicy == "public",
		AllowLoopback:  cfg.AllowLoopback,
		AllowPrivate:   cfg.AllowPrivate,
		AllowLinkLocal: cfg.AllowLinkLocal,
	}
	secureDialer := network.NewSecureDialer(nil, policy, cfg.MaxProbeTimeout)
	icmpRunner := network.RawICMPRunner{}
	icmpAvailable, icmpReason := icmpRunner.Available()
	if !cfg.ICMPEnabled {
		icmpAvailable = false
		icmpReason = "disabled_by_config"
	}
	probes := []diagnostics.Probe{
		dnsprobe.New(nil),
		tcpProbe.New(secureDialer),
		httpProbe.New(secureDialer),
		tlsProbe.New(secureDialer),
	}
	if icmpAvailable {
		probes = append(
			probes,
			pingProbe.New(secureDialer, icmpRunner),
			traceProbe.New(secureDialer, icmpRunner),
		)
	}
	manager := diagnostics.NewManager(
		repository,
		events,
		logger,
		cfg.RunWorkers,
		cfg.RunQueueSize,
		cfg.ProbeConcurrency,
		probes...,
	)
	manager.Start()
	service := diagnostics.NewService(
		repository,
		manager,
		policy,
		cfg.DefaultProbeTimeout,
		cfg.MaxProbeTimeout,
	)
	monitoringRepository := postgres.NewMonitoringRepository(pool)
	monitoringService := monitoring.NewService(monitoringRepository, service)
	identityRepository := postgres.NewIdentityRepository(pool)
	identityService := identity.NewService(
		identityRepository,
		identity.NewPasswordHasher(identity.DefaultPasswordParams()),
		cfg.SessionTTL,
	)
	collaborationService := collaboration.NewService(
		postgres.NewCollaborationRepository(pool),
	)
	reportsService := reports.NewService(
		postgres.NewReportsRepository(pool),
		service,
	)
	webhookSender := monitoring.NewWebhookSender(
		secureDialer,
		cfg.NotificationTimeout,
		cfg.Version,
	)
	var emailSender monitoring.EmailDelivery
	if cfg.SMTPHost != "" {
		emailSender = monitoring.NewSMTPSender(monitoring.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort,
			Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
			From: cfg.SMTPFrom, TLSMode: cfg.SMTPTLSMode,
			Timeout: cfg.NotificationTimeout,
		})
	}
	notifier := monitoring.NewChannelNotifier(
		monitoringRepository,
		webhookSender,
		emailSender,
	)
	scheduler := monitoring.NewScheduler(
		monitoringRepository,
		service,
		notifier,
		logger,
		cfg.MonitoringInterval,
	)
	scheduler.Start()

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:              logger,
		Pool:                pool,
		Version:             cfg.Version,
		WebOrigin:           cfg.WebOrigin,
		Runs:                service,
		Events:              events,
		Monitoring:          monitoringService,
		Identity:            identityService,
		Collaboration:       collaborationService,
		Reports:             reportsService,
		SessionCookieSecure: cfg.SessionCookieSecure,
		Checks: map[string]httpapi.Capability{
			"dns": {Available: true}, "tcp": {Available: true},
			"http": {Available: true}, "tls": {Available: true},
			"ping":       {Available: icmpAvailable, Reason: icmpReason},
			"traceroute": {Available: icmpAvailable, Reason: icmpReason},
		},
		Runtime: httpapi.RuntimeInfo{
			DefaultTimeoutMS: int(cfg.DefaultProbeTimeout.Milliseconds()),
			MaxTimeoutMS:     int(cfg.MaxProbeTimeout.Milliseconds()),
			RunWorkers:       cfg.RunWorkers,
			ProbeConcurrency: cfg.ProbeConcurrency,
			NetworkPolicy:    cfg.NetworkPolicy,
		},
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		config: cfg, logger: logger, pool: pool, manager: manager,
		scheduler: scheduler, server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.pool.Close()

	serverErrors := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server listening", "address", a.server.Addr)
		serverErrors <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		if err := a.scheduler.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown monitoring scheduler: %w", err)
		}
		if err := a.manager.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown diagnostic workers: %w", err)
		}
		a.logger.Info("graceful shutdown complete")
		return nil
	case err := <-serverErrors:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = a.scheduler.Shutdown(shutdownCtx)
		_ = a.manager.Shutdown(shutdownCtx)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}
