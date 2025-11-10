-- Remove index
DROP INDEX IF EXISTS idx_executions_secret_redacted;

-- Remove secret_redacted column
ALTER TABLE executions DROP COLUMN IF EXISTS secret_redacted;

-- Note: Cannot restore purged env data (this is intentional for security)
