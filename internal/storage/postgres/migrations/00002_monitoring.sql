-- +goose Up
CREATE TABLE monitored_targets (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    tags TEXT[] NOT NULL DEFAULT '{}',
    requested_checks TEXT[] NOT NULL,
    options JSONB NOT NULL DEFAULT '{}',
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds BETWEEN 60 AND 86400),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    failure_threshold INTEGER NOT NULL DEFAULT 3 CHECK (failure_threshold BETWEEN 1 AND 20),
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending',
    last_checked_at TIMESTAMPTZ,
    last_latency_ms BIGINT,
    tls_expires_at TIMESTAMPTZ,
    next_check_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE monitoring_checks (
    id UUID PRIMARY KEY,
    target_id UUID NOT NULL REFERENCES monitored_targets(id) ON DELETE CASCADE,
    run_id UUID NOT NULL UNIQUE REFERENCES diagnostic_runs(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'running',
    latency_ms BIGINT,
    tls_expires_at TIMESTAMPTZ,
    error_message TEXT,
    checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE maintenance_windows (
    id UUID PRIMARY KEY,
    target_id UUID NOT NULL REFERENCES monitored_targets(id) ON DELETE CASCADE,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (ends_at > starts_at)
);

CREATE TABLE notification_channels (
    id UUID PRIMARY KEY,
    target_id UUID NOT NULL REFERENCES monitored_targets(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('email', 'webhook')),
    destination TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_monitored_targets_due
    ON monitored_targets(next_check_at)
    WHERE enabled = TRUE;
CREATE INDEX idx_monitoring_checks_target_checked
    ON monitoring_checks(target_id, checked_at DESC);
CREATE INDEX idx_monitoring_checks_pending
    ON monitoring_checks(created_at)
    WHERE checked_at IS NULL;
CREATE INDEX idx_maintenance_windows_target_time
    ON maintenance_windows(target_id, starts_at, ends_at);
CREATE INDEX idx_notification_channels_target
    ON notification_channels(target_id);

-- +goose Down
DROP TABLE IF EXISTS notification_channels;
DROP TABLE IF EXISTS maintenance_windows;
DROP TABLE IF EXISTS monitoring_checks;
DROP TABLE IF EXISTS monitored_targets;
