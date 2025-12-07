-- Drop tables in reverse order of creation (respecting foreign key constraints)
DROP TABLE IF EXISTS session_messages;
DROP TABLE IF EXISTS workspace_sessions;
DROP TABLE IF EXISTS prompt_tasks;
DROP TABLE IF EXISTS workspace_prep_steps;
DROP TABLE IF EXISTS workspace_secrets;
DROP TABLE IF EXISTS workspaces;

-- Remove worker_id column from vms if it was added
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'vms' AND column_name = 'worker_id') THEN
        DROP INDEX IF EXISTS idx_vms_worker_id;
        ALTER TABLE vms DROP COLUMN worker_id;
    END IF;
END $$;
