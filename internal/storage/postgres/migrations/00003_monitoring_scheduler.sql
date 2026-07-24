-- +goose Up
ALTER TABLE monitoring_checks
    ALTER COLUMN run_id DROP NOT NULL;

-- +goose Down
DELETE FROM monitoring_checks WHERE run_id IS NULL;
ALTER TABLE monitoring_checks
    ALTER COLUMN run_id SET NOT NULL;
