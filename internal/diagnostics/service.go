package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strings"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

var (
	ErrUnsupportedCheck = errors.New("unsupported diagnostic check")
	ErrInvalidOptions   = errors.New("invalid diagnostic options")
)

type Service struct {
	repository     RunRepository
	manager        *Manager
	policy         target.Policy
	resolver       *net.Resolver
	defaultTimeout time.Duration
	maxTimeout     time.Duration
}

func NewService(
	repository RunRepository,
	manager *Manager,
	policy target.Policy,
	defaultTimeout time.Duration,
	maxTimeout time.Duration,
) *Service {
	return &Service{
		repository: repository, manager: manager, policy: policy,
		resolver: net.DefaultResolver, defaultTimeout: defaultTimeout, maxTimeout: maxTimeout,
	}
}

func (s *Service) Create(
	ctx context.Context,
	input string,
	checks []CheckType,
	options RunOptions,
) (DiagnosticRun, error) {
	parsed, err := target.Parse(input)
	if err != nil {
		return DiagnosticRun{}, err
	}
	if len(checks) == 0 {
		checks = []CheckType{CheckDNS}
	}
	uniqueChecks := make([]CheckType, 0, len(checks))
	seenChecks := make(map[CheckType]struct{}, len(checks))
	for _, check := range checks {
		switch check {
		case CheckDNS, CheckTCP, CheckHTTP, CheckTLS:
		default:
			return DiagnosticRun{}, ErrUnsupportedCheck
		}
		if _, exists := seenChecks[check]; !exists {
			seenChecks[check] = struct{}{}
			uniqueChecks = append(uniqueChecks, check)
		}
	}
	checks = uniqueChecks
	if err := s.validateTarget(ctx, parsed); err != nil {
		return DiagnosticRun{}, err
	}
	if options.TimeoutMS == 0 {
		options.TimeoutMS = int(s.defaultTimeout.Milliseconds())
	}
	if options.TimeoutMS < 500 || time.Duration(options.TimeoutMS)*time.Millisecond > s.maxTimeout {
		return DiagnosticRun{}, fmt.Errorf("timeout must be between 500ms and %s", s.maxTimeout)
	}
	if options.IPVersion == "" {
		options.IPVersion = "auto"
	}
	if options.IPVersion != "auto" && options.IPVersion != "ipv4" && options.IPVersion != "ipv6" {
		return DiagnosticRun{}, fmt.Errorf("%w: ipVersion must be auto, ipv4, or ipv6", ErrInvalidOptions)
	}
	if options.HTTPMethod == "" {
		options.HTTPMethod = http.MethodGet
	}
	options.HTTPMethod = strings.ToUpper(options.HTTPMethod)
	if options.HTTPMethod != http.MethodGet && options.HTTPMethod != http.MethodHead {
		return DiagnosticRun{}, fmt.Errorf("%w: HTTP method must be GET or HEAD", ErrInvalidOptions)
	}
	if options.MaxRedirects == 0 {
		options.MaxRedirects = 5
	}
	if options.MaxRedirects < 0 || options.MaxRedirects > 10 {
		return DiagnosticRun{}, fmt.Errorf("%w: maxRedirects must be between 0 and 10", ErrInvalidOptions)
	}
	if _, requested := seenChecks[CheckTCP]; requested {
		if len(options.TCPPorts) == 0 {
			options.TCPPorts = inferredPorts(parsed)
		}
		ports, err := normalizePorts(options.TCPPorts)
		if err != nil {
			return DiagnosticRun{}, err
		}
		options.TCPPorts = ports
	}

	run := DiagnosticRun{
		ID:              uuid.New(),
		TargetInput:     parsed.Input,
		NormalizedHost:  parsed.Host,
		NormalizedURL:   parsed.NormalizedURL,
		Target:          parsed,
		Status:          RunQueued,
		RequestedChecks: checks,
		Options:         options,
		Results:         []CheckResult{},
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.repository.Create(ctx, run); err != nil {
		return DiagnosticRun{}, err
	}
	if err := s.manager.Enqueue(run.ID); err != nil {
		_ = s.repository.UpdateStatus(ctx, run.ID, RunFailed)
		return DiagnosticRun{}, err
	}
	return run, nil
}

func inferredPorts(parsed target.Target) []int {
	if parsed.Port > 0 {
		return []int{parsed.Port}
	}
	if parsed.Scheme == "http" {
		return []int{80}
	}
	return []int{443}
}

func normalizePorts(ports []int) ([]int, error) {
	if len(ports) > 10 {
		return nil, fmt.Errorf("%w: at most 10 TCP ports are allowed", ErrInvalidOptions)
	}
	seen := make(map[int]struct{}, len(ports))
	normalized := make([]int, 0, len(ports))
	for _, port := range ports {
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("%w: TCP ports must be between 1 and 65535", ErrInvalidOptions)
		}
		if _, exists := seen[port]; !exists {
			seen[port] = struct{}{}
			normalized = append(normalized, port)
		}
	}
	sort.Ints(normalized)
	return normalized, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (DiagnosticRun, error) {
	run, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return DiagnosticRun{}, err
	}
	parsed, parseErr := target.Parse(run.TargetInput)
	if parseErr == nil {
		run.Target = parsed
	}
	return run, nil
}

func (s *Service) List(ctx context.Context, filter ListFilter) (Page, error) {
	return s.repository.List(ctx, filter)
}

func (s *Service) Cancel(ctx context.Context, id uuid.UUID) error {
	return s.manager.Cancel(ctx, id)
}

func (s *Service) validateTarget(ctx context.Context, parsed target.Target) error {
	if parsed.IsIP() {
		return s.policy.ValidateAddress(parsed.Address)
	}
	resolveContext, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	addresses, err := s.resolver.LookupNetIP(resolveContext, "ip", parsed.Host)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	if len(addresses) == 0 {
		return fmt.Errorf("resolve target: no addresses returned")
	}
	for _, address := range addresses {
		if err := s.policy.ValidateAddress(netip.Addr(address)); err != nil {
			return err
		}
	}
	return nil
}
