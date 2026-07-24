package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CollaborationRepository struct {
	pool *pgxpool.Pool
}

func NewCollaborationRepository(pool *pgxpool.Pool) *CollaborationRepository {
	return &CollaborationRepository{pool: pool}
}

func (r *CollaborationRepository) ListMembers(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]collaboration.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT account.id, account.email, account.display_name,
			membership.role, membership.created_at
		FROM workspace_members AS membership
		JOIN users AS account ON account.id = membership.user_id
		WHERE membership.workspace_id = $1
		ORDER BY
			CASE membership.role
				WHEN 'owner' THEN 1 WHEN 'admin' THEN 2
				WHEN 'operator' THEN 3 ELSE 4
			END,
			LOWER(account.display_name),
			account.id
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	defer rows.Close()

	members := make([]collaboration.Member, 0)
	for rows.Next() {
		var member collaboration.Member
		if err := rows.Scan(
			&member.UserID,
			&member.Email,
			&member.DisplayName,
			&member.Role,
			&member.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace members: %w", err)
	}
	return members, nil
}

func (r *CollaborationRepository) Member(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
) (collaboration.Member, error) {
	return getMember(ctx, r.pool, workspaceID, userID, false)
}

func (r *CollaborationRepository) AddMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	email string,
	role identity.Role,
	event collaboration.AuditEvent,
) (collaboration.Member, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return collaboration.Member{}, fmt.Errorf("begin add member: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var userID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		SELECT id FROM users WHERE LOWER(email) = LOWER($1)
	`, email).Scan(&userID); errors.Is(err, pgx.ErrNoRows) {
		return collaboration.Member{}, identity.ErrUserNotFound
	} else if err != nil {
		return collaboration.Member{}, fmt.Errorf("find invited user: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
	`, workspaceID, userID, role); err != nil {
		if isConstraint(err, "workspace_members_pkey") {
			return collaboration.Member{}, collaboration.ErrMemberExists
		}
		return collaboration.Member{}, fmt.Errorf("add workspace member: %w", err)
	}
	event.ResourceID = &userID
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return collaboration.Member{}, err
	}
	member, err := getMember(ctx, transaction, workspaceID, userID, false)
	if err != nil {
		return collaboration.Member{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return collaboration.Member{}, fmt.Errorf("commit add member: %w", err)
	}
	return member, nil
}

func (r *CollaborationRepository) UpdateMemberRole(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	role identity.Role,
	event collaboration.AuditEvent,
) (collaboration.Member, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return collaboration.Member{}, fmt.Errorf("begin update member: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	current, err := getMember(ctx, transaction, workspaceID, userID, true)
	if err != nil {
		return collaboration.Member{}, err
	}
	if current.Role == identity.RoleOwner && role != identity.RoleOwner {
		if err := retainOwner(ctx, transaction, workspaceID); err != nil {
			return collaboration.Member{}, err
		}
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE workspace_members SET role = $3
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID, role); err != nil {
		return collaboration.Member{}, fmt.Errorf("update workspace member: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return collaboration.Member{}, err
	}
	member, err := getMember(ctx, transaction, workspaceID, userID, false)
	if err != nil {
		return collaboration.Member{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return collaboration.Member{}, fmt.Errorf("commit update member: %w", err)
	}
	return member, nil
}

func (r *CollaborationRepository) RemoveMember(
	ctx context.Context,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin remove member: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	current, err := getMember(ctx, transaction, workspaceID, userID, true)
	if err != nil {
		return err
	}
	if current.Role == identity.RoleOwner {
		if err := retainOwner(ctx, transaction, workspaceID); err != nil {
			return err
		}
	}
	if _, err := transaction.Exec(ctx, `
		DELETE FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID); err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit remove member: %w", err)
	}
	return nil
}

func (r *CollaborationRepository) ListAudit(
	ctx context.Context,
	workspaceID uuid.UUID,
	page int,
	pageSize int,
) ([]collaboration.AuditEvent, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_events WHERE workspace_id = $1
	`, workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, actor_user_id, action, resource_type,
			resource_id, metadata, created_at
		FROM audit_events
		WHERE workspace_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, workspaceID, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := make([]collaboration.AuditEvent, 0)
	for rows.Next() {
		var event collaboration.AuditEvent
		var metadata []byte
		if err := rows.Scan(
			&event.ID,
			&event.WorkspaceID,
			&event.ActorUserID,
			&event.Action,
			&event.ResourceType,
			&event.ResourceID,
			&metadata,
			&event.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit event: %w", err)
		}
		if err := json.Unmarshal(metadata, &event.Metadata); err != nil {
			return nil, 0, fmt.Errorf("decode audit metadata: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate audit events: %w", err)
	}
	return events, total, nil
}

func (r *CollaborationRepository) ListAPIKeys(
	ctx context.Context,
	workspaceID uuid.UUID,
) ([]collaboration.APIKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, name, prefix, role, created_by, expires_at,
			last_used_at, revoked_at, created_at
		FROM api_keys
		WHERE workspace_id = $1
		ORDER BY created_at DESC, id DESC
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace API keys: %w", err)
	}
	defer rows.Close()
	keys := make([]collaboration.APIKey, 0)
	for rows.Next() {
		var key collaboration.APIKey
		if err := rows.Scan(
			&key.ID,
			&key.WorkspaceID,
			&key.Name,
			&key.Prefix,
			&key.Role,
			&key.CreatedBy,
			&key.ExpiresAt,
			&key.LastUsedAt,
			&key.RevokedAt,
			&key.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workspace API key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace API keys: %w", err)
	}
	return keys, nil
}

func (r *CollaborationRepository) CreateAPIKey(
	ctx context.Context,
	key collaboration.APIKey,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin API key creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO api_keys (
			id, workspace_id, name, prefix, token_hash, role, created_by,
			expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, key.ID, key.WorkspaceID, key.Name, key.Prefix, key.TokenHash,
		key.Role, key.CreatedBy, key.ExpiresAt, key.CreatedAt); err != nil {
		return fmt.Errorf("create workspace API key: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit API key creation: %w", err)
	}
	return nil
}

func (r *CollaborationRepository) RevokeAPIKey(
	ctx context.Context,
	workspaceID uuid.UUID,
	keyID uuid.UUID,
	revokedAt time.Time,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin API key revocation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var name string
	var prefix string
	if err := transaction.QueryRow(ctx, `
		SELECT name, prefix
		FROM api_keys
		WHERE id = $1 AND workspace_id = $2 AND revoked_at IS NULL
		FOR UPDATE
	`, keyID, workspaceID).Scan(&name, &prefix); errors.Is(err, pgx.ErrNoRows) {
		return collaboration.ErrAPIKeyMissing
	} else if err != nil {
		return fmt.Errorf("get workspace API key for revocation: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE api_keys SET revoked_at = $3
		WHERE id = $1 AND workspace_id = $2
	`, keyID, workspaceID, revokedAt); err != nil {
		return fmt.Errorf("revoke workspace API key: %w", err)
	}
	event.Metadata = map[string]any{"name": name, "prefix": prefix}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit API key revocation: %w", err)
	}
	return nil
}

type postgresQueryExecutor interface {
	postgresExecutor
	QueryRow(context.Context, string, ...any) pgx.Row
}

func getMember(
	ctx context.Context,
	executor postgresQueryExecutor,
	workspaceID uuid.UUID,
	userID uuid.UUID,
	forUpdate bool,
) (collaboration.Member, error) {
	query := `
		SELECT account.id, account.email, account.display_name,
			membership.role, membership.created_at
		FROM workspace_members AS membership
		JOIN users AS account ON account.id = membership.user_id
		WHERE membership.workspace_id = $1 AND membership.user_id = $2
	`
	if forUpdate {
		query += " FOR UPDATE OF membership"
	}
	var member collaboration.Member
	err := executor.QueryRow(ctx, query, workspaceID, userID).Scan(
		&member.UserID,
		&member.Email,
		&member.DisplayName,
		&member.Role,
		&member.JoinedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return collaboration.Member{}, collaboration.ErrMemberMissing
	}
	if err != nil {
		return collaboration.Member{}, fmt.Errorf("get workspace member: %w", err)
	}
	return member, nil
}

func retainOwner(
	ctx context.Context,
	transaction pgx.Tx,
	workspaceID uuid.UUID,
) error {
	rows, err := transaction.Query(ctx, `
		SELECT user_id FROM workspace_members
		WHERE workspace_id = $1 AND role = 'owner'
		FOR UPDATE
	`, workspaceID)
	if err != nil {
		return fmt.Errorf("count workspace owners: %w", err)
	}
	defer rows.Close()
	ownerCount := 0
	for rows.Next() {
		ownerCount++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("lock workspace owners: %w", err)
	}
	if ownerCount <= 1 {
		return collaboration.ErrLastOwner
	}
	return nil
}

func insertAuditEvent(
	ctx context.Context,
	executor postgresExecutor,
	event collaboration.AuditEvent,
) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	if _, err := executor.Exec(ctx, `
		INSERT INTO audit_events (
			id, workspace_id, actor_user_id, action, resource_type,
			resource_id, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.WorkspaceID, event.ActorUserID, event.Action,
		event.ResourceType, event.ResourceID, metadata, event.CreatedAt); err != nil {
		return fmt.Errorf("create audit event: %w", err)
	}
	return nil
}
