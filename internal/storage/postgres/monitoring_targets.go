package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/XxKotfeJxX/netscope/internal/monitoring"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitoringRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringRepository(pool *pgxpool.Pool) *MonitoringRepository {
	return &MonitoringRepository{pool: pool}
}

func (r *MonitoringRepository) CreateTarget(
	ctx context.Context,
	target monitoring.Target,
) error {
	options, checks, err := monitoringFields(target)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO monitored_targets (
			id, name, address, tags, requested_checks, options, interval_seconds,
			enabled, failure_threshold, status, next_check_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, target.ID, target.Name, target.Address, target.Tags, checks, options,
		target.IntervalSeconds, target.Enabled, target.FailureThreshold,
		target.Status, target.NextCheckAt, target.CreatedAt, target.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create monitored target: %w", err)
	}
	return nil
}

func (r *MonitoringRepository) GetTarget(
	ctx context.Context,
	id uuid.UUID,
) (monitoring.Target, error) {
	target, err := scanMonitoredTarget(r.pool.QueryRow(ctx, `
		SELECT id, name, address, tags, requested_checks, options,
			interval_seconds, enabled, failure_threshold, consecutive_failures,
			status, last_checked_at, last_latency_ms, tls_expires_at,
			next_check_at, created_at, updated_at
		FROM monitored_targets
		WHERE id = $1
	`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return monitoring.Target{}, monitoring.ErrTargetNotFound
	}
	if err != nil {
		return monitoring.Target{}, fmt.Errorf("get monitored target: %w", err)
	}
	return target, nil
}

func (r *MonitoringRepository) ListTargets(
	ctx context.Context,
	page int,
	pageSize int,
) (monitoring.Page, error) {
	page, pageSize = monitoringPagination(page, pageSize)
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM monitored_targets`).Scan(&total); err != nil {
		return monitoring.Page{}, fmt.Errorf("count monitored targets: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, address, tags, requested_checks, options,
			interval_seconds, enabled, failure_threshold, consecutive_failures,
			status, last_checked_at, last_latency_ms, tls_expires_at,
			next_check_at, created_at, updated_at
		FROM monitored_targets
		ORDER BY name, created_at
		LIMIT $1 OFFSET $2
	`, pageSize, (page-1)*pageSize)
	if err != nil {
		return monitoring.Page{}, fmt.Errorf("list monitored targets: %w", err)
	}
	defer rows.Close()

	items := make([]monitoring.Target, 0, pageSize)
	for rows.Next() {
		target, err := scanMonitoredTarget(rows)
		if err != nil {
			return monitoring.Page{}, fmt.Errorf("scan monitored target: %w", err)
		}
		items = append(items, target)
	}
	if err := rows.Err(); err != nil {
		return monitoring.Page{}, fmt.Errorf("iterate monitored targets: %w", err)
	}
	return monitoring.Page{
		Items: items, Page: page, PageSize: pageSize, TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

func (r *MonitoringRepository) UpdateTarget(
	ctx context.Context,
	target monitoring.Target,
) error {
	options, checks, err := monitoringFields(target)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `
		UPDATE monitored_targets
		SET name = $2, address = $3, tags = $4, requested_checks = $5,
			options = $6, interval_seconds = $7, failure_threshold = $8,
			next_check_at = LEAST(next_check_at, NOW()), updated_at = $9
		WHERE id = $1
	`, target.ID, target.Name, target.Address, target.Tags, checks, options,
		target.IntervalSeconds, target.FailureThreshold, target.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update monitored target: %w", err)
	}
	if command.RowsAffected() == 0 {
		return monitoring.ErrTargetNotFound
	}
	return nil
}

func (r *MonitoringRepository) DeleteTarget(ctx context.Context, id uuid.UUID) error {
	command, err := r.pool.Exec(ctx, `DELETE FROM monitored_targets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete monitored target: %w", err)
	}
	if command.RowsAffected() == 0 {
		return monitoring.ErrTargetNotFound
	}
	return nil
}

func (r *MonitoringRepository) SetTargetEnabled(
	ctx context.Context,
	id uuid.UUID,
	enabled bool,
) error {
	command, err := r.pool.Exec(ctx, `
		UPDATE monitored_targets
		SET enabled = $2,
			status = CASE WHEN $2 THEN 'pending' ELSE 'paused' END,
			next_check_at = CASE WHEN $2 THEN NOW() ELSE next_check_at END,
			updated_at = NOW()
		WHERE id = $1
	`, id, enabled)
	if err != nil {
		return fmt.Errorf("set monitored target state: %w", err)
	}
	if command.RowsAffected() == 0 {
		return monitoring.ErrTargetNotFound
	}
	return nil
}

func (r *MonitoringRepository) ListChecks(
	ctx context.Context,
	targetID uuid.UUID,
	page int,
	pageSize int,
) (monitoring.CheckPage, error) {
	page, pageSize = monitoringPagination(page, pageSize)
	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM monitoring_checks WHERE target_id = $1`,
		targetID,
	).Scan(&total); err != nil {
		return monitoring.CheckPage{}, fmt.Errorf("count monitoring checks: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_id, run_id, status, latency_ms, tls_expires_at,
			COALESCE(error_message, ''), checked_at, created_at
		FROM monitoring_checks
		WHERE target_id = $1
		ORDER BY COALESCE(checked_at, created_at) DESC
		LIMIT $2 OFFSET $3
	`, targetID, pageSize, (page-1)*pageSize)
	if err != nil {
		return monitoring.CheckPage{}, fmt.Errorf("list monitoring checks: %w", err)
	}
	defer rows.Close()
	items := make([]monitoring.Check, 0, pageSize)
	for rows.Next() {
		var check monitoring.Check
		if err := rows.Scan(
			&check.ID, &check.TargetID, &check.RunID, &check.Status,
			&check.LatencyMS, &check.TLSExpiresAt, &check.ErrorMessage,
			&check.CheckedAt, &check.CreatedAt,
		); err != nil {
			return monitoring.CheckPage{}, fmt.Errorf("scan monitoring check: %w", err)
		}
		items = append(items, check)
	}
	if err := rows.Err(); err != nil {
		return monitoring.CheckPage{}, fmt.Errorf("iterate monitoring checks: %w", err)
	}
	return monitoring.CheckPage{
		Items: items, Page: page, PageSize: pageSize, TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(pageSize))),
	}, nil
}

func (r *MonitoringRepository) CreateMaintenanceWindow(
	ctx context.Context,
	window monitoring.MaintenanceWindow,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO maintenance_windows (
			id, target_id, starts_at, ends_at, reason, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, window.ID, window.TargetID, window.StartsAt, window.EndsAt,
		window.Reason, window.CreatedAt)
	if err != nil {
		return fmt.Errorf("create maintenance window: %w", err)
	}
	return nil
}

func (r *MonitoringRepository) ListMaintenanceWindows(
	ctx context.Context,
	targetID uuid.UUID,
) ([]monitoring.MaintenanceWindow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_id, starts_at, ends_at, reason, created_at
		FROM maintenance_windows
		WHERE target_id = $1
		ORDER BY starts_at DESC
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("list maintenance windows: %w", err)
	}
	defer rows.Close()
	items := make([]monitoring.MaintenanceWindow, 0)
	for rows.Next() {
		var window monitoring.MaintenanceWindow
		if err := rows.Scan(
			&window.ID, &window.TargetID, &window.StartsAt, &window.EndsAt,
			&window.Reason, &window.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan maintenance window: %w", err)
		}
		items = append(items, window)
	}
	return items, rows.Err()
}

func (r *MonitoringRepository) DeleteMaintenanceWindow(
	ctx context.Context,
	targetID uuid.UUID,
	id uuid.UUID,
) error {
	command, err := r.pool.Exec(ctx,
		`DELETE FROM maintenance_windows WHERE id = $1 AND target_id = $2`,
		id, targetID,
	)
	if err != nil {
		return fmt.Errorf("delete maintenance window: %w", err)
	}
	if command.RowsAffected() == 0 {
		return monitoring.ErrTargetNotFound
	}
	return nil
}

func (r *MonitoringRepository) CreateNotificationChannel(
	ctx context.Context,
	channel monitoring.NotificationChannel,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_channels (
			id, target_id, kind, destination, enabled, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`, channel.ID, channel.TargetID, channel.Kind, channel.Destination,
		channel.Enabled, channel.CreatedAt)
	if err != nil {
		return fmt.Errorf("create notification channel: %w", err)
	}
	return nil
}

func (r *MonitoringRepository) ListNotificationChannels(
	ctx context.Context,
	targetID uuid.UUID,
) ([]monitoring.NotificationChannel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_id, kind, destination, enabled, created_at
		FROM notification_channels
		WHERE target_id = $1
		ORDER BY created_at
	`, targetID)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()
	items := make([]monitoring.NotificationChannel, 0)
	for rows.Next() {
		var channel monitoring.NotificationChannel
		if err := rows.Scan(
			&channel.ID, &channel.TargetID, &channel.Kind,
			&channel.Destination, &channel.Enabled, &channel.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification channel: %w", err)
		}
		items = append(items, channel)
	}
	return items, rows.Err()
}

func (r *MonitoringRepository) DeleteNotificationChannel(
	ctx context.Context,
	targetID uuid.UUID,
	id uuid.UUID,
) error {
	command, err := r.pool.Exec(ctx,
		`DELETE FROM notification_channels WHERE id = $1 AND target_id = $2`,
		id, targetID,
	)
	if err != nil {
		return fmt.Errorf("delete notification channel: %w", err)
	}
	if command.RowsAffected() == 0 {
		return monitoring.ErrTargetNotFound
	}
	return nil
}

func (r *MonitoringRepository) ClaimDueTargets(
	ctx context.Context,
	limit int,
) ([]monitoring.Target, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if _, err := r.pool.Exec(ctx, `
		UPDATE monitored_targets AS target
		SET status = 'maintenance',
			next_check_at = active_window.ends_at,
			updated_at = NOW()
		FROM (
			SELECT target_id, MAX(ends_at) AS ends_at
			FROM maintenance_windows
			WHERE starts_at <= NOW() AND ends_at > NOW()
			GROUP BY target_id
		) AS active_window
		WHERE target.id = active_window.target_id
			AND target.enabled = TRUE
			AND target.next_check_at <= NOW()
	`); err != nil {
		return nil, fmt.Errorf("mark targets in maintenance: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		WITH due AS (
			SELECT target.id
			FROM monitored_targets AS target
			WHERE target.enabled = TRUE
				AND target.next_check_at <= NOW()
				AND NOT EXISTS (
					SELECT 1 FROM monitoring_checks AS check_run
					WHERE check_run.target_id = target.id
						AND check_run.checked_at IS NULL
				)
				AND NOT EXISTS (
					SELECT 1 FROM maintenance_windows AS maintenance
					WHERE maintenance.target_id = target.id
						AND maintenance.starts_at <= NOW()
						AND maintenance.ends_at > NOW()
				)
			ORDER BY target.next_check_at
			FOR UPDATE SKIP LOCKED
			LIMIT $1
		)
		UPDATE monitored_targets AS target
		SET next_check_at = NOW() + make_interval(secs => target.interval_seconds),
			status = CASE WHEN target.status = 'maintenance' THEN 'pending' ELSE target.status END,
			updated_at = NOW()
		FROM due
		WHERE target.id = due.id
		RETURNING target.id, target.name, target.address, target.tags,
			target.requested_checks, target.options, target.interval_seconds,
			target.enabled, target.failure_threshold, target.consecutive_failures,
			target.status, target.last_checked_at, target.last_latency_ms,
			target.tls_expires_at, target.next_check_at, target.created_at,
			target.updated_at
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim due monitored targets: %w", err)
	}
	defer rows.Close()
	items := make([]monitoring.Target, 0, limit)
	for rows.Next() {
		target, err := scanMonitoredTarget(rows)
		if err != nil {
			return nil, fmt.Errorf("scan due monitored target: %w", err)
		}
		items = append(items, target)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due monitored targets: %w", err)
	}
	return items, nil
}

func (r *MonitoringRepository) CreateScheduledCheck(
	ctx context.Context,
	targetID uuid.UUID,
	runID uuid.UUID,
) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO monitoring_checks (id, target_id, run_id, status)
		VALUES ($1, $2, $3, 'pending')
	`, uuid.New(), targetID, runID)
	if err != nil {
		return fmt.Errorf("create scheduled monitoring check: %w", err)
	}
	return nil
}

func (r *MonitoringRepository) ListPendingChecks(
	ctx context.Context,
	limit int,
) ([]monitoring.Check, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, target_id, run_id, status, latency_ms, tls_expires_at,
			COALESCE(error_message, ''), checked_at, created_at
		FROM monitoring_checks
		WHERE checked_at IS NULL AND run_id IS NOT NULL
		ORDER BY created_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending monitoring checks: %w", err)
	}
	defer rows.Close()
	items := make([]monitoring.Check, 0, limit)
	for rows.Next() {
		var check monitoring.Check
		if err := rows.Scan(
			&check.ID, &check.TargetID, &check.RunID, &check.Status,
			&check.LatencyMS, &check.TLSExpiresAt, &check.ErrorMessage,
			&check.CheckedAt, &check.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan pending monitoring check: %w", err)
		}
		items = append(items, check)
	}
	return items, rows.Err()
}

func (r *MonitoringRepository) CompleteCheck(
	ctx context.Context,
	check monitoring.Check,
) (monitoring.Target, bool, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return monitoring.Target{}, false, fmt.Errorf("begin monitoring completion: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	previous, err := scanMonitoredTarget(transaction.QueryRow(ctx, `
		SELECT id, name, address, tags, requested_checks, options,
			interval_seconds, enabled, failure_threshold, consecutive_failures,
			status, last_checked_at, last_latency_ms, tls_expires_at,
			next_check_at, created_at, updated_at
		FROM monitored_targets WHERE id = $1 FOR UPDATE
	`, check.TargetID))
	if errors.Is(err, pgx.ErrNoRows) {
		return monitoring.Target{}, false, monitoring.ErrTargetNotFound
	}
	if err != nil {
		return monitoring.Target{}, false, fmt.Errorf("lock monitored target: %w", err)
	}

	checkedAt := check.CheckedAt
	if checkedAt == nil {
		now := time.Now().UTC()
		checkedAt = &now
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE monitoring_checks
		SET status = $2, latency_ms = $3, tls_expires_at = $4,
			error_message = NULLIF($5, ''), checked_at = $6
		WHERE id = $1
	`, check.ID, check.Status, check.LatencyMS, check.TLSExpiresAt,
		check.ErrorMessage, checkedAt); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("complete monitoring check: %w", err)
	}

	failures := 0
	status := check.Status
	if check.Status == monitoring.StatusUnavailable {
		failures = previous.ConsecutiveFailures + 1
		if failures < previous.FailureThreshold {
			status = monitoring.StatusWarning
		}
	}
	notify := (previous.Status != monitoring.StatusUnavailable &&
		status == monitoring.StatusUnavailable) ||
		(previous.Status == monitoring.StatusUnavailable &&
			status != monitoring.StatusUnavailable)
	if _, err := transaction.Exec(ctx, `
		UPDATE monitored_targets
		SET consecutive_failures = $2, status = $3, last_checked_at = $4,
			last_latency_ms = $5,
			tls_expires_at = COALESCE($6, tls_expires_at),
			updated_at = NOW()
		WHERE id = $1
	`, check.TargetID, failures, status, checkedAt, check.LatencyMS,
		check.TLSExpiresAt); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("update monitored target result: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("commit monitoring completion: %w", err)
	}
	updated, err := r.GetTarget(ctx, check.TargetID)
	return updated, notify, err
}

func (r *MonitoringRepository) RecordDispatchFailure(
	ctx context.Context,
	targetID uuid.UUID,
	message string,
) (monitoring.Target, bool, error) {
	transaction, err := r.pool.Begin(ctx)
	if err != nil {
		return monitoring.Target{}, false, fmt.Errorf("begin dispatch failure: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()
	previous, err := scanMonitoredTarget(transaction.QueryRow(ctx, `
		SELECT id, name, address, tags, requested_checks, options,
			interval_seconds, enabled, failure_threshold, consecutive_failures,
			status, last_checked_at, last_latency_ms, tls_expires_at,
			next_check_at, created_at, updated_at
		FROM monitored_targets WHERE id = $1 FOR UPDATE
	`, targetID))
	if err != nil {
		return monitoring.Target{}, false, fmt.Errorf("lock dispatch target: %w", err)
	}
	now := time.Now().UTC()
	failures := previous.ConsecutiveFailures + 1
	status := monitoring.StatusWarning
	if failures >= previous.FailureThreshold {
		status = monitoring.StatusUnavailable
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO monitoring_checks (
			id, target_id, status, error_message, checked_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $5)
	`, uuid.New(), targetID, status, message, now); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("record dispatch failure: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		UPDATE monitored_targets
		SET consecutive_failures = $2, status = $3, last_checked_at = $4,
			last_latency_ms = NULL, updated_at = NOW()
		WHERE id = $1
	`, targetID, failures, status, now); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("update dispatch failure: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return monitoring.Target{}, false, fmt.Errorf("commit dispatch failure: %w", err)
	}
	updated, err := r.GetTarget(ctx, targetID)
	return updated, previous.Status != monitoring.StatusUnavailable &&
		status == monitoring.StatusUnavailable, err
}

type rowScanner interface {
	Scan(...any) error
}

func scanMonitoredTarget(scanner rowScanner) (monitoring.Target, error) {
	var target monitoring.Target
	var checks []string
	var options []byte
	err := scanner.Scan(
		&target.ID, &target.Name, &target.Address, &target.Tags, &checks, &options,
		&target.IntervalSeconds, &target.Enabled, &target.FailureThreshold,
		&target.ConsecutiveFailures, &target.Status, &target.LastCheckedAt,
		&target.LastLatencyMS, &target.TLSExpiresAt, &target.NextCheckAt,
		&target.CreatedAt, &target.UpdatedAt,
	)
	if err != nil {
		return monitoring.Target{}, err
	}
	if err := json.Unmarshal(options, &target.Options); err != nil {
		return monitoring.Target{}, fmt.Errorf("decode monitoring options: %w", err)
	}
	target.Checks = stringsToChecks(checks)
	return target, nil
}

func monitoringFields(target monitoring.Target) ([]byte, []string, error) {
	options, err := json.Marshal(target.Options)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal monitoring options: %w", err)
	}
	checks := make([]string, len(target.Checks))
	for index, check := range target.Checks {
		checks[index] = string(check)
	}
	return options, checks, nil
}

func monitoringPagination(page int, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
