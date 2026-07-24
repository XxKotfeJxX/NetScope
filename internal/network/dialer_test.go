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

func TestResolveVersionFiltersAddresses(t *testing.T) {
	t.Parallel()

	dialer := NewSecureDialer(
		fixedResolver{addresses: []netip.Addr{
			netip.MustParseAddr("2001:db8::1"),
			netip.MustParseAddr("192.0.2.1"),
		}},
		target.Policy{},
		time.Second,
	)

	addresses, err := dialer.ResolveVersion(context.Background(), "example.com", "ipv4")
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if len(addresses) != 1 || addresses[0].String() != "192.0.2.1" {
		t.Fatalf("ResolveVersion() = %v, want [192.0.2.1]", addresses)
	}
}

func TestResolveVersionRejectsUnavailableFamily(t *testing.T) {
	t.Parallel()

	dialer := NewSecureDialer(
		fixedResolver{addresses: []netip.Addr{netip.MustParseAddr("192.0.2.1")}},
		target.Policy{},
		time.Second,
	)

	_, err := dialer.ResolveVersion(context.Background(), "example.com", "ipv6")
	if !errors.Is(err, ErrIPVersionUnavailable) {
		t.Fatalf("ResolveVersion() error = %v, want ErrIPVersionUnavailable", err)
	}
}

func TestWithIPVersionAppliesToDialContext(t *testing.T) {
	t.Parallel()

	dialer := NewSecureDialer(
		fixedResolver{addresses: []netip.Addr{netip.MustParseAddr("192.0.2.1")}},
		target.Policy{},
		10*time.Millisecond,
	)

	ctx := WithIPVersion(context.Background(), "ipv6")
	_, err := dialer.DialContext(ctx, "tcp", "example.com:443")
	if !errors.Is(err, ErrIPVersionUnavailable) {
		t.Fatalf("DialContext() error = %v, want ErrIPVersionUnavailable", err)
	}
}
