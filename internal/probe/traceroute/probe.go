package traceroute

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
	return diagnostics.CheckTraceroute
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	options diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	addresses, err := p.resolver.ResolveVersion(ctx, parsedTarget.Host, options.IPVersion)
	if err != nil {
		return failedResult(started, Result{Hops: []Hop{}}, "resolve_failed", err.Error(), ctx)
	}
	data := Result{
		Address: addresses[0].String(), MaxHops: options.MaxHops, Hops: []Hop{},
	}
	samples, err := p.runner.Trace(
		ctx,
		addresses[0],
		options.MaxHops,
		time.Duration(options.TimeoutMS)*time.Millisecond,
	)
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return failedResult(started, data, "icmp_unavailable", err.Error(), ctx)
	}

	responsive := 0
	for _, sample := range samples {
		hop := Hop{Number: sample.Hop, Status: "timeout", Destination: sample.Destination}
		if !sample.Timeout {
			hop.Status = "replied"
			hop.Address = sample.Address.String()
			hop.RTTMS = float64(sample.RTT.Microseconds()) / 1000
			responsive++
		}
		if sample.Destination {
			data.Reached = true
		}
		data.Hops = append(data.Hops, hop)
	}

	status := diagnostics.CheckPassed
	summary := "Route reached the destination"
	errorCode := ""
	errorMessage := ""
	switch {
	case ctx.Err() != nil:
		status = diagnostics.CheckCancelled
		summary = "Traceroute cancelled"
		errorCode = "probe_cancelled"
		errorMessage = ctx.Err().Error()
	case data.Reached:
	case responsive > 0:
		status = diagnostics.CheckWarning
		summary = "Route did not reach the destination within the hop limit"
	default:
		status = diagnostics.CheckFailed
		summary = "No traceroute hops replied"
		errorCode = "trace_no_reply"
		errorMessage = "No router returned an ICMP response."
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
	return result(started, data, status, "Traceroute could not run", code, message)
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
		ID: uuid.New(), Type: diagnostics.CheckTraceroute, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: summary,
		Data: encoded, ErrorCode: errorCode, ErrorMessage: errorMessage,
		StartedAt: started, CompletedAt: completed,
	}
}
