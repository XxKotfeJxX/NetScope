//go:build integration

package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/XxKotfeJxX/netscope/internal/reports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReportsRepositoryLifecycle(t *testing.T) {
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
		TRUNCATE public_report_links, run_comments, audit_events,
			user_sessions, workspace_members, workspaces, users,
			diagnostic_runs
		CASCADE
	`); err != nil {
		t.Fatalf("truncate report tables: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := createIntegrationIdentity(
		t,
		ctx,
		pool,
		"report-owner@example.com",
		now,
	)
	run := diagnostics.DiagnosticRun{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID,
		TargetInput: "example.com", NormalizedHost: "example.com",
		Status:          diagnostics.RunCompleted,
		RequestedChecks: []diagnostics.CheckType{diagnostics.CheckDNS},
		Options:         diagnostics.RunOptions{TimeoutMS: 1000},
		CreatedAt:       now,
	}
	if err := NewRunRepository(pool).Create(ctx, run); err != nil {
		t.Fatalf("Create(run) error = %v", err)
	}
	repository := NewReportsRepository(pool)
	comment := reports.Comment{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID, RunID: run.ID,
		AuthorID: owner.user.ID, AuthorName: owner.user.DisplayName,
		AuthorEmail: owner.user.Email, Body: "Evidence reviewed.",
		CreatedAt: now, UpdatedAt: now,
	}
	event := reportIntegrationAudit(
		owner.workspace.ID,
		owner.user.ID,
		comment.ID,
		"report.comment_created",
		now,
	)
	if err := repository.CreateComment(ctx, comment, event); err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	comments, err := repository.ListComments(ctx, owner.workspace.ID, run.ID)
	if err != nil || len(comments) != 1 ||
		comments[0].AuthorEmail != owner.user.Email {
		t.Fatalf("ListComments() = %#v, %v", comments, err)
	}

	token := "ns_share_integration-secret-abcdefghijklmnopqrstuvwxyz"
	hash := sha256.Sum256([]byte(token))
	link := reports.PublicLink{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID, RunID: run.ID,
		TokenPrefix: "ns_share_" + uuid.NewString()[:10],
		TokenHash:   hash[:], CreatedBy: owner.user.ID,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	event = reportIntegrationAudit(
		owner.workspace.ID,
		owner.user.ID,
		link.ID,
		"report.public_link_created",
		now,
	)
	if err := repository.CreatePublicLink(ctx, link, event); err != nil {
		t.Fatalf("CreatePublicLink() error = %v", err)
	}
	resolved, err := repository.ResolvePublicLink(ctx, hash[:], now)
	if err != nil || resolved.RunID != run.ID ||
		resolved.WorkspaceName != owner.workspace.Name {
		t.Fatalf("ResolvePublicLink() = %#v, %v", resolved, err)
	}
	event.ID = uuid.New()
	event.Action = "report.public_link_revoked"
	if err := repository.RevokePublicLink(
		ctx,
		owner.workspace.ID,
		run.ID,
		link.ID,
		now.Add(time.Minute),
		event,
	); err != nil {
		t.Fatalf("RevokePublicLink() error = %v", err)
	}
	if _, err := repository.ResolvePublicLink(
		ctx,
		hash[:],
		now.Add(2*time.Minute),
	); !errors.Is(err, reports.ErrPublicLinkMissing) {
		t.Fatalf("ResolvePublicLink(revoked) error = %v", err)
	}
	event.ID = uuid.New()
	event.Action = "report.comment_deleted"
	if err := repository.DeleteComment(
		ctx,
		owner.workspace.ID,
		run.ID,
		comment.ID,
		event,
	); err != nil {
		t.Fatalf("DeleteComment() error = %v", err)
	}
}

func reportIntegrationAudit(
	workspaceID uuid.UUID,
	actorID uuid.UUID,
	resourceID uuid.UUID,
	action string,
	createdAt time.Time,
) collaboration.AuditEvent {
	return collaboration.AuditEvent{
		ID: uuid.New(), WorkspaceID: workspaceID, ActorUserID: actorID,
		Action: action, ResourceType: "report", ResourceID: &resourceID,
		Metadata: map[string]any{}, CreatedAt: createdAt,
	}
}
