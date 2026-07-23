package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/storage/postgres"
	"github.com/XxKotfeJxX/netscope/internal/transport/httpapi"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	config Config
	logger *slog.Logger
	pool   *pgxpool.Pool
	server *http.Server
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

	router := httpapi.NewRouter(httpapi.Dependencies{
		Logger:    logger,
		Pool:      pool,
		Version:   cfg.Version,
		WebOrigin: cfg.WebOrigin,
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return &App{config: cfg, logger: logger, pool: pool, server: server}, nil
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
		a.logger.Info("graceful shutdown complete")
		return nil
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	}
}
