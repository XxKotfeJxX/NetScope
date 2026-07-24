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
			monitored_targets, check_results, diagnostic_runs
	`); err != nil {
		t.Fatalf("truncate test tables: %v", err)
	}

	repository := NewMonitoringRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	target := monitoring.Target{
		ID: uuid.New(), Name: "Production", Address: "example.com",
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
	page, err := repository.ListTargets(ctx, 1, 20)
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
	if err := repository.DeleteTarget(ctx, target.ID); err != nil {
		t.Fatalf("DeleteTarget() error = %v", err)
	}
}
