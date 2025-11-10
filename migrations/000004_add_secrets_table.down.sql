-- Drop trigger
DROP TRIGGER IF EXISTS secrets_updated_at ON secrets;

-- Drop function
DROP FUNCTION IF EXISTS update_secrets_updated_at();

-- Drop indexes
DROP INDEX IF EXISTS idx_secret_audit_logs_actor;
DROP INDEX IF EXISTS idx_secret_audit_logs_action;
DROP INDEX IF EXISTS idx_secret_audit_logs_timestamp;
DROP INDEX IF EXISTS idx_secret_audit_logs_secret_id;
DROP INDEX IF EXISTS idx_secrets_tags;
DROP INDEX IF EXISTS idx_secrets_expires_at;
DROP INDEX IF EXISTS idx_secrets_created_at;
DROP INDEX IF EXISTS idx_secrets_scope;
DROP INDEX IF EXISTS idx_secrets_name;

-- Drop tables
DROP TABLE IF EXISTS secret_audit_logs;
DROP TABLE IF EXISTS secrets;
