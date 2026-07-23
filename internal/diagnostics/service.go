package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

var ErrUnsupportedCheck = errors.New("unsupported diagnostic check")

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
	for _, check := range checks {
		if check != CheckDNS {
			return DiagnosticRun{}, ErrUnsupportedCheck
		}
	}
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
