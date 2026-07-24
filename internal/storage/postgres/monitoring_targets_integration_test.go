//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMonitoringRepositoryLifecycle(t *testing.T) {
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
	if _, err := pool.Exec(ctx, `
		TRUNCATE notification_channels, maintenance_windows, monitoring_checks,
			monitored_targets, check_results, diagnostic_runs,
			user_sessions, workspace_members, workspaces, users CASCADE
	`); err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}
	workspaceID := createIntegrationWorkspace(t, ctx, pool)

	repository := NewMonitoringRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	target := monitoring.Target{
		ID: uuid.New(), WorkspaceID: workspaceID,
		Name: "Production", Address: "example.com",
		Tags: []string{"production"},
		Checks: []diagnostics.CheckType{
			diagnostics.CheckDNS, diagnostics.CheckHTTP,
		},
		Options: diagnostics.RunOptions{
			TimeoutMS: 5000, HTTPMethod: "GET", IPVersion: "auto",
			MaxRedirects: 5, PingPackets: 4, MaxHops: 20,
		},
		IntervalSeconds: 300, Enabled: true, FailureThreshold: 3,
		Status: monitoring.StatusPending, NextCheckAt: now,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repository.CreateTarget(ctx, target); err != nil {
		t.Fatalf("CreateTarget() error = %v", err)
	}
	stored, err := repository.GetTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetTarget() error = %v", err)
	}
	if stored.Name != target.Name || len(stored.Checks) != 2 {
		t.Fatalf("stored target = %+v", stored)
	}
	page, err := repository.ListTargets(ctx, workspaceID, 1, 20)
	if err != nil || page.TotalItems != 1 {
		t.Fatalf("ListTargets() = %+v, %v", page, err)
	}
	if err := repository.SetTargetEnabled(ctx, target.ID, false); err != nil {
		t.Fatalf("SetTargetEnabled() error = %v", err)
	}
	stored, err = repository.GetTarget(ctx, target.ID)
	if err != nil || stored.Enabled || stored.Status != monitoring.StatusPaused {
		t.Fatalf("paused target = %+v, %v", stored, err)
	}

	window := monitoring.MaintenanceWindow{
		ID: uuid.New(), TargetID: target.ID, StartsAt: now,
		EndsAt: now.Add(time.Hour), Reason: "deploy", CreatedAt: now,
	}
	if err := repository.CreateMaintenanceWindow(ctx, window); err != nil {
		t.Fatalf("CreateMaintenanceWindow() error = %v", err)
	}
	windows, err := repository.ListMaintenanceWindows(ctx, target.ID)
	if err != nil || len(windows) != 1 {
		t.Fatalf("ListMaintenanceWindows() = %#v, %v", windows, err)
	}

	channel := monitoring.NotificationChannel{
		ID: uuid.New(), TargetID: target.ID, Kind: monitoring.NotificationWebhook,
		Destination: "https://hooks.example.com/netscope", Enabled: true,
		CreatedAt: now,
	}
	if err := repository.CreateNotificationChannel(ctx, channel); err != nil {
		t.Fatalf("CreateNotificationChannel() error = %v", err)
	}
	channels, err := repository.ListNotificationChannels(ctx, target.ID)
	if err != nil || len(channels) != 1 {
		t.Fatalf("ListNotificationChannels() = %#v, %v", channels, err)
	}
	if err := repository.SetTargetEnabled(ctx, target.ID, true); err != nil {
		t.Fatalf("resume target: %v", err)
	}
	due, err := repository.ClaimDueTargets(ctx, 10)
	if err != nil || len(due) != 0 {
		t.Fatalf("maintenance ClaimDueTargets() = %#v, %v", due, err)
	}
	stored, err = repository.GetTarget(ctx, target.ID)
	if err != nil || stored.Status != monitoring.StatusMaintenance {
		t.Fatalf("maintenance target = %+v, %v", stored, err)
	}
	if err := repository.DeleteMaintenanceWindow(ctx, target.ID, window.ID); err != nil {
		t.Fatalf("DeleteMaintenanceWindow() error = %v", err)
	}
	if err := repository.SetTargetEnabled(ctx, target.ID, true); err != nil {
		t.Fatalf("reschedule target: %v", err)
	}
	due, err = repository.ClaimDueTargets(ctx, 10)
	if err != nil || len(due) != 1 {
		t.Fatalf("ClaimDueTargets() = %#v, %v", due, err)
	}
	runID := uuid.New()
	runRepository := NewRunRepository(pool)
	if err := runRepository.Create(ctx, diagnostics.DiagnosticRun{
		ID: runID, WorkspaceID: workspaceID,
		TargetInput: target.Address, NormalizedHost: target.Address,
		Status: diagnostics.RunQueued, RequestedChecks: target.Checks,
		Options: target.Options, CreatedAt: now,
	}); err != nil {
		t.Fatalf("create scheduled run: %v", err)
	}
	if err := repository.CreateScheduledCheck(ctx, target.ID, runID); err != nil {
		t.Fatalf("CreateScheduledCheck() error = %v", err)
	}
	pending, err := repository.ListPendingChecks(ctx, 10)
	if err != nil || len(pending) != 1 || pending[0].RunID == nil {
		t.Fatalf("ListPendingChecks() = %#v, %v", pending, err)
	}
	latency := int64(125)
	checkedAt := now.Add(time.Second)
	pending[0].Status = monitoring.StatusOperational
	pending[0].LatencyMS = &latency
	pending[0].CheckedAt = &checkedAt
	updated, notify, err := repository.CompleteCheck(ctx, pending[0])
	if err != nil || notify || updated.Status != monitoring.StatusOperational {
		t.Fatalf("CompleteCheck() = %+v, %t, %v", updated, notify, err)
	}
	if updated.LastLatencyMS == nil || *updated.LastLatencyMS != latency {
		t.Fatalf("last latency = %v", updated.LastLatencyMS)
	}
	for attempt := 1; attempt <= target.FailureThreshold; attempt++ {
		updated, notify, err = repository.RecordDispatchFailure(
			ctx,
			target.ID,
			"queue unavailable",
		)
		if err != nil {
			t.Fatalf("RecordDispatchFailure(%d) error = %v", attempt, err)
		}
		if notify != (attempt == target.FailureThreshold) {
			t.Fatalf("notification at attempt %d = %t", attempt, notify)
		}
	}
	if updated.Status != monitoring.StatusUnavailable ||
		updated.ConsecutiveFailures != target.FailureThreshold {
		t.Fatalf("failed target = %+v", updated)
	}
	if err := repository.DeleteTarget(ctx, target.ID); err != nil {
		t.Fatalf("DeleteTarget() error = %v", err)
	}
}
