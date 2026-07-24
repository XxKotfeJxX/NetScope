package ping

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type resolver interface {
	ResolveVersion(context.Context, string, string) ([]netip.Addr, error)
}

type Probe struct {
	resolver resolver
	runner   network.ICMPRunner
}

func New(resolver resolver, runner network.ICMPRunner) *Probe {
	return &Probe{resolver: resolver, runner: runner}
}

func (p *Probe) Type() diagnostics.CheckType {
	return diagnostics.CheckPing
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	options diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	addresses, err := p.resolver.ResolveVersion(ctx, parsedTarget.Host, options.IPVersion)
	if err != nil {
		return failedResult(started, Result{Samples: []Sample{}}, "resolve_failed", err.Error(), ctx)
	}
	data := Result{
		Address: addresses[0].String(), PacketsSent: options.PingPackets, Samples: []Sample{},
	}
	samples, err := p.runner.Ping(
		ctx,
		addresses[0],
		options.PingPackets,
		time.Duration(options.TimeoutMS)*time.Millisecond,
	)
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return failedResult(started, data, "icmp_unavailable", err.Error(), ctx)
	}

	var totalRTT float64
	for _, sample := range samples {
		item := Sample{Sequence: sample.Sequence, Status: "timeout"}
		if !sample.Timeout {
			item.Status = "received"
			item.Address = sample.Address.String()
			item.RTTMS = milliseconds(sample.RTT)
			data.PacketsReceived++
			totalRTT += item.RTTMS
			if data.PacketsReceived == 1 || item.RTTMS < data.MinRTTMS {
				data.MinRTTMS = item.RTTMS
			}
			if item.RTTMS > data.MaxRTTMS {
				data.MaxRTTMS = item.RTTMS
			}
		}
		data.Samples = append(data.Samples, item)
	}
	if data.PacketsSent > 0 {
		data.PacketLossPercent = float64(data.PacketsSent-data.PacketsReceived) /
			float64(data.PacketsSent) * 100
	}
	if data.PacketsReceived > 0 {
		data.AverageRTTMS = totalRTT / float64(data.PacketsReceived)
	}

	status := diagnostics.CheckPassed
	summary := "All ICMP echo requests received a reply"
	errorCode := ""
	errorMessage := ""
	switch {
	case ctx.Err() != nil:
		status = diagnostics.CheckCancelled
		summary = "Ping cancelled"
		errorCode = "probe_cancelled"
		errorMessage = ctx.Err().Error()
	case data.PacketsReceived == 0:
		status = diagnostics.CheckFailed
		summary = "No ICMP echo replies received"
		errorCode = "icmp_no_reply"
		errorMessage = "The target did not return an ICMP echo reply."
	case data.PacketsReceived < data.PacketsSent:
		status = diagnostics.CheckWarning
		summary = "Some ICMP echo requests timed out"
	}
	return result(started, data, status, summary, errorCode, errorMessage)
}

func failedResult(
	started time.Time,
	data Result,
	code string,
	message string,
	ctx context.Context,
) diagnostics.CheckResult {
	status := diagnostics.CheckFailed
	if ctx.Err() != nil {
		status = diagnostics.CheckCancelled
		code = "probe_cancelled"
		message = ctx.Err().Error()
	}
	return result(started, data, status, "Ping could not run", code, message)
}

func result(
	started time.Time,
	data Result,
	status diagnostics.CheckStatus,
	summary string,
	errorCode string,
	errorMessage string,
) diagnostics.CheckResult {
	completed := time.Now().UTC()
	encoded, _ := json.Marshal(data)
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckPing, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: summary,
		Data: encoded, ErrorCode: errorCode, ErrorMessage: errorMessage,
		StartedAt: started, CompletedAt: completed,
	}
}

func milliseconds(duration time.Duration) float64 {
	return float64(duration.Microseconds()) / 1000
}
