package tcp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"sort"
	"syscall"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/network"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type Probe struct {
	dialer *network.SecureDialer
}

func New(dialer *network.SecureDialer) *Probe {
	return &Probe{dialer: dialer}
}

func (p *Probe) Type() diagnostics.CheckType {
	return diagnostics.CheckTCP
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	options diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	ports := append([]int(nil), options.TCPPorts...)
	resultChannel := make(chan PortResult, len(ports))
	semaphore := make(chan struct{}, 4)

	for _, port := range ports {
		go func(port int) {
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultChannel <- cancelledResult(port)
				return
			}
			resultChannel <- p.checkPort(ctx, parsedTarget.Host, port)
		}(port)
	}

	data := Result{Ports: make([]PortResult, 0, len(ports))}
	passed := 0
	for range ports {
		portResult := <-resultChannel
		if portResult.Status == "passed" {
			passed++
		}
		data.Ports = append(data.Ports, portResult)
	}
	sort.Slice(data.Ports, func(i, j int) bool { return data.Ports[i].Port < data.Ports[j].Port })

	completed := time.Now().UTC()
	status := diagnostics.CheckFailed
	summary := "No TCP ports accepted a connection"
	switch {
	case ctx.Err() != nil:
		status = diagnostics.CheckCancelled
		summary = "TCP checks cancelled"
	case passed == len(ports) && passed > 0:
		status = diagnostics.CheckPassed
		summary = "All TCP connections succeeded"
	case passed > 0:
		status = diagnostics.CheckWarning
		summary = "Some TCP connections failed"
	}

	encoded, _ := json.Marshal(data)
	return diagnostics.CheckResult{
		ID: uuid.New(), Type: diagnostics.CheckTCP, Status: status,
		DurationMS: completed.Sub(started).Milliseconds(), Summary: summary,
		Data: encoded, StartedAt: started, CompletedAt: completed,
	}
}

func (p *Probe) checkPort(ctx context.Context, host string, port int) PortResult {
	started := time.Now()
	connection, err := p.dialer.DialContext(ctx, "tcp", network.PortAddress(host, port))
	duration := time.Since(started).Milliseconds()
	if err != nil {
		code, message := classifyError(ctx, err)
		return PortResult{
			Port: port, Status: "failed", ConnectTimeMS: duration,
			ErrorCode: code, ErrorMessage: message,
		}
	}
	resolvedIP := network.AddressFromConnection(connection)
	_ = connection.Close()
	return PortResult{
		Port: port, Status: "passed", ResolvedIP: resolvedIP, ConnectTimeMS: duration,
	}
}

func cancelledResult(port int) PortResult {
	return PortResult{
		Port: port, Status: "cancelled", ErrorCode: "probe_cancelled",
		ErrorMessage: "The TCP check was cancelled.",
	}
}

func classifyError(ctx context.Context, err error) (string, string) {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded), errors.Is(err, context.DeadlineExceeded):
		return "timeout", "The TCP connection timed out."
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return "probe_cancelled", "The TCP check was cancelled."
	case errors.Is(err, syscall.ECONNREFUSED):
		return "connection_refused", "The target refused the TCP connection."
	case errors.Is(err, syscall.ENETUNREACH):
		return "network_unreachable", "The target network is unreachable."
	case errors.Is(err, syscall.EHOSTUNREACH):
		return "host_unreachable", "The target host is unreachable."
	case errors.Is(err, os.ErrPermission):
		return "permission_denied", "The operating system denied the connection."
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return "timeout", "The TCP connection timed out."
	}
	return "unknown_error", "The TCP connection could not be established."
}
