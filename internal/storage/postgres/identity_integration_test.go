//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIdentityRepositoryLifecycle(t *testing.T) {
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
		TRUNCATE user_sessions, workspace_members, workspaces, users CASCADE
	`); err != nil {
		t.Fatalf("truncate identity tables: %v", err)
	}

	repository := NewIdentityRepository(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	legacyRunID := uuid.New()
	legacyTargetID := uuid.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO diagnostic_runs (
			id, target_input, normalized_host, status, requested_checks,
			options, created_at
		) VALUES ($1, 'example.com', 'example.com', 'completed',
			ARRAY['dns'], '{}', $2)
	`, legacyRunID, now); err != nil {
		t.Fatalf("create legacy diagnostic run: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO monitored_targets (
			id, name, address, tags, requested_checks, options,
			interval_seconds, enabled, failure_threshold, status,
			next_check_at, created_at, updated_at
		) VALUES ($1, 'Legacy', 'example.com', ARRAY['legacy'], ARRAY['dns'],
			'{}', 300, TRUE, 3, 'pending', $2, $2, $2)
	`, legacyTargetID, now); err != nil {
		t.Fatalf("create legacy monitored target: %v", err)
	}
	user := identity.User{
		ID: uuid.New(), Email: "owner@example.com", DisplayName: "Owner",
		CreatedAt: now, UpdatedAt: now,
	}
	workspace := identity.Workspace{
		ID: uuid.New(), Name: "Acme", Slug: "acme-test",
		Role: identity.RoleOwner, CreatedBy: user.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	membership := identity.Membership{
		WorkspaceID: workspace.ID, UserID: user.ID,
		Role: identity.RoleOwner, CreatedAt: now,
	}
	session := identity.Session{
		ID: uuid.New(), UserID: user.ID, TokenHash: []byte("token-hash"),
		ExpiresAt: now.Add(time.Hour), CreatedAt: now, LastUsedAt: now,
	}
	if err := repository.CreateRegistration(
		ctx,
		user,
		"password-hash",
		workspace,
		membership,
		session,
	); err != nil {
		t.Fatalf("CreateRegistration() error = %v", err)
	}
	var claimedRunWorkspace uuid.UUID
	var claimedTargetWorkspace uuid.UUID
	if err := pool.QueryRow(ctx,
		`SELECT workspace_id FROM diagnostic_runs WHERE id = $1`,
		legacyRunID,
	).Scan(&claimedRunWorkspace); err != nil {
		t.Fatalf("load claimed run: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`SELECT workspace_id FROM monitored_targets WHERE id = $1`,
		legacyTargetID,
	).Scan(&claimedTargetWorkspace); err != nil {
		t.Fatalf("load claimed target: %v", err)
	}
	if claimedRunWorkspace != workspace.ID ||
		claimedTargetWorkspace != workspace.ID {
		t.Fatalf(
			"legacy workspaces = %s, %s, want %s",
			claimedRunWorkspace,
			claimedTargetWorkspace,
			workspace.ID,
		)
	}
	stored, hash, err := repository.UserByEmail(ctx, "OWNER@example.com")
	if err != nil || stored.ID != user.ID || hash != "password-hash" {
		t.Fatalf("UserByEmail() = %#v, %q, %v", stored, hash, err)
	}
	authSession, account, err := repository.SessionByTokenHash(
		ctx,
		session.TokenHash,
	)
	if err != nil || authSession.ID != session.ID || account.ID != user.ID {
		t.Fatalf("SessionByTokenHash() = %#v, %#v, %v", authSession, account, err)
	}
	workspaces, err := repository.ListWorkspaces(ctx, user.ID)
	if err != nil || len(workspaces) != 1 ||
		workspaces[0].Role != identity.RoleOwner {
		t.Fatalf("ListWorkspaces() = %#v, %v", workspaces, err)
	}
	if err := repository.CreateRegistration(
		ctx,
		identity.User{
			ID: uuid.New(), Email: "OWNER@example.com", DisplayName: "Duplicate",
			CreatedAt: now, UpdatedAt: now,
		},
		"hash",
		identity.Workspace{
			ID: uuid.New(), Name: "Duplicate", Slug: "duplicate",
			CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now,
		},
		membership,
		identity.Session{},
	); !errors.Is(err, identity.ErrEmailExists) {
		t.Fatalf("duplicate CreateRegistration() error = %v", err)
	}
	if err := repository.DeleteSession(ctx, session.TokenHash); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if _, _, err := repository.SessionByTokenHash(
		ctx,
		session.TokenHash,
	); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("SessionByTokenHash(after delete) error = %v", err)
	}
}
