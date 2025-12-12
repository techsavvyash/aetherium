-- Add workers table for tracking distributed workers
CREATE TABLE IF NOT EXISTS workers (
    id VARCHAR(255) PRIMARY KEY,
    hostname VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Location
    zone VARCHAR(100),
    labels JSONB DEFAULT '{}',

    -- Capabilities
    capabilities JSONB DEFAULT '[]',

    -- Resources
    cpu_cores INTEGER NOT NULL DEFAULT 0,
    memory_mb BIGINT NOT NULL DEFAULT 0,
    disk_gb BIGINT NOT NULL DEFAULT 0,

    -- Resource usage
    used_cpu_cores INTEGER NOT NULL DEFAULT 0,
    used_memory_mb BIGINT NOT NULL DEFAULT 0,
    used_disk_gb BIGINT NOT NULL DEFAULT 0,

    -- VM tracking
    vm_count INTEGER NOT NULL DEFAULT 0,
    max_vms INTEGER NOT NULL DEFAULT 100,

    -- Metadata
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add worker_metrics table for historical tracking
CREATE TABLE IF NOT EXISTS worker_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    worker_id VARCHAR(255) NOT NULL REFERENCES workers(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Resource metrics
    cpu_usage FLOAT NOT NULL DEFAULT 0,
    memory_usage FLOAT NOT NULL DEFAULT 0,
    disk_usage FLOAT NOT NULL DEFAULT 0,

    -- VM metrics
    vm_count INTEGER NOT NULL DEFAULT 0,
    tasks_processed INTEGER NOT NULL DEFAULT 0,

    -- Network metrics (optional)
    network_in_mb FLOAT DEFAULT 0,
    network_out_mb FLOAT DEFAULT 0,

    metadata JSONB DEFAULT '{}'
);

-- Add indices for efficient queries
CREATE INDEX idx_workers_status ON workers(status);
CREATE INDEX idx_workers_zone ON workers(zone);
CREATE INDEX idx_workers_last_seen ON workers(last_seen);
CREATE INDEX idx_workers_labels ON workers USING GIN(labels);
CREATE INDEX idx_workers_capabilities ON workers USING GIN(capabilities);

CREATE INDEX idx_worker_metrics_worker_id ON worker_metrics(worker_id);
CREATE INDEX idx_worker_metrics_timestamp ON worker_metrics(timestamp);
CREATE INDEX idx_worker_metrics_worker_timestamp ON worker_metrics(worker_id, timestamp);

-- Add worker_id column to vms table if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'vms' AND column_name = 'worker_id'
    ) THEN
        ALTER TABLE vms ADD COLUMN worker_id VARCHAR(255) REFERENCES workers(id) ON DELETE SET NULL;
        CREATE INDEX idx_vms_worker_id ON vms(worker_id);
    END IF;
END $$;

-- Add worker_id column to tasks table if it doesn't exist and make it a foreign key
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'tasks' AND column_name = 'worker_id'
    ) THEN
        -- Drop existing worker_id if it's not a foreign key
        ALTER TABLE tasks DROP COLUMN worker_id;
    END IF;

    ALTER TABLE tasks ADD COLUMN worker_id VARCHAR(255) REFERENCES workers(id) ON DELETE SET NULL;
    CREATE INDEX idx_tasks_worker_id ON tasks(worker_id);
END $$;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for workers table
CREATE TRIGGER update_workers_updated_at
    BEFORE UPDATE ON workers
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
