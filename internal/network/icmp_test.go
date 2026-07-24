package network

import (
	"net"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

func TestMatchingEchoReply(t *testing.T) {
	t.Parallel()

	message := &icmp.Message{
		Type: ipv4.ICMPTypeEchoReply,
		Body: &icmp.Echo{ID: 41, Seq: 7},
	}
	if !isMatchingEchoReply(message, 1, 41, 7) {
		t.Fatal("isMatchingEchoReply() = false, want true")
	}
	if isMatchingEchoReply(message, 1, 42, 7) {
		t.Fatal("isMatchingEchoReply() accepted another identifier")
	}
}

func TestAddressFromNetAddr(t *testing.T) {
	t.Parallel()

	address, err := addressFromNetAddr(&net.IPAddr{IP: net.ParseIP("192.0.2.1")})
	if err != nil {
		t.Fatalf("addressFromNetAddr() error = %v", err)
	}
	if address.String() != "192.0.2.1" {
		t.Fatalf("addressFromNetAddr() = %s, want 192.0.2.1", address)
	}
}

func TestBoundedAttemptTimeout(t *testing.T) {
	t.Parallel()

	if got := boundedAttemptTimeout(time.Second, 20); got != 100*time.Millisecond {
		t.Fatalf("boundedAttemptTimeout() = %s, want 100ms", got)
	}
	if got := boundedAttemptTimeout(10*time.Second, 2); got != 2*time.Second {
		t.Fatalf("boundedAttemptTimeout() = %s, want 2s", got)
	}
}
