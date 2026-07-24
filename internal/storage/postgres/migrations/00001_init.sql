-- +goose Up
CREATE TABLE diagnostic_runs (
    id UUID PRIMARY KEY,
    target_input TEXT NOT NULL,
    normalized_host TEXT NOT NULL,
    normalized_url TEXT,
    status TEXT NOT NULL,
    requested_checks TEXT[] NOT NULL,
    options JSONB NOT NULL DEFAULT '{}',
    summary JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ
);

CREATE TABLE check_results (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES diagnostic_runs(id) ON DELETE CASCADE,
    check_type TEXT NOT NULL,
    status TEXT NOT NULL,
    duration_ms BIGINT,
    summary TEXT,
    data JSONB NOT NULL DEFAULT '{}',
    error_code TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    UNIQUE (run_id, check_type)
);

CREATE INDEX idx_runs_created_at ON diagnostic_runs(created_at DESC);
CREATE INDEX idx_runs_status ON diagnostic_runs(status);
CREATE INDEX idx_results_run_id ON check_results(run_id);

-- +goose Down
DROP TABLE IF EXISTS check_results;
DROP TABLE IF EXISTS diagnostic_runs;

