-- +goose Up
CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    prefix TEXT NOT NULL UNIQUE,
    token_hash BYTEA NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('operator', 'viewer')),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_workspace_created
    ON api_keys (workspace_id, created_at DESC);

CREATE INDEX idx_api_keys_active_hash
    ON api_keys (token_hash)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS api_keys;
