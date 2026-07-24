package network

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
)

type fixedResolver struct {
	addresses []netip.Addr
}

func (r fixedResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return r.addresses, nil
}

func TestResolveAppliesPolicyToEveryAddress(t *testing.T) {
	t.Parallel()

	dialer := NewSecureDialer(
		fixedResolver{addresses: []netip.Addr{
			netip.MustParseAddr("1.1.1.1"),
			netip.MustParseAddr("127.0.0.1"),
		}},
		target.Policy{Public: true},
		time.Second,
	)

	_, err := dialer.Resolve(context.Background(), "example.com")
	if !errors.Is(err, target.ErrAddressBlocked) {
		t.Fatalf("Resolve() error = %v, want ErrAddressBlocked", err)
	}
}
