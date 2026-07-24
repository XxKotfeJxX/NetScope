package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
)

type Alert struct {
	Target  Target       `json:"target"`
	Status  TargetStatus `json:"status"`
	Message string       `json:"message"`
	SentAt  time.Time    `json:"sentAt"`
}

type Notifier interface {
	Notify(context.Context, Alert) error
}

type NopNotifier struct{}

func (NopNotifier) Notify(context.Context, Alert) error {
	return nil
}

type Scheduler struct {
	repository Repository
	runs       RunService
	notifier   Notifier
	logger     *slog.Logger
	interval   time.Duration
	cancel     context.CancelFunc
	waitGroup  sync.WaitGroup
}

func NewScheduler(
	repository Repository,
	runs RunService,
	notifier Notifier,
	logger *slog.Logger,
	interval time.Duration,
) *Scheduler {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	if notifier == nil {
		notifier = NopNotifier{}
	}
	return &Scheduler{
		repository: repository, runs: runs, notifier: notifier,
		logger: logger, interval: interval,
	}
}

func (s *Scheduler) Start() {
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.waitGroup.Add(1)
	go s.loop(ctx)
}

func (s *Scheduler) Shutdown(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}
	s.cancel()
	done := make(chan struct{})
	go func() {
		s.waitGroup.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	defer s.waitGroup.Done()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.cycle(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cycle(ctx)
		}
	}
}

func (s *Scheduler) cycle(ctx context.Context) {
	if err := s.reconcile(ctx); err != nil {
		s.logger.Error("reconcile monitoring checks", "error", err)
	}
	if err := s.dispatch(ctx); err != nil {
		s.logger.Error("dispatch monitoring checks", "error", err)
	}
}

func (s *Scheduler) dispatch(ctx context.Context) error {
	targets, err := s.repository.ClaimDueTargets(ctx, 20)
	if err != nil {
		return err
	}
	for _, target := range targets {
		run, err := s.runs.Create(ctx, target.Address, target.Checks, target.Options)
		if err != nil {
			updated, notify, recordErr := s.repository.RecordDispatchFailure(
				ctx,
				target.ID,
				err.Error(),
			)
			if recordErr != nil {
				s.logger.Error(
					"record monitoring dispatch failure",
					"target_id", target.ID,
					"error", recordErr,
				)
				continue
			}
			if notify {
				s.publishAlert(ctx, updated, "Monitoring checks could not be started")
			}
			continue
		}
		if err := s.repository.CreateScheduledCheck(ctx, target.ID, run.ID); err != nil {
			s.logger.Error(
				"link scheduled diagnostic run",
				"target_id", target.ID,
				"run_id", run.ID,
				"error", err,
			)
		}
	}
	return nil
}

func (s *Scheduler) reconcile(ctx context.Context) error {
	checks, err := s.repository.ListPendingChecks(ctx, 50)
	if err != nil {
		return err
	}
	for _, pending := range checks {
		if pending.RunID == nil {
			continue
		}
		run, err := s.runs.Get(ctx, *pending.RunID)
		if err != nil {
			s.logger.Error(
				"load scheduled diagnostic run",
				"run_id", pending.RunID,
				"error", err,
			)
			continue
		}
		if run.Status == diagnostics.RunQueued || run.Status == diagnostics.RunRunning {
			continue
		}
		completed := completedCheck(pending, run)
		updated, notify, err := s.repository.CompleteCheck(ctx, completed)
		if err != nil {
			s.logger.Error(
				"complete scheduled monitoring check",
				"target_id", pending.TargetID,
				"run_id", pending.RunID,
				"error", err,
			)
			continue
		}
		if notify {
			message := "Target recovered"
			if updated.Status == StatusUnavailable {
				message = fmt.Sprintf(
					"Target failed %d consecutive checks",
					updated.ConsecutiveFailures,
				)
			}
			s.publishAlert(ctx, updated, message)
		}
	}
	return nil
}

func (s *Scheduler) publishAlert(ctx context.Context, target Target, message string) {
	alert := Alert{
		Target: target, Status: target.Status, Message: message,
		SentAt: time.Now().UTC(),
	}
	if err := s.notifier.Notify(ctx, alert); err != nil {
		s.logger.Error(
			"deliver monitoring alert",
			"target_id", target.ID,
			"status", target.Status,
			"error", err,
		)
	}
}

func completedCheck(pending Check, run diagnostics.DiagnosticRun) Check {
	pending.Status = StatusOperational
	for _, result := range run.Results {
		if result.Status == diagnostics.CheckWarning {
			pending.Status = StatusWarning
		}
		if result.Status == diagnostics.CheckFailed ||
			result.Status == diagnostics.CheckCancelled {
			pending.Status = StatusUnavailable
			if pending.ErrorMessage == "" {
				pending.ErrorMessage = result.ErrorMessage
			}
		}
		if result.Type == diagnostics.CheckTLS {
			var data struct {
				ValidUntil string `json:"validUntil"`
				Expired    bool   `json:"expired"`
			}
			if json.Unmarshal(result.Data, &data) == nil {
				if expiresAt, err := time.Parse(time.RFC3339, data.ValidUntil); err == nil {
					pending.TLSExpiresAt = &expiresAt
				}
				if data.Expired {
					pending.Status = StatusWarning
				}
			}
		}
	}
	if run.Status == diagnostics.RunFailed ||
		run.Status == diagnostics.RunCancelled ||
		run.Status == diagnostics.RunInterrupted {
		pending.Status = StatusUnavailable
	}
	if run.Status == diagnostics.RunPartial && pending.Status != StatusUnavailable {
		pending.Status = StatusWarning
	}
	if run.StartedAt != nil && run.CompletedAt != nil {
		latency := max(run.CompletedAt.Sub(*run.StartedAt).Milliseconds(), 0)
		pending.LatencyMS = &latency
	}
	if pending.ErrorMessage == "" && pending.Status == StatusUnavailable {
		messages := make([]string, 0)
		for _, result := range run.Results {
			if result.ErrorMessage != "" {
				messages = append(messages, result.ErrorMessage)
			}
		}
		pending.ErrorMessage = strings.Join(messages, "; ")
	}
	now := time.Now().UTC()
	pending.CheckedAt = &now
	return pending
}
