package httpcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

func TestProbeAgainstLocalServer(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Set-Cookie", "secret=value")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	parsed, err := target.Parse(server.URL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	dialer := network.NewSecureDialer(
		nil,
		target.Policy{AllowLoopback: true},
		time.Second,
	)
	result := New(dialer).Run(
		context.Background(),
		parsed,
		diagnostics.RunOptions{
			TimeoutMS: 2000, HTTPMethod: http.MethodGet,
			FollowRedirects: true, MaxRedirects: 2,
		},
	)
	if result.Status != diagnostics.CheckPassed {
		t.Fatalf("status = %q, error = %s", result.Status, result.ErrorMessage)
	}
}

func TestTargetURLBracketsIPv6(t *testing.T) {
	t.Parallel()

	got := targetURL(target.Target{
		Host: "2001:db8::1", Address: netip.MustParseAddr("2001:db8::1"),
	})
	if got != "https://[2001:db8::1]" {
		t.Fatalf("targetURL() = %q", got)
	}
}
