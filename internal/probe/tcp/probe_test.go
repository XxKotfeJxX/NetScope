package tcp

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

type localResolver struct {
	address netip.Addr
}

func (r localResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{r.address}, nil
}

func TestProbeConnectsAndClosesSocket(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr == nil {
			_ = connection.Close()
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	dialer := network.NewSecureDialer(
		localResolver{address: netip.MustParseAddr("127.0.0.1")},
		target.Policy{AllowLoopback: true},
		time.Second,
	)
	result := New(dialer).Run(
		context.Background(),
		target.Target{Host: "localhost", Kind: target.KindHostname},
		diagnostics.RunOptions{TCPPorts: []int{port}},
	)

	if result.Status != diagnostics.CheckPassed {
		t.Fatalf("status = %q, want %q", result.Status, diagnostics.CheckPassed)
	}
}
