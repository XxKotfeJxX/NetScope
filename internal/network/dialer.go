package network

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

type SecureDialer struct {
	resolver Resolver
	policy   target.Policy
	timeout  time.Duration
}

func NewSecureDialer(resolver Resolver, policy target.Policy, timeout time.Duration) *SecureDialer {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &SecureDialer{resolver: resolver, policy: policy, timeout: timeout}
}

func (d *SecureDialer) Resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		address = address.Unmap()
		if err := d.policy.ValidateAddress(address); err != nil {
			return nil, err
		}
		return []netip.Addr{address}, nil
	}

	addresses, err := d.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve host: %w", err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve host: no addresses returned")
	}
	for _, address := range addresses {
		if err := d.policy.ValidateAddress(address); err != nil {
			return nil, err
		}
	}
	return addresses, nil
}

func (d *SecureDialer) DialContext(
	ctx context.Context,
	networkName string,
	address string,
) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse network address: %w", err)
	}
	addresses, err := d.Resolve(ctx, host)
	if err != nil {
		return nil, err
	}

	var lastError error
	for _, resolved := range addresses {
		dialer := net.Dialer{Timeout: d.timeout, KeepAlive: 30 * time.Second}
		connection, dialErr := dialer.DialContext(
			ctx,
			networkName,
			net.JoinHostPort(resolved.String(), port),
		)
		if dialErr == nil {
			return connection, nil
		}
		lastError = dialErr
		if ctx.Err() != nil {
			break
		}
	}
	return nil, fmt.Errorf("connect to allowed address on port %s: %w", port, lastError)
}

func AddressFromConnection(connection net.Conn) string {
	host, _, err := net.SplitHostPort(connection.RemoteAddr().String())
	if err != nil {
		return connection.RemoteAddr().String()
	}
	return host
}

func PortAddress(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}
