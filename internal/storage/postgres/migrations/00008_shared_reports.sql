-- +goose Up
CREATE TABLE run_comments (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    run_id UUID NOT NULL REFERENCES diagnostic_runs(id) ON DELETE CASCADE,
    author_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    body TEXT NOT NULL CHECK (char_length(body) BETWEEN 1 AND 2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_run_comments_workspace_run_created
    ON run_comments (workspace_id, run_id, created_at, id);

CREATE TABLE public_report_links (
    id UUID PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    run_id UUID NOT NULL REFERENCES diagnostic_runs(id) ON DELETE CASCADE,
    token_prefix TEXT NOT NULL UNIQUE,
    token_hash BYTEA NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    last_viewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_public_report_links_workspace_run
    ON public_report_links (workspace_id, run_id, created_at DESC);

CREATE INDEX idx_public_report_links_active_hash
    ON public_report_links (token_hash)
    WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS public_report_links;
DROP TABLE IF EXISTS run_comments;
