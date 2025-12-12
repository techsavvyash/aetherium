-- VMs table - tracks virtual machine state
CREATE TABLE vms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    orchestrator VARCHAR(50) NOT NULL, -- 'firecracker' or 'docker'
    status VARCHAR(50) NOT NULL, -- 'created', 'starting', 'running', 'stopping', 'stopped', 'failed'

    -- Configuration
    kernel_path VARCHAR(500),
    rootfs_path VARCHAR(500),
    socket_path VARCHAR(500),
    vcpu_count INTEGER,
    memory_mb INTEGER,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    stopped_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}'::jsonb,

    -- Indexes
    CONSTRAINT vms_name_key UNIQUE (name)
);

CREATE INDEX idx_vms_status ON vms(status);
CREATE INDEX idx_vms_orchestrator ON vms(orchestrator);
CREATE INDEX idx_vms_created_at ON vms(created_at DESC);

-- Tasks table - distributed task queue
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(100) NOT NULL, -- 'vm_create', 'vm_execute', 'vm_delete', etc.
    status VARCHAR(50) NOT NULL DEFAULT 'pending', -- 'pending', 'processing', 'completed', 'failed', 'retrying'
    priority INTEGER NOT NULL DEFAULT 0, -- higher = more important

    -- Task payload
    payload JSONB NOT NULL,
    result JSONB,
    error TEXT,

    -- Execution tracking
    vm_id UUID REFERENCES vms(id) ON DELETE SET NULL,
    worker_id VARCHAR(255), -- ID of worker processing the task
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_count INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_vm_id ON tasks(vm_id);
CREATE INDEX idx_tasks_scheduled_at ON tasks(scheduled_at);
CREATE INDEX idx_tasks_priority_scheduled ON tasks(priority DESC, scheduled_at ASC);

-- Jobs table - execution jobs (collections of commands)
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Job configuration
    vm_id UUID REFERENCES vms(id) ON DELETE CASCADE,
    commands JSONB NOT NULL, -- Array of commands to execute

    -- Results
    results JSONB,
    error TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_vm_id ON jobs(vm_id);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- Executions table - command execution history
CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    vm_id UUID REFERENCES vms(id) ON DELETE CASCADE,

    -- Command details
    command VARCHAR(500) NOT NULL,
    args JSONB,
    env JSONB,

    -- Execution result
    exit_code INTEGER,
    stdout TEXT,
    stderr TEXT,
    error TEXT,

    -- Timestamps
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER, -- milliseconds

    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_executions_job_id ON executions(job_id);
CREATE INDEX idx_executions_vm_id ON executions(vm_id);
CREATE INDEX idx_executions_started_at ON executions(started_at DESC);
