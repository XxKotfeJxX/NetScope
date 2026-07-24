package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/collaboration"
	"github.com/XxKotfeJxX/netscope/internal/reports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportsRepository struct {
	pool *pgxpool.Pool
}

func NewReportsRepository(pool *pgxpool.Pool) *ReportsRepository {
	return &ReportsRepository{pool: pool}
}

func (r *ReportsRepository) ListComments(
	ctx context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
) ([]reports.Comment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT comment.id, comment.workspace_id, comment.run_id,
			comment.author_user_id, account.display_name, account.email,
			comment.body, comment.created_at, comment.updated_at
		FROM run_comments AS comment
		JOIN users AS account ON account.id = comment.author_user_id
		WHERE comment.workspace_id = $1 AND comment.run_id = $2
		ORDER BY comment.created_at, comment.id
	`, workspaceID, runID)
	if err != nil {
		return nil, fmt.Errorf("list report comments: %w", err)
	}
	defer rows.Close()
	comments := make([]reports.Comment, 0)
	for rows.Next() {
		var comment reports.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.WorkspaceID,
			&comment.RunID,
			&comment.AuthorID,
			&comment.AuthorName,
			&comment.AuthorEmail,
			&comment.Body,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan report comment: %w", err)
		}
		comments = append(comments, comment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate report comments: %w", err)
	}
	return comments, nil
}

func (r *ReportsRepository) Comment(
	ctx context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
	commentID uuid.UUID,
) (reports.Comment, error) {
	var comment reports.Comment
	err := r.pool.QueryRow(ctx, `
		SELECT comment.id, comment.workspace_id, comment.run_id,
			comment.author_user_id, account.display_name, account.email,
			comment.body, comment.created_at, comment.updated_at
		FROM run_comments AS comment
		JOIN users AS account ON account.id = comment.author_user_id
		WHERE comment.workspace_id = $1 AND comment.run_id = $2
			AND comment.id = $3
	`, workspaceID, runID, commentID).Scan(
		&comment.ID,
		&comment.WorkspaceID,
		&comment.RunID,
		&comment.AuthorID,
		&comment.AuthorName,
		&comment.AuthorEmail,
		&comment.Body,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return reports.Comment{}, reports.ErrCommentMissing
	}
	if err != nil {
		return reports.Comment{}, fmt.Errorf("get report comment: %w", err)
	}
	return comment, nil
}

func (r *ReportsRepository) CreateComment(
	ctx context.Context,
	comment reports.Comment,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin report comment creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO run_comments (
			id, workspace_id, run_id, author_user_id, body, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, comment.ID, comment.WorkspaceID, comment.RunID, comment.AuthorID,
		comment.Body, comment.CreatedAt, comment.UpdatedAt); err != nil {
		return fmt.Errorf("create report comment: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit report comment creation: %w", err)
	}
	return nil
}

func (r *ReportsRepository) DeleteComment(
	ctx context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
	commentID uuid.UUID,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin report comment deletion: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	result, err := transaction.Exec(ctx, `
		DELETE FROM run_comments
		WHERE workspace_id = $1 AND run_id = $2 AND id = $3
	`, workspaceID, runID, commentID)
	if err != nil {
		return fmt.Errorf("delete report comment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return reports.ErrCommentMissing
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit report comment deletion: %w", err)
	}
	return nil
}

func (r *ReportsRepository) ListPublicLinks(
	ctx context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
) ([]reports.PublicLink, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, run_id, token_prefix, created_by, expires_at,
			revoked_at, last_viewed_at, created_at
		FROM public_report_links
		WHERE workspace_id = $1 AND run_id = $2
		ORDER BY created_at DESC, id DESC
	`, workspaceID, runID)
	if err != nil {
		return nil, fmt.Errorf("list public report links: %w", err)
	}
	defer rows.Close()
	links := make([]reports.PublicLink, 0)
	for rows.Next() {
		var link reports.PublicLink
		if err := rows.Scan(
			&link.ID,
			&link.WorkspaceID,
			&link.RunID,
			&link.TokenPrefix,
			&link.CreatedBy,
			&link.ExpiresAt,
			&link.RevokedAt,
			&link.LastViewedAt,
			&link.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan public report link: %w", err)
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public report links: %w", err)
	}
	return links, nil
}

func (r *ReportsRepository) CreatePublicLink(
	ctx context.Context,
	link reports.PublicLink,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public report link creation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	if _, err := transaction.Exec(ctx, `
		INSERT INTO public_report_links (
			id, workspace_id, run_id, token_prefix, token_hash, created_by,
			expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, link.ID, link.WorkspaceID, link.RunID, link.TokenPrefix, link.TokenHash,
		link.CreatedBy, link.ExpiresAt, link.CreatedAt); err != nil {
		return fmt.Errorf("create public report link: %w", err)
	}
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit public report link creation: %w", err)
	}
	return nil
}

func (r *ReportsRepository) RevokePublicLink(
	ctx context.Context,
	workspaceID uuid.UUID,
	runID uuid.UUID,
	linkID uuid.UUID,
	revokedAt time.Time,
	event collaboration.AuditEvent,
) error {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin public report link revocation: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	var prefix string
	err = transaction.QueryRow(ctx, `
		UPDATE public_report_links SET revoked_at = $4
		WHERE workspace_id = $1 AND run_id = $2 AND id = $3
			AND revoked_at IS NULL
		RETURNING token_prefix
	`, workspaceID, runID, linkID, revokedAt).Scan(&prefix)
	if errors.Is(err, pgx.ErrNoRows) {
		return reports.ErrPublicLinkMissing
	}
	if err != nil {
		return fmt.Errorf("revoke public report link: %w", err)
	}
	event.Metadata["prefix"] = prefix
	if err := insertAuditEvent(ctx, transaction, event); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit public report link revocation: %w", err)
	}
	return nil
}

func (r *ReportsRepository) ResolvePublicLink(
	ctx context.Context,
	tokenHash []byte,
	viewedAt time.Time,
) (reports.ResolvedPublicLink, error) {
	var resolved reports.ResolvedPublicLink
	err := r.pool.QueryRow(ctx, `
		UPDATE public_report_links AS link
		SET last_viewed_at = $2
		FROM workspaces AS workspace
		WHERE link.token_hash = $1
			AND link.workspace_id = workspace.id
			AND link.revoked_at IS NULL
			AND link.expires_at > $2
		RETURNING link.id, link.workspace_id, link.run_id, link.token_prefix,
			link.created_by, link.expires_at, link.revoked_at,
			link.last_viewed_at, link.created_at, workspace.name
	`, tokenHash, viewedAt).Scan(
		&resolved.ID,
		&resolved.WorkspaceID,
		&resolved.RunID,
		&resolved.TokenPrefix,
		&resolved.CreatedBy,
		&resolved.ExpiresAt,
		&resolved.RevokedAt,
		&resolved.LastViewedAt,
		&resolved.CreatedAt,
		&resolved.WorkspaceName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return reports.ResolvedPublicLink{}, reports.ErrPublicLinkMissing
	}
	if err != nil {
		return reports.ResolvedPublicLink{}, fmt.Errorf(
			"resolve public report link: %w",
			err,
		)
	}
	return resolved, nil
}
