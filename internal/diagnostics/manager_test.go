package diagnostics

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/target"
	"github.com/google/uuid"
)

type memoryRepository struct {
	mutex   sync.Mutex
	run     DiagnosticRun
	results []CheckResult
}

func (r *memoryRepository) Create(_ context.Context, run DiagnosticRun) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.run = run
	return nil
}

func (r *memoryRepository) GetByID(_ context.Context, id uuid.UUID) (DiagnosticRun, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.run.ID != id {
		return DiagnosticRun{}, ErrRunNotFound
	}
	run := r.run
	run.Results = append([]CheckResult(nil), r.results...)
	return run, nil
}

func (r *memoryRepository) List(context.Context, ListFilter) (Page, error) {
	return Page{}, nil
}

func (r *memoryRepository) UpdateStatus(_ context.Context, id uuid.UUID, status RunStatus) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.run.ID != id {
		return ErrRunNotFound
	}
	r.run.Status = status
	return nil
}

func (r *memoryRepository) SaveResult(_ context.Context, result CheckResult) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.results = append(r.results, result)
	return nil
}

type passingProbe struct{}

func (passingProbe) Type() CheckType {
	return CheckDNS
}

func (passingProbe) Run(_ context.Context, _ target.Target, _ RunOptions) CheckResult {
	now := time.Now().UTC()
	return CheckResult{
		ID: uuid.New(), Type: CheckDNS, Status: CheckPassed,
		Data: json.RawMessage(`{"a":["192.0.2.1"]}`), StartedAt: now, CompletedAt: now,
	}
}

func TestManagerExecutesQueuedRun(t *testing.T) {
	t.Parallel()

	runID := uuid.New()
	repository := &memoryRepository{run: DiagnosticRun{
		ID: runID, TargetInput: "example.com", Status: RunQueued,
		RequestedChecks: []CheckType{CheckDNS}, Options: RunOptions{TimeoutMS: 1000},
	}}
	manager := NewManager(
		repository,
		NewHub(),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		1,
		1,
		1,
		passingProbe{},
	)
	manager.Start()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := manager.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown() error = %v", err)
		}
	})

	if err := manager.Enqueue(runID); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		run, err := repository.GetByID(context.Background(), runID)
		if err != nil {
			t.Fatalf("GetByID() error = %v", err)
		}
		if run.Status == RunCompleted {
			if len(run.Results) != 1 {
				t.Fatalf("results = %d, want 1", len(run.Results))
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("run did not complete")
}
