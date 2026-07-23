//go:build integration

package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunRepositoryLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if err := Migrate(ctx, config); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("NewWithConfig() error = %v", err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "TRUNCATE check_results, diagnostic_runs"); err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}

	repository := NewRunRepository(pool)
	runID := uuid.New()
	run := diagnostics.DiagnosticRun{
		ID: runID, TargetInput: "example.com", NormalizedHost: "example.com",
		Status: diagnostics.RunQueued, RequestedChecks: []diagnostics.CheckType{diagnostics.CheckDNS},
		Options:   diagnostics.RunOptions{TimeoutMS: 5000, IPVersion: "auto"},
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := repository.Create(ctx, run); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	result := diagnostics.CheckResult{
		ID: uuid.New(), RunID: runID, Type: diagnostics.CheckDNS,
		Status: diagnostics.CheckPassed, DurationMS: 12, Summary: "resolved",
		Data: json.RawMessage(`{"a":["192.0.2.1"]}`), StartedAt: now, CompletedAt: now,
	}
	if err := repository.SaveResult(ctx, result); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}
	if err := repository.UpdateStatus(ctx, runID, diagnostics.RunCompleted); err != nil {
		t.Fatalf("UpdateStatus() error = %v", err)
	}

	stored, err := repository.GetByID(ctx, runID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Status != diagnostics.RunCompleted {
		t.Fatalf("status = %q, want %q", stored.Status, diagnostics.RunCompleted)
	}
	if len(stored.Results) != 1 || stored.Results[0].Type != diagnostics.CheckDNS {
		t.Fatalf("results = %#v, want one DNS result", stored.Results)
	}

	page, err := repository.List(ctx, diagnostics.ListFilter{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if page.TotalItems != 1 || len(page.Items) != 1 {
		t.Fatalf("page = %#v, want one item", page)
	}
}
