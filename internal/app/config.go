package app

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment         string
	Version             string
	HTTPHost            string
	HTTPPort            int
	WebOrigin           string
	DatabaseURL         string
	RunWorkers          int
	RunQueueSize        int
	ProbeConcurrency    int
	DefaultProbeTimeout time.Duration
	MaxProbeTimeout     time.Duration
	MaxTCPPorts         int
	MaxRedirects        int
	NetworkPolicy       string
	AllowLoopback       bool
	AllowPrivate        bool
	AllowLinkLocal      bool
	ICMPEnabled         bool
	MonitoringInterval  time.Duration
	LogLevel            slog.Level
	LogFormat           string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Environment:         envString("APP_ENV", "development"),
		Version:             envString("APP_VERSION", "dev"),
		HTTPHost:            envString("HTTP_HOST", "0.0.0.0"),
		WebOrigin:           envString("WEB_ORIGIN", "http://localhost:5173"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		NetworkPolicy:       envString("NETWORK_POLICY", "local"),
		LogFormat:           envString("LOG_FORMAT", "json"),
		DefaultProbeTimeout: 5 * time.Second,
		MaxProbeTimeout:     30 * time.Second,
		MonitoringInterval:  10 * time.Second,
		LogLevel:            slog.LevelInfo,
	}

	var err error
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.HTTPPort, err = envInt("HTTP_PORT", 8080, 1, 65535); err != nil {
		return Config{}, err
	}
	if cfg.RunWorkers, err = envInt("RUN_WORKERS", 4, 1, 64); err != nil {
		return Config{}, err
	}
	if cfg.RunQueueSize, err = envInt("RUN_QUEUE_SIZE", 100, 1, 10000); err != nil {
		return Config{}, err
	}
	if cfg.ProbeConcurrency, err = envInt("PROBE_CONCURRENCY", 8, 1, 64); err != nil {
		return Config{}, err
	}
	if cfg.MaxTCPPorts, err = envInt("MAX_TCP_PORTS", 10, 1, 100); err != nil {
		return Config{}, err
	}
	if cfg.MaxRedirects, err = envInt("MAX_REDIRECTS", 5, 0, 10); err != nil {
		return Config{}, err
	}
	if cfg.DefaultProbeTimeout, err = envDuration("DEFAULT_PROBE_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.MaxProbeTimeout, err = envDuration("MAX_PROBE_TIMEOUT", 30*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.MonitoringInterval, err = envDuration(
		"MONITORING_POLL_INTERVAL",
		10*time.Second,
	); err != nil {
		return Config{}, err
	}
	if cfg.DefaultProbeTimeout > cfg.MaxProbeTimeout {
		return Config{}, fmt.Errorf("DEFAULT_PROBE_TIMEOUT must not exceed MAX_PROBE_TIMEOUT")
	}
	if cfg.AllowLoopback, err = envBool("ALLOW_LOOPBACK_TARGETS", true); err != nil {
		return Config{}, err
	}
	if cfg.AllowPrivate, err = envBool("ALLOW_PRIVATE_TARGETS", true); err != nil {
		return Config{}, err
	}
	if cfg.AllowLinkLocal, err = envBool("ALLOW_LINK_LOCAL_TARGETS", false); err != nil {
		return Config{}, err
	}
	if cfg.ICMPEnabled, err = envBool("ICMP_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.NetworkPolicy != "local" && cfg.NetworkPolicy != "public" {
		return Config{}, fmt.Errorf("NETWORK_POLICY must be local or public")
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(envString("LOG_LEVEL", "info"))); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL: %w", err)
	}
	if cfg.LogFormat != "json" && cfg.LogFormat != "text" {
		return Config{}, fmt.Errorf("LOG_FORMAT must be json or text")
	}

	return cfg, nil
}

func NewLogger(cfg Config) *slog.Logger {
	options := &slog.HandlerOptions{Level: cfg.LogLevel}
	var handler slog.Handler
	if cfg.LogFormat == "text" {
		handler = slog.NewTextHandler(io.Writer(os.Stdout), options)
	} else {
		handler = slog.NewJSONHandler(io.Writer(os.Stdout), options)
	}
	return slog.New(handler).With("service", "netscope", "version", cfg.Version)
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback, minValue, maxValue int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minValue || parsed > maxValue {
		return 0, fmt.Errorf("%s must be between %d and %d", key, minValue, maxValue)
	}
	return parsed, nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func envBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return parsed, nil
}
