-- Create secrets table for storing encrypted secret metadata
CREATE TABLE secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Secret identification
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,

    -- Encrypted value (encrypted by Vault, stored as base64)
    encrypted_value TEXT NOT NULL,

    -- Encryption metadata
    encryption_key_id VARCHAR(100) NOT NULL DEFAULT 'aetherium',
    encryption_version INTEGER NOT NULL DEFAULT 1,

    -- Scoping (global, vm-specific, user-specific)
    scope VARCHAR(50) NOT NULL DEFAULT 'global',
    scope_id UUID,  -- VM ID, User ID, etc.

    -- Lifecycle
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    rotated_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    access_count INTEGER DEFAULT 0,

    -- Metadata
    tags JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Constraints
    CONSTRAINT check_scope_id CHECK (
        (scope = 'global' AND scope_id IS NULL) OR
        (scope != 'global' AND scope_id IS NOT NULL)
    )
);

-- Indexes for efficient queries
CREATE INDEX idx_secrets_name ON secrets(name);
CREATE INDEX idx_secrets_scope ON secrets(scope, scope_id);
CREATE INDEX idx_secrets_created_at ON secrets(created_at DESC);
CREATE INDEX idx_secrets_expires_at ON secrets(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_secrets_tags ON secrets USING GIN (tags);

-- Create audit log table for secret access
CREATE TABLE secret_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_id UUID NOT NULL REFERENCES secrets(id) ON DELETE CASCADE,
    secret_name VARCHAR(255) NOT NULL,

    -- Action details
    action VARCHAR(50) NOT NULL,  -- 'create', 'read', 'update', 'delete', 'rotate'
    actor VARCHAR(255),           -- User/service that performed the action
    actor_ip INET,

    -- Context
    vm_id UUID,                   -- If secret was used in VM execution
    execution_id UUID,            -- If secret was used in specific execution

    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Metadata
    metadata JSONB DEFAULT '{}'
);

-- Indexes for audit log queries
CREATE INDEX idx_secret_audit_logs_secret_id ON secret_audit_logs(secret_id);
CREATE INDEX idx_secret_audit_logs_timestamp ON secret_audit_logs(timestamp DESC);
CREATE INDEX idx_secret_audit_logs_action ON secret_audit_logs(action);
CREATE INDEX idx_secret_audit_logs_actor ON secret_audit_logs(actor);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_secrets_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update updated_at
CREATE TRIGGER secrets_updated_at
    BEFORE UPDATE ON secrets
    FOR EACH ROW
    EXECUTE FUNCTION update_secrets_updated_at();

-- Comments for documentation
COMMENT ON TABLE secrets IS 'Stores encrypted secret metadata. Actual secrets are encrypted by Vault.';
COMMENT ON COLUMN secrets.encrypted_value IS 'Base64-encoded ciphertext from Vault transit encryption';
COMMENT ON COLUMN secrets.encryption_key_id IS 'Vault transit key name used for encryption';
COMMENT ON COLUMN secrets.scope IS 'Scope of the secret: global, vm, user, etc.';
COMMENT ON COLUMN secrets.access_count IS 'Number of times this secret has been accessed';
COMMENT ON TABLE secret_audit_logs IS 'Audit trail for all secret operations';
