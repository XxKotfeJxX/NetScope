package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	runs       RunService
}

type RunService interface {
	Supports(diagnostics.CheckType) bool
	Create(
		context.Context,
		string,
		[]diagnostics.CheckType,
		diagnostics.RunOptions,
	) (diagnostics.DiagnosticRun, error)
	Get(context.Context, uuid.UUID) (diagnostics.DiagnosticRun, error)
}

func NewService(repository Repository, runs RunService) *Service {
	return &Service{repository: repository, runs: runs}
}

func (s *Service) CreateTarget(
	ctx context.Context,
	input TargetInput,
) (Target, error) {
	normalized, err := s.normalizeInput(input)
	if err != nil {
		return Target{}, err
	}
	now := time.Now().UTC()
	created := Target{
		ID: uuid.New(), Name: normalized.Name, Address: normalized.Address,
		Tags: normalized.Tags, Checks: normalized.Checks, Options: normalized.Options,
		IntervalSeconds: normalized.IntervalSeconds, Enabled: true,
		FailureThreshold: normalized.FailureThreshold, Status: StatusPending,
		NextCheckAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repository.CreateTarget(ctx, created); err != nil {
		return Target{}, err
	}
	return created, nil
}

func (s *Service) GetTarget(ctx context.Context, id uuid.UUID) (Target, error) {
	return s.repository.GetTarget(ctx, id)
}

func (s *Service) ListTargets(
	ctx context.Context,
	page int,
	pageSize int,
) (Page, error) {
	return s.repository.ListTargets(ctx, page, pageSize)
}

func (s *Service) UpdateTarget(
	ctx context.Context,
	id uuid.UUID,
	input TargetInput,
) (Target, error) {
	normalized, err := s.normalizeInput(input)
	if err != nil {
		return Target{}, err
	}
	existing, err := s.repository.GetTarget(ctx, id)
	if err != nil {
		return Target{}, err
	}
	existing.Name = normalized.Name
	existing.Address = normalized.Address
	existing.Tags = normalized.Tags
	existing.Checks = normalized.Checks
	existing.Options = normalized.Options
	existing.IntervalSeconds = normalized.IntervalSeconds
	existing.FailureThreshold = normalized.FailureThreshold
	existing.UpdatedAt = time.Now().UTC()
	if err := s.repository.UpdateTarget(ctx, existing); err != nil {
		return Target{}, err
	}
	return s.repository.GetTarget(ctx, id)
}

func (s *Service) DeleteTarget(ctx context.Context, id uuid.UUID) error {
	return s.repository.DeleteTarget(ctx, id)
}

func (s *Service) SetTargetEnabled(
	ctx context.Context,
	id uuid.UUID,
	enabled bool,
) error {
	return s.repository.SetTargetEnabled(ctx, id, enabled)
}

func (s *Service) ListChecks(
	ctx context.Context,
	targetID uuid.UUID,
	page int,
	pageSize int,
) (CheckPage, error) {
	if _, err := s.repository.GetTarget(ctx, targetID); err != nil {
		return CheckPage{}, err
	}
	return s.repository.ListChecks(ctx, targetID, page, pageSize)
}

func (s *Service) CreateMaintenanceWindow(
	ctx context.Context,
	targetID uuid.UUID,
	startsAt time.Time,
	endsAt time.Time,
	reason string,
) (MaintenanceWindow, error) {
	if !endsAt.After(startsAt) || !endsAt.After(time.Now()) {
		return MaintenanceWindow{}, fmt.Errorf(
			"%w: maintenance end must be after start and in the future",
			ErrInvalidTarget,
		)
	}
	if _, err := s.repository.GetTarget(ctx, targetID); err != nil {
		return MaintenanceWindow{}, err
	}
	window := MaintenanceWindow{
		ID: uuid.New(), TargetID: targetID, StartsAt: startsAt.UTC(),
		EndsAt: endsAt.UTC(), Reason: strings.TrimSpace(reason),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repository.CreateMaintenanceWindow(ctx, window); err != nil {
		return MaintenanceWindow{}, err
	}
	return window, nil
}

func (s *Service) ListMaintenanceWindows(
	ctx context.Context,
	targetID uuid.UUID,
) ([]MaintenanceWindow, error) {
	return s.repository.ListMaintenanceWindows(ctx, targetID)
}

func (s *Service) DeleteMaintenanceWindow(
	ctx context.Context,
	targetID uuid.UUID,
	id uuid.UUID,
) error {
	return s.repository.DeleteMaintenanceWindow(ctx, targetID, id)
}

func (s *Service) CreateNotificationChannel(
	ctx context.Context,
	targetID uuid.UUID,
	kind NotificationKind,
	destination string,
) (NotificationChannel, error) {
	destination = strings.TrimSpace(destination)
	switch kind {
	case NotificationEmail:
		if _, err := mail.ParseAddress(destination); err != nil {
			return NotificationChannel{}, fmt.Errorf(
				"%w: invalid notification email",
				ErrInvalidTarget,
			)
		}
	case NotificationWebhook:
		parsed, err := url.Parse(destination)
		if err != nil || parsed.Host == "" ||
			(parsed.Scheme != "https" && parsed.Scheme != "http") {
			return NotificationChannel{}, fmt.Errorf(
				"%w: invalid webhook URL",
				ErrInvalidTarget,
			)
		}
	default:
		return NotificationChannel{}, fmt.Errorf(
			"%w: notification kind must be email or webhook",
			ErrInvalidTarget,
		)
	}
	if _, err := s.repository.GetTarget(ctx, targetID); err != nil {
		return NotificationChannel{}, err
	}
	channel := NotificationChannel{
		ID: uuid.New(), TargetID: targetID, Kind: kind,
		Destination: destination, Enabled: true, CreatedAt: time.Now().UTC(),
	}
	if err := s.repository.CreateNotificationChannel(ctx, channel); err != nil {
		return NotificationChannel{}, err
	}
	return channel, nil
}

func (s *Service) ListNotificationChannels(
	ctx context.Context,
	targetID uuid.UUID,
) ([]NotificationChannel, error) {
	return s.repository.ListNotificationChannels(ctx, targetID)
}

func (s *Service) DeleteNotificationChannel(
	ctx context.Context,
	targetID uuid.UUID,
	id uuid.UUID,
) error {
	return s.repository.DeleteNotificationChannel(ctx, targetID, id)
}

func (s *Service) normalizeInput(input TargetInput) (TargetInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 100 ||
		strings.IndexFunc(input.Name, unicode.IsControl) >= 0 {
		return TargetInput{}, fmt.Errorf(
			"%w: name must contain 1 to 100 printable characters",
			ErrInvalidTarget,
		)
	}
	parsed, err := target.Parse(input.Address)
	if err != nil {
		return TargetInput{}, err
	}
	input.Address = parsed.Input
	if input.IntervalSeconds == 0 {
		input.IntervalSeconds = 300
	}
	if input.IntervalSeconds < 60 || input.IntervalSeconds > 86400 {
		return TargetInput{}, fmt.Errorf(
			"%w: intervalSeconds must be between 60 and 86400",
			ErrInvalidTarget,
		)
	}
	if input.FailureThreshold == 0 {
		input.FailureThreshold = 3
	}
	if input.FailureThreshold < 1 || input.FailureThreshold > 20 {
		return TargetInput{}, fmt.Errorf(
			"%w: failureThreshold must be between 1 and 20",
			ErrInvalidTarget,
		)
	}
	if len(input.Checks) == 0 {
		input.Checks = []diagnostics.CheckType{
			diagnostics.CheckDNS, diagnostics.CheckTCP,
			diagnostics.CheckTLS, diagnostics.CheckHTTP,
		}
	}
	seenChecks := make(map[diagnostics.CheckType]struct{}, len(input.Checks))
	checks := make([]diagnostics.CheckType, 0, len(input.Checks))
	for _, check := range input.Checks {
		if !s.runs.Supports(check) {
			return TargetInput{}, fmt.Errorf(
				"%w: check %s is unavailable",
				ErrInvalidTarget,
				check,
			)
		}
		if _, exists := seenChecks[check]; !exists {
			seenChecks[check] = struct{}{}
			checks = append(checks, check)
		}
	}
	input.Checks = checks
	if input.Options.TimeoutMS == 0 {
		input.Options.TimeoutMS = 5000
	}
	if input.Options.HTTPMethod == "" {
		input.Options.HTTPMethod = http.MethodGet
	}
	if input.Options.IPVersion == "" {
		input.Options.IPVersion = "auto"
	}
	if input.Options.MaxRedirects == 0 {
		input.Options.MaxRedirects = 5
	}
	if input.Options.PingPackets == 0 {
		input.Options.PingPackets = 4
	}
	if input.Options.MaxHops == 0 {
		input.Options.MaxHops = 20
	}
	tags := make([]string, 0, len(input.Tags))
	seenTags := make(map[string]struct{}, len(input.Tags))
	for _, tag := range input.Tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		if len(tag) > 32 || strings.IndexFunc(tag, unicode.IsControl) >= 0 {
			return TargetInput{}, fmt.Errorf(
				"%w: tags must be at most 32 printable characters",
				ErrInvalidTarget,
			)
		}
		if _, exists := seenTags[tag]; !exists {
			seenTags[tag] = struct{}{}
			tags = append(tags, tag)
		}
	}
	if len(tags) > 10 {
		return TargetInput{}, fmt.Errorf(
			"%w: at most 10 tags are allowed",
			ErrInvalidTarget,
		)
	}
	sort.Strings(tags)
	input.Tags = tags
	return input, nil
}
