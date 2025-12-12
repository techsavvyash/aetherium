-- Workspaces table - extends VMs with AI workspace configuration
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Link to underlying VM
    vm_id UUID REFERENCES vms(id) ON DELETE CASCADE,

    -- Status: creating, preparing, ready, failed, stopped
    status VARCHAR(50) NOT NULL DEFAULT 'creating',

    -- AI Assistant configuration
    ai_assistant VARCHAR(50) NOT NULL DEFAULT 'claude-code', -- 'claude-code' or 'ampcode'
    ai_assistant_config JSONB DEFAULT '{}'::jsonb,

    -- Working directory for prompts
    working_directory VARCHAR(500) DEFAULT '/workspace',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ready_at TIMESTAMP WITH TIME ZONE,
    stopped_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,

    CONSTRAINT workspaces_name_key UNIQUE (name)
);

CREATE INDEX idx_workspaces_status ON workspaces(status);
CREATE INDEX idx_workspaces_vm_id ON workspaces(vm_id);
CREATE INDEX idx_workspaces_ai_assistant ON workspaces(ai_assistant);
CREATE INDEX idx_workspaces_created_at ON workspaces(created_at DESC);

-- Workspace secrets table - encrypted storage for API keys and credentials
CREATE TABLE workspace_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Secret metadata
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Type: 'api_key', 'ssh_key', 'token', 'password', 'other'
    secret_type VARCHAR(50) NOT NULL DEFAULT 'api_key',

    -- Encrypted value (AES-256-GCM)
    encrypted_value BYTEA NOT NULL,

    -- Encryption metadata
    encryption_key_id VARCHAR(255) NOT NULL DEFAULT 'default',
    nonce BYTEA NOT NULL,

    -- Scope: 'workspace' (tied to workspace) or 'global' (shared across workspaces)
    scope VARCHAR(50) NOT NULL DEFAULT 'workspace',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique name per workspace (or globally for global scope)
    CONSTRAINT unique_secret_per_workspace UNIQUE NULLS NOT DISTINCT (workspace_id, name)
);

CREATE INDEX idx_secrets_workspace_id ON workspace_secrets(workspace_id);
CREATE INDEX idx_secrets_scope ON workspace_secrets(scope);
CREATE INDEX idx_secrets_name ON workspace_secrets(name);

-- Preparation steps table - git clones, scripts, env vars
CREATE TABLE workspace_prep_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Step type: 'git_clone', 'script', 'env_var'
    step_type VARCHAR(50) NOT NULL,

    -- Execution order (0-based)
    step_order INTEGER NOT NULL DEFAULT 0,

    -- Step configuration (type-specific)
    -- git_clone: {url, branch, dest_path, ssh_key_secret_id}
    -- script: {content, interpreter, args}
    -- env_var: {key, value, secret_id, is_secret}
    config JSONB NOT NULL,

    -- Execution status: 'pending', 'running', 'completed', 'failed', 'skipped'
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Execution result
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    error TEXT,

    -- Timing
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_prep_steps_workspace_id ON workspace_prep_steps(workspace_id);
CREATE INDEX idx_prep_steps_status ON workspace_prep_steps(status);
CREATE INDEX idx_prep_steps_order ON workspace_prep_steps(workspace_id, step_order);

-- Prompt tasks table - queued prompts for execution
CREATE TABLE prompt_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Prompt content
    prompt TEXT NOT NULL,
    system_prompt TEXT,

    -- Execution context
    working_directory VARCHAR(500),
    environment JSONB DEFAULT '{}'::jsonb,

    -- Queue management
    priority INTEGER NOT NULL DEFAULT 5, -- 0-10, higher = more important
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed, cancelled

    -- Execution result
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    error TEXT,

    -- Timing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,

    -- Metadata (e.g., source: 'dashboard', 'api', 'webhook')
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_prompt_tasks_workspace_id ON prompt_tasks(workspace_id);
CREATE INDEX idx_prompt_tasks_status ON prompt_tasks(status);
CREATE INDEX idx_prompt_tasks_priority_scheduled ON prompt_tasks(workspace_id, priority DESC, scheduled_at ASC);
CREATE INDEX idx_prompt_tasks_created_at ON prompt_tasks(created_at DESC);

-- Interactive sessions table - WebSocket session tracking
CREATE TABLE workspace_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,

    -- Session status: 'active', 'disconnected', 'terminated'
    status VARCHAR(50) NOT NULL DEFAULT 'active',

    -- Client info
    client_ip VARCHAR(45),
    user_agent TEXT,

    -- Connection times
    connected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMP WITH TIME ZONE,

    -- Session data
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_sessions_workspace_id ON workspace_sessions(workspace_id);
CREATE INDEX idx_sessions_status ON workspace_sessions(status);
CREATE INDEX idx_sessions_connected_at ON workspace_sessions(connected_at DESC);

-- Session messages table - conversation history for interactive sessions
CREATE TABLE session_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES workspace_sessions(id) ON DELETE CASCADE,

    -- Message type: 'prompt', 'response', 'system', 'error'
    message_type VARCHAR(50) NOT NULL,

    -- Content
    content TEXT NOT NULL,

    -- For responses: execution details
    exit_code INTEGER,

    -- Timing
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_messages_session_id ON session_messages(session_id);
CREATE INDEX idx_messages_created_at ON session_messages(session_id, created_at);

-- Add worker_id to vms table if not exists (for associating VMs with workers)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'vms' AND column_name = 'worker_id') THEN
        ALTER TABLE vms ADD COLUMN worker_id VARCHAR(255);
        CREATE INDEX idx_vms_worker_id ON vms(worker_id);
    END IF;
END $$;
