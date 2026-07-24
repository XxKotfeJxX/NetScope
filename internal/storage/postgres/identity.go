package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/XxKotfeJxX/netscope/internal/identity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IdentityRepository struct {
	pool *pgxpool.Pool
}

func NewIdentityRepository(pool *pgxpool.Pool) *IdentityRepository {
	return &IdentityRepository{pool: pool}
}

func (r *IdentityRepository) CreateRegistration(
	ctx context.Context,
	user identity.User,
	passwordHash string,
	workspace identity.Workspace,
	membership identity.Membership,
	session identity.Session,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin identity registration: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, `
		INSERT INTO users (
			id, email, display_name, password_hash, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, user.ID, user.Email, user.DisplayName, passwordHash,
		user.CreatedAt, user.UpdatedAt); err != nil {
		if isConstraint(err, "idx_users_email_unique") {
			return identity.ErrEmailExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	if err := insertWorkspace(ctx, transaction, workspace); err != nil {
		return err
	}
	if err := insertMembership(ctx, transaction, membership); err != nil {
		return err
	}
	if err := insertSession(ctx, transaction, session); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit identity registration: %w", err)
	}
	return nil
}

func (r *IdentityRepository) UserByEmail(
	ctx context.Context,
	email string,
) (identity.User, string, error) {
	var user identity.User
	var passwordHash string
	err := r.pool.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`, email).Scan(
		&user.ID, &user.Email, &user.DisplayName, &passwordHash,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.User{}, "", identity.ErrUserNotFound
	}
	if err != nil {
		return identity.User{}, "", fmt.Errorf("find user by email: %w", err)
	}
	return user, passwordHash, nil
}

func (r *IdentityRepository) CreateSession(
	ctx context.Context,
	session identity.Session,
) error {
	return insertSession(ctx, r.pool, session)
}

func (r *IdentityRepository) SessionByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (identity.Session, identity.User, error) {
	var session identity.Session
	var user identity.User
	err := r.pool.QueryRow(ctx, `
		SELECT session.id, session.user_id, session.token_hash,
			session.expires_at, session.created_at, session.last_used_at,
			account.id, account.email, account.display_name,
			account.created_at, account.updated_at
		FROM user_sessions AS session
		JOIN users AS account ON account.id = session.user_id
		WHERE session.token_hash = $1 AND session.expires_at > NOW()
	`, tokenHash).Scan(
		&session.ID, &session.UserID, &session.TokenHash,
		&session.ExpiresAt, &session.CreatedAt, &session.LastUsedAt,
		&user.ID, &user.Email, &user.DisplayName,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Session{}, identity.User{}, identity.ErrUnauthenticated
	}
	if err != nil {
		return identity.Session{}, identity.User{}, fmt.Errorf(
			"authenticate session: %w",
			err,
		)
	}
	_, _ = r.pool.Exec(ctx, `
		UPDATE user_sessions SET last_used_at = NOW() WHERE id = $1
	`, session.ID)
	return session, user, nil
}

func (r *IdentityRepository) DeleteSession(
	ctx context.Context,
	tokenHash []byte,
) error {
	if _, err := r.pool.Exec(
		ctx,
		`DELETE FROM user_sessions WHERE token_hash = $1`,
		tokenHash,
	); err != nil {
		return fmt.Errorf("delete user session: %w", err)
	}
	return nil
}

func (r *IdentityRepository) ListWorkspaces(
	ctx context.Context,
	userID uuid.UUID,
) ([]identity.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT workspace.id, workspace.name, workspace.slug,
			membership.role, workspace.created_by,
			workspace.created_at, workspace.updated_at
		FROM workspaces AS workspace
		JOIN workspace_members AS membership
			ON membership.workspace_id = workspace.id
		WHERE membership.user_id = $1
		ORDER BY workspace.created_at, workspace.name
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user workspaces: %w", err)
	}
	defer rows.Close()

	workspaces := make([]identity.Workspace, 0)
	for rows.Next() {
		var workspace identity.Workspace
		if err := rows.Scan(
			&workspace.ID, &workspace.Name, &workspace.Slug,
			&workspace.Role, &workspace.CreatedBy,
			&workspace.CreatedAt, &workspace.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user workspace: %w", err)
		}
		workspaces = append(workspaces, workspace)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user workspaces: %w", err)
	}
	return workspaces, nil
}

func (r *IdentityRepository) CreateWorkspace(
	ctx context.Context,
	workspace identity.Workspace,
	membership identity.Membership,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workspace creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if err := insertWorkspace(ctx, transaction, workspace); err != nil {
		return err
	}
	if err := insertMembership(ctx, transaction, membership); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit workspace creation: %w", err)
	}
	return nil
}

func (r *IdentityRepository) Membership(
	ctx context.Context,
	userID uuid.UUID,
	workspaceID uuid.UUID,
) (identity.Membership, error) {
	var membership identity.Membership
	err := r.pool.QueryRow(ctx, `
		SELECT workspace_id, user_id, role, created_at
		FROM workspace_members
		WHERE workspace_id = $1 AND user_id = $2
	`, workspaceID, userID).Scan(
		&membership.WorkspaceID, &membership.UserID,
		&membership.Role, &membership.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Membership{}, identity.ErrWorkspaceNotFound
	}
	if err != nil {
		return identity.Membership{}, fmt.Errorf("get workspace membership: %w", err)
	}
	return membership, nil
}

type postgresExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertWorkspace(
	ctx context.Context,
	executor postgresExecutor,
	workspace identity.Workspace,
) error {
	_, err := executor.Exec(ctx, `
		INSERT INTO workspaces (
			id, name, slug, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, workspace.ID, workspace.Name, workspace.Slug, workspace.CreatedBy,
		workspace.CreatedAt, workspace.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	return nil
}

func insertMembership(
	ctx context.Context,
	executor postgresExecutor,
	membership identity.Membership,
) error {
	_, err := executor.Exec(ctx, `
		INSERT INTO workspace_members (
			workspace_id, user_id, role, created_at
		) VALUES ($1, $2, $3, $4)
	`, membership.WorkspaceID, membership.UserID,
		membership.Role, membership.CreatedAt)
	if err != nil {
		return fmt.Errorf("create workspace membership: %w", err)
	}
	return nil
}

func insertSession(
	ctx context.Context,
	executor postgresExecutor,
	session identity.Session,
) error {
	_, err := executor.Exec(ctx, `
		INSERT INTO user_sessions (
			id, user_id, token_hash, expires_at, created_at, last_used_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt,
		session.CreatedAt, session.LastUsedAt)
	if err != nil {
		return fmt.Errorf("create user session: %w", err)
	}
	return nil
}

func isConstraint(err error, name string) bool {
	var postgresError *pgconn.PgError
	return errors.As(err, &postgresError) &&
		postgresError.ConstraintName == name
}
