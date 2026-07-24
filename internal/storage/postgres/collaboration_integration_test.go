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
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCollaborationRepositoryLifecycle(t *testing.T) {
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
		TRUNCATE audit_events, user_sessions, workspace_members, workspaces, users
		CASCADE
	`); err != nil {
		t.Fatalf("truncate collaboration tables: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	owner := createIntegrationIdentity(
		t,
		ctx,
		pool,
		"collab-owner@example.com",
		now,
	)
	operator := createIntegrationIdentity(
		t,
		ctx,
		pool,
		"collab-operator@example.com",
		now,
	)
	repository := NewCollaborationRepository(pool)
	event := collaboration.AuditEvent{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID,
		ActorUserID: owner.user.ID, Action: "workspace.member_added",
		ResourceType: "workspace_member", Metadata: map[string]any{
			"email": operator.user.Email,
			"role":  identity.RoleOperator,
		},
		CreatedAt: now,
	}
	member, err := repository.AddMember(
		ctx,
		owner.workspace.ID,
		operator.user.Email,
		identity.RoleOperator,
		event,
	)
	if err != nil || member.UserID != operator.user.ID {
		t.Fatalf("AddMember() = %#v, %v", member, err)
	}
	if _, err := repository.AddMember(
		ctx,
		owner.workspace.ID,
		operator.user.Email,
		identity.RoleViewer,
		event,
	); !errors.Is(err, collaboration.ErrMemberExists) {
		t.Fatalf("duplicate AddMember() error = %v", err)
	}
	event.ID = uuid.New()
	event.Action = "workspace.member_role_updated"
	member, err = repository.UpdateMemberRole(
		ctx,
		owner.workspace.ID,
		operator.user.ID,
		identity.RoleAdmin,
		event,
	)
	if err != nil || member.Role != identity.RoleAdmin {
		t.Fatalf("UpdateMemberRole() = %#v, %v", member, err)
	}
	event.ID = uuid.New()
	event.Action = "workspace.member_removed"
	if err := repository.RemoveMember(
		ctx,
		owner.workspace.ID,
		operator.user.ID,
		event,
	); err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}
	if _, err := repository.Member(
		ctx,
		owner.workspace.ID,
		operator.user.ID,
	); !errors.Is(err, collaboration.ErrMemberMissing) {
		t.Fatalf("Member(after remove) error = %v", err)
	}
	if _, err := repository.UpdateMemberRole(
		ctx,
		owner.workspace.ID,
		owner.user.ID,
		identity.RoleAdmin,
		event,
	); !errors.Is(err, collaboration.ErrLastOwner) {
		t.Fatalf("UpdateMemberRole(last owner) error = %v", err)
	}
	events, total, err := repository.ListAudit(
		ctx,
		owner.workspace.ID,
		1,
		10,
	)
	if err != nil || total != 3 || len(events) != 3 {
		t.Fatalf("ListAudit() = %#v, %d, %v", events, total, err)
	}
	token := identity.APIKeyTokenPrefix + "integration-secret-abcdefghijklmnopqrstuvwxyz"
	tokenHash := sha256.Sum256([]byte(token))
	key := collaboration.APIKey{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID, Name: "Integration",
		Prefix: "ns_key_integratio", TokenHash: tokenHash[:],
		Role: identity.RoleOperator, CreatedBy: owner.user.ID,
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	}
	keyEvent := collaboration.AuditEvent{
		ID: uuid.New(), WorkspaceID: owner.workspace.ID,
		ActorUserID: owner.user.ID, Action: "workspace.api_key_created",
		ResourceType: "api_key", ResourceID: &key.ID,
		Metadata: map[string]any{"name": key.Name}, CreatedAt: now,
	}
	if err := repository.CreateAPIKey(ctx, key, keyEvent); err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	credential, err := NewIdentityRepository(pool).APIKeyByTokenHash(
		ctx,
		tokenHash[:],
	)
	if err != nil || credential.Workspace.ID != owner.workspace.ID ||
		credential.Workspace.Role != identity.RoleOperator {
		t.Fatalf("APIKeyByTokenHash() = %#v, %v", credential, err)
	}
	keyEvent.ID = uuid.New()
	keyEvent.Action = "workspace.api_key_revoked"
	if err := repository.RevokeAPIKey(
		ctx,
		owner.workspace.ID,
		key.ID,
		now.Add(time.Minute),
		keyEvent,
	); err != nil {
		t.Fatalf("RevokeAPIKey() error = %v", err)
	}
	if _, err := NewIdentityRepository(pool).APIKeyByTokenHash(
		ctx,
		tokenHash[:],
	); !errors.Is(err, identity.ErrUnauthenticated) {
		t.Fatalf("APIKeyByTokenHash(revoked) error = %v", err)
	}
}

type integrationIdentity struct {
	user      identity.User
	workspace identity.Workspace
}

func createIntegrationIdentity(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	email string,
	now time.Time,
) integrationIdentity {
	t.Helper()
	user := identity.User{
		ID: uuid.New(), Email: email, DisplayName: email,
		CreatedAt: now, UpdatedAt: now,
	}
	workspace := identity.Workspace{
		ID: uuid.New(), Name: email, Slug: uuid.NewString(),
		Role: identity.RoleOwner, CreatedBy: user.ID,
		CreatedAt: now, UpdatedAt: now,
	}
	membership := identity.Membership{
		WorkspaceID: workspace.ID, UserID: user.ID,
		Role: identity.RoleOwner, CreatedAt: now,
	}
	session := identity.Session{
		ID: uuid.New(), UserID: user.ID, TokenHash: uuid.New().NodeID(),
		ExpiresAt: now.Add(time.Hour), CreatedAt: now, LastUsedAt: now,
	}
	if err := NewIdentityRepository(pool).CreateRegistration(
		ctx,
		user,
		"hash",
		workspace,
		membership,
		session,
	); err != nil {
		t.Fatalf("CreateRegistration(%s) error = %v", email, err)
	}
	return integrationIdentity{user: user, workspace: workspace}
}
