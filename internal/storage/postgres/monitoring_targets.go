package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

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
