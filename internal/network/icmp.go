package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type EchoSample struct {
	Sequence int
	Address  netip.Addr
	RTT      time.Duration
	Timeout  bool
}

type TraceSample struct {
	Hop         int
	Address     netip.Addr
	RTT         time.Duration
	Timeout     bool
	Destination bool
}

type ICMPRunner interface {
	Ping(context.Context, netip.Addr, int, time.Duration) ([]EchoSample, error)
	Trace(context.Context, netip.Addr, int, time.Duration) ([]TraceSample, error)
}

type RawICMPRunner struct{}

var icmpRunSequence atomic.Uint32

func (RawICMPRunner) Available() (bool, string) {
	for _, candidate := range []netip.Addr{
		netip.MustParseAddr("127.0.0.1"),
		netip.MustParseAddr("::1"),
	} {
		connection, _, err := listenICMP(candidate)
		if err == nil {
			_ = connection.Close()
			return true, ""
		}
	}
	return false, "raw_icmp_unavailable"
}

func (RawICMPRunner) Ping(
	ctx context.Context,
	address netip.Addr,
	count int,
	timeout time.Duration,
) ([]EchoSample, error) {
	connection, protocol, err := listenICMP(address)
	if err != nil {
		return nil, fmt.Errorf("open ICMP socket: %w", err)
	}
	defer func() { _ = connection.Close() }()

	identifier := nextICMPIdentifier()
	perAttempt := boundedAttemptTimeout(timeout, count)
	samples := make([]EchoSample, 0, count)
	for sequence := 1; sequence <= count; sequence++ {
		if err := ctx.Err(); err != nil {
			return samples, err
		}
		started := time.Now()
		if err := sendEcho(connection, address, identifier, sequence); err != nil {
			return samples, fmt.Errorf("send ICMP echo: %w", err)
		}
		peer, readErr := readICMPReply(
			ctx,
			connection,
			protocol,
			identifier,
			sequence,
			perAttempt,
			false,
		)
		sample := EchoSample{Sequence: sequence, RTT: time.Since(started)}
		switch {
		case readErr == nil:
			sample.Address = peer
		case isTimeout(readErr):
			sample.Timeout = true
		default:
			return samples, readErr
		}
		samples = append(samples, sample)

		if sequence < count {
			timer := time.NewTimer(150 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return samples, ctx.Err()
			case <-timer.C:
			}
		}
	}
	return samples, nil
}

func (RawICMPRunner) Trace(
	ctx context.Context,
	address netip.Addr,
	maxHops int,
	timeout time.Duration,
) ([]TraceSample, error) {
	connection, protocol, err := listenICMP(address)
	if err != nil {
		return nil, fmt.Errorf("open ICMP socket: %w", err)
	}
	defer func() { _ = connection.Close() }()

	identifier := nextICMPIdentifier()
	perHop := boundedAttemptTimeout(timeout, maxHops)
	hops := make([]TraceSample, 0, maxHops)
	for hop := 1; hop <= maxHops; hop++ {
		if err := ctx.Err(); err != nil {
			return hops, err
		}
		if err := setICMPHopLimit(connection, address, hop); err != nil {
			return hops, fmt.Errorf("set ICMP hop limit: %w", err)
		}
		started := time.Now()
		if err := sendEcho(connection, address, identifier, hop); err != nil {
			return hops, fmt.Errorf("send ICMP trace probe: %w", err)
		}
		peer, readErr := readICMPReply(
			ctx,
			connection,
			protocol,
			identifier,
			hop,
			perHop,
			true,
		)
		sample := TraceSample{Hop: hop, RTT: time.Since(started)}
		switch {
		case readErr == nil:
			sample.Address = peer
			sample.Destination = peer == address
		case isTimeout(readErr):
			sample.Timeout = true
		default:
			return hops, readErr
		}
		hops = append(hops, sample)
		if sample.Destination {
			break
		}
	}
	return hops, nil
}

func listenICMP(address netip.Addr) (*icmp.PacketConn, int, error) {
	if address.Is4() {
		connection, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		return connection, 1, err
	}
	connection, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	return connection, 58, err
}

func sendEcho(
	connection *icmp.PacketConn,
	address netip.Addr,
	identifier int,
	sequence int,
) error {
	messageType := icmp.Type(ipv4.ICMPTypeEcho)
	if address.Is6() {
		messageType = ipv6.ICMPTypeEchoRequest
	}
	payload, err := (&icmp.Message{
		Type: messageType,
		Code: 0,
		Body: &icmp.Echo{
			ID: identifier, Seq: sequence, Data: []byte("NETSCOPE"),
		},
	}).Marshal(nil)
	if err != nil {
		return err
	}
	_, err = connection.WriteTo(payload, &net.IPAddr{IP: net.IP(address.AsSlice())})
	return err
}

func setICMPHopLimit(connection *icmp.PacketConn, address netip.Addr, hop int) error {
	if address.Is4() {
		return connection.IPv4PacketConn().SetTTL(hop)
	}
	return connection.IPv6PacketConn().SetHopLimit(hop)
}

func readICMPReply(
	ctx context.Context,
	connection *icmp.PacketConn,
	protocol int,
	identifier int,
	sequence int,
	timeout time.Duration,
	acceptTimeExceeded bool,
) (netip.Addr, error) {
	deadline := time.Now().Add(timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := connection.SetReadDeadline(deadline); err != nil {
		return netip.Addr{}, err
	}

	buffer := make([]byte, 1500)
	for {
		count, peer, err := connection.ReadFrom(buffer)
		if err != nil {
			if ctx.Err() != nil {
				return netip.Addr{}, ctx.Err()
			}
			return netip.Addr{}, err
		}
		message, err := icmp.ParseMessage(protocol, buffer[:count])
		if err != nil {
			continue
		}
		if isMatchingEchoReply(message, protocol, identifier, sequence) ||
			(acceptTimeExceeded && isMatchingTimeExceeded(message, protocol, identifier, sequence)) {
			return addressFromNetAddr(peer)
		}
	}
}

func isMatchingEchoReply(
	message *icmp.Message,
	protocol int,
	identifier int,
	sequence int,
) bool {
	if protocol == 1 && message.Type != ipv4.ICMPTypeEchoReply {
		return false
	}
	if protocol == 58 && message.Type != ipv6.ICMPTypeEchoReply {
		return false
	}
	echo, ok := message.Body.(*icmp.Echo)
	return ok && echo.ID == identifier && echo.Seq == sequence
}

func isMatchingTimeExceeded(
	message *icmp.Message,
	protocol int,
	identifier int,
	sequence int,
) bool {
	if protocol == 1 && message.Type != ipv4.ICMPTypeTimeExceeded {
		return false
	}
	if protocol == 58 && message.Type != ipv6.ICMPTypeTimeExceeded {
		return false
	}
	body, ok := message.Body.(*icmp.TimeExceeded)
	if !ok {
		return false
	}
	offset := 40
	if protocol == 1 {
		if len(body.Data) < 20 {
			return false
		}
		offset = int(body.Data[0]&0x0f) * 4
	}
	if offset < 0 || len(body.Data) <= offset {
		return false
	}
	inner, err := icmp.ParseMessage(protocol, body.Data[offset:])
	return err == nil && isMatchingEchoRequest(inner, protocol, identifier, sequence)
}

func isMatchingEchoRequest(
	message *icmp.Message,
	protocol int,
	identifier int,
	sequence int,
) bool {
	if protocol == 1 && message.Type != ipv4.ICMPTypeEcho {
		return false
	}
	if protocol == 58 && message.Type != ipv6.ICMPTypeEchoRequest {
		return false
	}
	echo, ok := message.Body.(*icmp.Echo)
	return ok && echo.ID == identifier && echo.Seq == sequence
}

func addressFromNetAddr(address net.Addr) (netip.Addr, error) {
	switch value := address.(type) {
	case *net.IPAddr:
		result, ok := netip.AddrFromSlice(value.IP)
		if ok {
			return result.Unmap(), nil
		}
	case *net.UDPAddr:
		result, ok := netip.AddrFromSlice(value.IP)
		if ok {
			return result.Unmap(), nil
		}
	}
	return netip.Addr{}, fmt.Errorf("unsupported ICMP peer address %q", address.String())
}

func boundedAttemptTimeout(total time.Duration, attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	value := total / time.Duration(attempts)
	if value < 100*time.Millisecond {
		return 100 * time.Millisecond
	}
	if value > 2*time.Second {
		return 2 * time.Second
	}
	return value
}

func nextICMPIdentifier() int {
	return (os.Getpid() + int(icmpRunSequence.Add(1))) & 0xffff
}

func isTimeout(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}
