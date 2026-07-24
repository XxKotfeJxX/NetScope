package tlscheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

func TestProbeInspectsUntrustedCertificate(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	parsed, err := target.Parse(server.URL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	dialer := network.NewSecureDialer(nil, target.Policy{AllowLoopback: true}, time.Second)

	result := New(dialer).Run(context.Background(), parsed, diagnostics.RunOptions{})

	if result.Status != diagnostics.CheckFailed {
		t.Fatalf("status = %q, want %q", result.Status, diagnostics.CheckFailed)
	}
	var data Result
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if data.Subject == "" || len(data.Chain) == 0 {
		t.Fatalf("certificate details were not captured: %#v", data)
	}
	if data.ChainValid {
		t.Fatal("self-signed test certificate was marked trusted")
	}
}
