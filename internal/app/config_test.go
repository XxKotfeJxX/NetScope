package app

import (
	"strings"
	"testing"
)

func TestLoadConfigProductionSecurityDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://netscope:test@db/netscope")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WEB_ORIGIN", "https://netscope.example")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !cfg.SessionCookieSecure || !cfg.CSRFProtection {
		t.Fatalf(
			"production security defaults = secure cookie %t, csrf %t",
			cfg.SessionCookieSecure,
			cfg.CSRFProtection,
		)
	}
	if cfg.TrustedProxyHeaders {
		t.Fatal("proxy headers are trusted by default")
	}
}

func TestLoadConfigRejectsInsecureProductionOrigin(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://netscope:test@db/netscope")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WEB_ORIGIN", "http://netscope.example")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("LoadConfig() error = %v, want HTTPS validation", err)
	}
}

func TestLoadConfigRejectsOriginWithPath(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://netscope:test@db/netscope")
	t.Setenv("WEB_ORIGIN", "https://netscope.example/application")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("LoadConfig() error = %v, want origin validation", err)
	}
}

func TestLoadConfigRejectsDisabledProductionCSRF(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://netscope:test@db/netscope")
	t.Setenv("APP_ENV", "production")
	t.Setenv("WEB_ORIGIN", "https://netscope.example")
	t.Setenv("CSRF_PROTECTION_ENABLED", "false")

	_, err := LoadConfig()
	if err == nil || !strings.Contains(err.Error(), "CSRF_PROTECTION_ENABLED") {
		t.Fatalf("LoadConfig() error = %v, want CSRF validation", err)
	}
}
