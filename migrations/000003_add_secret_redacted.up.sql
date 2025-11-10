-- Add secret_redacted column to executions table
ALTER TABLE executions ADD COLUMN secret_redacted BOOLEAN NOT NULL DEFAULT FALSE;

-- Create index for filtering by secret_redacted
CREATE INDEX idx_executions_secret_redacted ON executions(secret_redacted);

-- Purge existing env data for security (user confirmed this is acceptable)
-- This removes any potentially leaked secrets from the database
UPDATE executions SET env = NULL WHERE env IS NOT NULL;

-- Add comment to document the security enhancement
COMMENT ON COLUMN executions.secret_redacted IS 'Indicates if transient secrets were used during execution (not persisted to DB)';
COMMENT ON COLUMN executions.env IS 'Environment variables (persisted). TransientSecrets are never stored here.';
