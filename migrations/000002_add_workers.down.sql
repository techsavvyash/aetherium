-- Drop triggers
DROP TRIGGER IF EXISTS update_workers_updated_at ON workers;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Remove worker_id columns from existing tables
ALTER TABLE vms DROP COLUMN IF EXISTS worker_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS worker_id;

-- Drop indices
DROP INDEX IF EXISTS idx_workers_status;
DROP INDEX IF EXISTS idx_workers_zone;
DROP INDEX IF EXISTS idx_workers_last_seen;
DROP INDEX IF EXISTS idx_workers_labels;
DROP INDEX IF EXISTS idx_workers_capabilities;

DROP INDEX IF EXISTS idx_worker_metrics_worker_id;
DROP INDEX IF EXISTS idx_worker_metrics_timestamp;
DROP INDEX IF EXISTS idx_worker_metrics_worker_timestamp;

DROP INDEX IF EXISTS idx_vms_worker_id;
DROP INDEX IF EXISTS idx_tasks_worker_id;

-- Drop tables
DROP TABLE IF EXISTS worker_metrics;
DROP TABLE IF EXISTS workers;
