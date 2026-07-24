//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func createIntegrationWorkspace(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) uuid.UUID {
	t.Helper()
	userID := uuid.New()
	workspaceID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	if _, err := pool.Exec(ctx, `
		INSERT INTO users (
			id, email, display_name, password_hash, created_at, updated_at
		) VALUES ($1, $2, 'Integration Owner', 'test-hash', $3, $3)
	`, userID, "integration+"+userID.String()+"@example.com", now); err != nil {
		t.Fatalf("create integration user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspaces (
			id, name, slug, created_by, created_at, updated_at
		) VALUES ($1, 'Integration', $2, $3, $4, $4)
	`, workspaceID, "integration-"+workspaceID.String(), userID, now); err != nil {
		t.Fatalf("create integration workspace: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO workspace_members (
			workspace_id, user_id, role, created_at
		) VALUES ($1, $2, 'owner', $3)
	`, workspaceID, userID, now); err != nil {
		t.Fatalf("create integration membership: %v", err)
	}
	return workspaceID
}
