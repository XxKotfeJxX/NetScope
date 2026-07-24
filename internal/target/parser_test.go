package target

import (
	"errors"
	"net/netip"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Target
		wantErr error
	}{
		{
			name:  "hostname",
			input: " Example.COM ",
			want:  Target{Input: "Example.COM", Kind: KindHostname, Host: "example.com"},
		},
		{
			name:  "unicode hostname",
			input: "münich.example",
			want:  Target{Input: "münich.example", Kind: KindHostname, Host: "xn--mnich-kva.example"},
		},
		{
			name:  "URL normalization",
			input: "https://Example.COM/test#fragment",
			want: Target{
				Input:         "https://Example.COM/test#fragment",
				Kind:          KindURL,
				Host:          "example.com",
				Scheme:        "https",
				Port:          443,
				Path:          "/test",
				NormalizedURL: "https://example.com/test",
			},
		},
		{
			name:  "IPv4 with port",
			input: "1.1.1.1:53",
			want: Target{
				Input:   "1.1.1.1:53",
				Kind:    KindIPv4,
				Host:    "1.1.1.1",
				Port:    53,
				Address: netip.MustParseAddr("1.1.1.1"),
			},
		},
		{
			name:  "bracketed IPv6",
			input: "[2606:4700:4700::1111]",
			want: Target{
				Input:   "[2606:4700:4700::1111]",
				Kind:    KindIPv6,
				Host:    "2606:4700:4700::1111",
				Address: netip.MustParseAddr("2606:4700:4700::1111"),
			},
		},
		{name: "CIDR", input: "10.0.0.0/8", wantErr: ErrCIDR},
		{name: "wildcard", input: "*.example.com", wantErr: ErrWildcard},
		{name: "credentials", input: "https://user:secret@example.com", wantErr: ErrCredentials},
		{name: "bad port", input: "example.com:70000", wantErr: ErrPort},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(test.input)
			if test.wantErr != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("error = %v, want %v", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestPolicy(t *testing.T) {
	t.Parallel()

	public := Policy{Public: true}
	for _, value := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1"} {
		if err := public.ValidateAddress(netip.MustParseAddr(value)); !errors.Is(err, ErrAddressBlocked) {
			t.Errorf("public policy accepted %s", value)
		}
	}
	if err := public.ValidateAddress(netip.MustParseAddr("1.1.1.1")); err != nil {
		t.Errorf("public policy rejected public address: %v", err)
	}

	local := Policy{AllowLoopback: true, AllowPrivate: true}
	if err := local.ValidateAddress(netip.MustParseAddr("127.0.0.1")); err != nil {
		t.Errorf("local policy rejected loopback: %v", err)
	}
}
