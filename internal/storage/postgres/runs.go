package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/XxKotfeJxX/netscope/internal/diagnostics"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RunRepository struct {
	pool *pgxpool.Pool
}

func NewRunRepository(pool *pgxpool.Pool) *RunRepository {
	return &RunRepository{pool: pool}
}

func (r *RunRepository) Create(ctx context.Context, run diagnostics.DiagnosticRun) error {
	options, err := json.Marshal(run.Options)
	if err != nil {
		return fmt.Errorf("marshal run options: %w", err)
	}
	checks := make([]string, len(run.RequestedChecks))
	for index, check := range run.RequestedChecks {
		checks[index] = string(check)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO diagnostic_runs (
			id, workspace_id, target_input, normalized_host, normalized_url, status,
			requested_checks, options, summary, created_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, '{}', $9)
	`, run.ID, run.WorkspaceID, run.TargetInput, run.NormalizedHost,
		run.NormalizedURL, run.Status, checks, options, run.CreatedAt)
	if err != nil {
		return fmt.Errorf("create diagnostic run: %w", err)
	}
	return nil
}

func (r *RunRepository) GetByID(ctx context.Context, id uuid.UUID) (diagnostics.DiagnosticRun, error) {
	var run diagnostics.DiagnosticRun
	var checks []string
	var options []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(workspace_id, '00000000-0000-0000-0000-000000000000'),
			target_input, normalized_host, COALESCE(normalized_url, ''),
			status, requested_checks, options, summary, created_at, started_at,
			completed_at, cancelled_at
		FROM diagnostic_runs
		WHERE id = $1
	`, id).Scan(
		&run.ID, &run.WorkspaceID, &run.TargetInput,
		&run.NormalizedHost, &run.NormalizedURL,
		&run.Status, &checks, &options, &run.Summary, &run.CreatedAt,
		&run.StartedAt, &run.CompletedAt, &run.CancelledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return diagnostics.DiagnosticRun{}, diagnostics.ErrRunNotFound
	}
	if err != nil {
		return diagnostics.DiagnosticRun{}, fmt.Errorf("get diagnostic run: %w", err)
	}
	if err := json.Unmarshal(options, &run.Options); err != nil {
		return diagnostics.DiagnosticRun{}, fmt.Errorf("decode run options: %w", err)
	}
	run.RequestedChecks = stringsToChecks(checks)

	results, err := r.resultsByRunID(ctx, id)
	if err != nil {
		return diagnostics.DiagnosticRun{}, err
	}
	run.Results = results
	return run, nil
}

func (r *RunRepository) List(ctx context.Context, filter diagnostics.ListFilter) (diagnostics.Page, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM diagnostic_runs
		 WHERE workspace_id = $1 AND ($2 = '' OR status = $2)`,
		filter.WorkspaceID,
		string(filter.Status),
	).Scan(&total); err != nil {
		return diagnostics.Page{}, fmt.Errorf("count diagnostic runs: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, workspace_id, target_input, normalized_host,
			COALESCE(normalized_url, ''),
			status, requested_checks, options, summary, created_at, started_at,
			completed_at, cancelled_at
		FROM diagnostic_runs
		WHERE workspace_id = $1 AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`, filter.WorkspaceID, string(filter.Status),
		filter.PageSize, (filter.Page-1)*filter.PageSize)
	if err != nil {
		return diagnostics.Page{}, fmt.Errorf("list diagnostic runs: %w", err)
	}
	defer rows.Close()

	items := make([]diagnostics.DiagnosticRun, 0, filter.PageSize)
	for rows.Next() {
		var run diagnostics.DiagnosticRun
		var checks []string
		var options []byte
		if err := rows.Scan(
			&run.ID, &run.WorkspaceID, &run.TargetInput,
			&run.NormalizedHost, &run.NormalizedURL,
			&run.Status, &checks, &options, &run.Summary, &run.CreatedAt,
			&run.StartedAt, &run.CompletedAt, &run.CancelledAt,
		); err != nil {
			return diagnostics.Page{}, fmt.Errorf("scan diagnostic run: %w", err)
		}
		if err := json.Unmarshal(options, &run.Options); err != nil {
			return diagnostics.Page{}, fmt.Errorf("decode run options: %w", err)
		}
		run.RequestedChecks = stringsToChecks(checks)
		run.Results = []diagnostics.CheckResult{}
		items = append(items, run)
	}
	if err := rows.Err(); err != nil {
		return diagnostics.Page{}, fmt.Errorf("iterate diagnostic runs: %w", err)
	}

	return diagnostics.Page{
		Items:      items,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(filter.PageSize))),
	}, nil
}

func (r *RunRepository) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status diagnostics.RunStatus,
) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE diagnostic_runs
		SET status = $2,
			started_at = CASE WHEN $2 = 'running' THEN COALESCE(started_at, NOW()) ELSE started_at END,
			completed_at = CASE WHEN $2 IN ('completed', 'partial', 'failed', 'cancelled', 'interrupted')
				THEN COALESCE(completed_at, NOW()) ELSE completed_at END,
			cancelled_at = CASE WHEN $2 = 'cancelled' THEN COALESCE(cancelled_at, NOW()) ELSE cancelled_at END
		WHERE id = $1
	`, id, status)
	if err != nil {
		return fmt.Errorf("update run status: %w", err)
	}
	if command.RowsAffected() == 0 {
		return diagnostics.ErrRunNotFound
	}
	return nil
}

func (r *RunRepository) SaveResult(ctx context.Context, result diagnostics.CheckResult) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO check_results (
			id, run_id, check_type, status, duration_ms, summary, data,
			error_code, error_message, started_at, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''),
			NULLIF($9, ''), $10, $11)
		ON CONFLICT (run_id, check_type) DO UPDATE SET
			status = EXCLUDED.status,
			duration_ms = EXCLUDED.duration_ms,
			summary = EXCLUDED.summary,
			data = EXCLUDED.data,
			error_code = EXCLUDED.error_code,
			error_message = EXCLUDED.error_message,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at
	`, result.ID, result.RunID, result.Type, result.Status, result.DurationMS,
		result.Summary, result.Data, result.ErrorCode, result.ErrorMessage,
		result.StartedAt, result.CompletedAt)
	if err != nil {
		return fmt.Errorf("save check result: %w", err)
	}
	return nil
}

func (r *RunRepository) resultsByRunID(
	ctx context.Context,
	runID uuid.UUID,
) ([]diagnostics.CheckResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, run_id, check_type, status, COALESCE(duration_ms, 0),
			COALESCE(summary, ''), data, COALESCE(error_code, ''),
			COALESCE(error_message, ''), started_at, completed_at
		FROM check_results
		WHERE run_id = $1
		ORDER BY started_at, check_type
	`, runID)
	if err != nil {
		return nil, fmt.Errorf("list check results: %w", err)
	}
	defer rows.Close()

	results := make([]diagnostics.CheckResult, 0)
	for rows.Next() {
		var result diagnostics.CheckResult
		if err := rows.Scan(
			&result.ID, &result.RunID, &result.Type, &result.Status,
			&result.DurationMS, &result.Summary, &result.Data, &result.ErrorCode,
			&result.ErrorMessage, &result.StartedAt, &result.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan check result: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate check results: %w", err)
	}
	return results, nil
}

func stringsToChecks(values []string) []diagnostics.CheckType {
	checks := make([]diagnostics.CheckType, len(values))
	for index, value := range values {
		checks[index] = diagnostics.CheckType(value)
	}
	return checks
}
