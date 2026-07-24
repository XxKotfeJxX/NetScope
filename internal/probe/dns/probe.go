package dns

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
	LookupCNAME(context.Context, string) (string, error)
	LookupMX(context.Context, string) ([]*net.MX, error)
	LookupNS(context.Context, string) ([]*net.NS, error)
	LookupTXT(context.Context, string) ([]string, error)
	LookupAddr(context.Context, string) ([]string, error)
}

type Probe struct {
	resolver Resolver
}

func New(resolver Resolver) *Probe {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &Probe{resolver: resolver}
}

func (p *Probe) Type() diagnostics.CheckType {
	return diagnostics.CheckDNS
}

func (p *Probe) Run(
	ctx context.Context,
	parsedTarget target.Target,
	_ diagnostics.RunOptions,
) diagnostics.CheckResult {
	started := time.Now().UTC()
	data := Result{
		A:      []string{},
		AAAA:   []string{},
		MX:     []MXRecord{},
		NS:     []string{},
		TXT:    []string{},
		PTR:    []string{},
		Errors: make(map[string]RecordError),
	}

	var mutex sync.Mutex
	var waitGroup sync.WaitGroup
	run := func(name string, lookup func() error) {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if err := lookup(); err != nil {
				mutex.Lock()
				data.Errors[name] = classifyError(err)
				mutex.Unlock()
			}
		}()
	}

	if parsedTarget.IsIP() {
		run("ptr", func() error {
			records, err := p.resolver.LookupAddr(ctx, parsedTarget.Host)
			mutex.Lock()
			data.PTR = records
			mutex.Unlock()
			return err
		})
	} else {
		run("a_aaaa", func() error {
			addresses, err := p.resolver.LookupNetIP(ctx, "ip", parsedTarget.Host)
			mutex.Lock()
			for _, address := range addresses {
				if address.Is4() {
					data.A = append(data.A, address.String())
				} else {
					data.AAAA = append(data.AAAA, address.String())
				}
			}
			sort.Strings(data.A)
			sort.Strings(data.AAAA)
			mutex.Unlock()
			return err
		})
		run("cname", func() error {
			record, err := p.resolver.LookupCNAME(ctx, parsedTarget.Host)
			mutex.Lock()
			data.CNAME = record
			mutex.Unlock()
			return err
		})
		run("mx", func() error {
			records, err := p.resolver.LookupMX(ctx, parsedTarget.Host)
			mutex.Lock()
			for _, record := range records {
				data.MX = append(data.MX, MXRecord{Host: record.Host, Preference: record.Pref})
			}
			mutex.Unlock()
			return err
		})
		run("ns", func() error {
			records, err := p.resolver.LookupNS(ctx, parsedTarget.Host)
			mutex.Lock()
			for _, record := range records {
				data.NS = append(data.NS, record.Host)
			}
			mutex.Unlock()
			return err
		})
		run("txt", func() error {
			records, err := p.resolver.LookupTXT(ctx, parsedTarget.Host)
			mutex.Lock()
			data.TXT = records
			mutex.Unlock()
			return err
		})
	}
	waitGroup.Wait()

	completed := time.Now().UTC()
	status := diagnostics.CheckPassed
	errorCode := ""
	errorMessage := ""
	summary := "DNS records resolved"
	if ctx.Err() != nil {
		status = diagnostics.CheckCancelled
		errorCode = "probe_cancelled"
		errorMessage = "DNS lookup was cancelled."
		summary = "DNS lookup cancelled"
	} else if len(data.A)+len(data.AAAA)+len(data.PTR) == 0 {
		status = diagnostics.CheckFailed
		errorCode = "dns_not_found"
		errorMessage = "No address records were returned for this target."
		summary = "No address records found"
	}

	encoded, _ := json.Marshal(data)
	return diagnostics.CheckResult{
		ID:           uuid.New(),
		Type:         diagnostics.CheckDNS,
		Status:       status,
		DurationMS:   completed.Sub(started).Milliseconds(),
		Summary:      summary,
		Data:         encoded,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		StartedAt:    started,
		CompletedAt:  completed,
	}
}

func classifyError(err error) RecordError {
	if errors.Is(err, context.DeadlineExceeded) {
		return RecordError{Code: "probe_timeout", Message: "The DNS lookup timed out."}
	}
	if errors.Is(err, context.Canceled) {
		return RecordError{Code: "probe_cancelled", Message: "The DNS lookup was cancelled."}
	}
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return RecordError{Code: "dns_not_found", Message: "No records of this type were found."}
	}
	if strings.TrimSpace(err.Error()) == "" {
		return RecordError{Code: "dns_error", Message: "The DNS lookup failed."}
	}
	return RecordError{Code: "dns_error", Message: "The DNS resolver returned an error."}
}
