package ping

import (
	"context"
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
	samples []network.EchoSample
}

func (r fixedRunner) Ping(context.Context, netip.Addr, int, time.Duration) ([]network.EchoSample, error) {
	return r.samples, nil
}

func (fixedRunner) Trace(
	context.Context,
	netip.Addr,
	int,
	time.Duration,
) ([]network.TraceSample, error) {
	return nil, nil
}

func TestProbeSummarizesPacketLoss(t *testing.T) {
	t.Parallel()

	address := netip.MustParseAddr("192.0.2.1")
	probe := New(fixedResolver{address: address}, fixedRunner{samples: []network.EchoSample{
		{Sequence: 1, Address: address, RTT: 20 * time.Millisecond},
		{Sequence: 2, Timeout: true},
		{Sequence: 3, Address: address, RTT: 40 * time.Millisecond},
		{Sequence: 4, Address: address, RTT: 30 * time.Millisecond},
	}})
	result := probe.Run(context.Background(), target.Target{Host: "example.com"}, diagnostics.RunOptions{
		TimeoutMS: 5000, PingPackets: 4, IPVersion: "auto",
	})
	if result.Status != diagnostics.CheckWarning {
		t.Fatalf("status = %s, want warning", result.Status)
	}
	var data Result
	if err := resultData(result.Data, &data); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if data.PacketsReceived != 3 || data.PacketLossPercent != 25 || data.AverageRTTMS != 30 {
		t.Fatalf("unexpected ping result: %+v", data)
	}
}
