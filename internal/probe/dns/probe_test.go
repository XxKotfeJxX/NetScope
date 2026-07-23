package dns

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

type fakeResolver struct{}

func (fakeResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{
		netip.MustParseAddr("2001:db8::1"),
		netip.MustParseAddr("192.0.2.1"),
	}, nil
}

func (fakeResolver) LookupCNAME(context.Context, string) (string, error) {
	return "example.com.", nil
}

func (fakeResolver) LookupMX(context.Context, string) ([]*net.MX, error) {
	return nil, &net.DNSError{IsNotFound: true}
}

func (fakeResolver) LookupNS(context.Context, string) ([]*net.NS, error) {
	return []*net.NS{{Host: "ns1.example.com."}}, nil
}

func (fakeResolver) LookupTXT(context.Context, string) ([]string, error) {
	return []string{"hello"}, nil
}

func (fakeResolver) LookupAddr(context.Context, string) ([]string, error) {
	return []string{"resolver.example."}, nil
}

func TestProbeHostname(t *testing.T) {
	t.Parallel()

	result := New(fakeResolver{}).Run(
		context.Background(),
		target.Target{Host: "example.com", Kind: target.KindHostname},
		diagnostics.RunOptions{},
	)

	if result.Status != diagnostics.CheckPassed {
		t.Fatalf("status = %q, want %q", result.Status, diagnostics.CheckPassed)
	}
	if result.ErrorCode != "" {
		t.Fatalf("error code = %q, want empty", result.ErrorCode)
	}
}

func TestProbeIPAddressUsesPTR(t *testing.T) {
	t.Parallel()

	result := New(fakeResolver{}).Run(
		context.Background(),
		target.Target{
			Host:    "192.0.2.1",
			Kind:    target.KindIPv4,
			Address: netip.MustParseAddr("192.0.2.1"),
		},
		diagnostics.RunOptions{},
	)

	if result.Status != diagnostics.CheckPassed {
		t.Fatalf("status = %q, want %q", result.Status, diagnostics.CheckPassed)
	}
}
