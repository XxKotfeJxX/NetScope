-- +goose Up
ALTER TABLE diagnostic_runs
    ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;

ALTER TABLE monitored_targets
    ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE;

CREATE INDEX idx_runs_workspace_created
    ON diagnostic_runs(workspace_id, created_at DESC);
CREATE INDEX idx_targets_workspace_name
    ON monitored_targets(workspace_id, name);

-- Existing v0.3 data remains temporarily unassigned. The first successfully
-- registered owner claims it atomically in the registration transaction.

-- +goose Down
DROP INDEX IF EXISTS idx_targets_workspace_name;
DROP INDEX IF EXISTS idx_runs_workspace_created;
ALTER TABLE monitored_targets DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE diagnostic_runs DROP COLUMN IF EXISTS workspace_id;
