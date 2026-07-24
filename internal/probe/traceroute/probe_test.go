package traceroute

import (
	"context"
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
)

type fixedResolver struct {
	address netip.Addr
}

func (r fixedResolver) ResolveVersion(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{r.address}, nil
}

type fixedRunner struct {
	hops []network.TraceSample
}

func (fixedRunner) Ping(
	context.Context,
	netip.Addr,
	int,
	time.Duration,
) ([]network.EchoSample, error) {
	return nil, nil
}

func (r fixedRunner) Trace(
	context.Context,
	netip.Addr,
	int,
	time.Duration,
) ([]network.TraceSample, error) {
	return r.hops, nil
}

func TestProbeBuildsRouteJournal(t *testing.T) {
	t.Parallel()

	destination := netip.MustParseAddr("192.0.2.10")
	probe := New(fixedResolver{address: destination}, fixedRunner{hops: []network.TraceSample{
		{Hop: 1, Address: netip.MustParseAddr("192.0.2.1"), RTT: 2 * time.Millisecond},
		{Hop: 2, Timeout: true},
		{Hop: 3, Address: destination, RTT: 18 * time.Millisecond, Destination: true},
	}})
	result := probe.Run(context.Background(), target.Target{Host: "example.com"}, diagnostics.RunOptions{
		TimeoutMS: 5000, MaxHops: 20, IPVersion: "auto",
	})
	if result.Status != diagnostics.CheckPassed {
		t.Fatalf("status = %s, want passed", result.Status)
	}
	var data Result
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !data.Reached || len(data.Hops) != 3 || data.Hops[1].Status != "timeout" {
		t.Fatalf("unexpected traceroute result: %+v", data)
	}
}
