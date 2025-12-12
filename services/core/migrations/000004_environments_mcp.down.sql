-- Rollback migration: 000004_environments_mcp

-- Drop indexes
DROP INDEX IF EXISTS idx_workspaces_idle;
DROP INDEX IF EXISTS idx_workspaces_environment;
DROP INDEX IF EXISTS idx_environments_name;

-- Remove columns from workspaces
ALTER TABLE workspaces DROP COLUMN IF EXISTS idle_since;
ALTER TABLE workspaces DROP COLUMN IF EXISTS environment_id;

-- Drop environments table
DROP TABLE IF EXISTS environments;
