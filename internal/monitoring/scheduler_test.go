package monitoring

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
)

type schedulerRepository struct {
	Repository
	due      []Target
	linked   uuid.UUID
	failures int
}

func (r *schedulerRepository) ClaimDueTargets(
	context.Context,
	int,
) ([]Target, error) {
	return r.due, nil
}

func (r *schedulerRepository) CreateScheduledCheck(
	_ context.Context,
	_ uuid.UUID,
	runID uuid.UUID,
) error {
	r.linked = runID
	return nil
}

func (r *schedulerRepository) ListPendingChecks(
	context.Context,
	int,
) ([]Check, error) {
	return nil, nil
}

func (r *schedulerRepository) RecordDispatchFailure(
	context.Context,
	uuid.UUID,
	string,
) (Target, bool, error) {
	r.failures++
	return Target{}, false, nil
}

type schedulerRuns struct {
	created diagnostics.DiagnosticRun
	err     error
}

func (schedulerRuns) Supports(diagnostics.CheckType) bool {
	return true
}

func (r schedulerRuns) Create(
	context.Context,
	string,
	[]diagnostics.CheckType,
	diagnostics.RunOptions,
) (diagnostics.DiagnosticRun, error) {
	return r.created, r.err
}

func (schedulerRuns) Get(
	context.Context,
	uuid.UUID,
) (diagnostics.DiagnosticRun, error) {
	return diagnostics.DiagnosticRun{}, nil
}

func TestSchedulerDispatchLinksDiagnosticRun(t *testing.T) {
	t.Parallel()

	runID := uuid.New()
	repository := &schedulerRepository{due: []Target{{
		ID: uuid.New(), Address: "example.com",
		Checks: []diagnostics.CheckType{diagnostics.CheckDNS},
	}}}
	scheduler := NewScheduler(
		repository,
		schedulerRuns{created: diagnostics.DiagnosticRun{ID: runID}},
		NopNotifier{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		time.Second,
	)
	if err := scheduler.dispatch(context.Background()); err != nil {
		t.Fatalf("dispatch() error = %v", err)
	}
	if repository.linked != runID {
		t.Fatalf("linked run = %s, want %s", repository.linked, runID)
	}
}

func TestCompletedCheckCapturesLatencyAndTLSExpiry(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	completed := started.Add(250 * time.Millisecond)
	expires := started.Add(30 * 24 * time.Hour)
	check := completedCheck(Check{}, diagnostics.DiagnosticRun{
		Status: diagnostics.RunCompleted, StartedAt: &started, CompletedAt: &completed,
		Results: []diagnostics.CheckResult{{
			Type: diagnostics.CheckTLS, Status: diagnostics.CheckWarning,
			Data: json.RawMessage(`{"validUntil":"` + expires.Format(time.RFC3339) + `"}`),
		}},
	})
	if check.Status != StatusWarning {
		t.Fatalf("status = %s, want warning", check.Status)
	}
	if check.LatencyMS == nil || *check.LatencyMS != 250 {
		t.Fatalf("latency = %v, want 250", check.LatencyMS)
	}
	if check.TLSExpiresAt == nil || !check.TLSExpiresAt.Equal(expires) {
		t.Fatalf("TLS expiry = %v, want %v", check.TLSExpiresAt, expires)
	}
	if check.CheckedAt == nil {
		t.Fatal("checkedAt is nil")
	}
}
