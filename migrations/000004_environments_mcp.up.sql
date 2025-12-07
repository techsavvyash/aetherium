-- Migration: 000004_environments_mcp
-- Description: Add environments table for reusable workspace templates with MCP server support

-- Environments: Reusable templates for workspaces
CREATE TABLE environments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,

    -- VM Configuration
    vcpus INTEGER DEFAULT 2,
    memory_mb INTEGER DEFAULT 2048,

    -- Repository to clone
    git_repo_url VARCHAR(500),
    git_branch VARCHAR(255) DEFAULT 'main',

    -- Working directory inside VM
    working_directory VARCHAR(500) DEFAULT '/workspace',

    -- Tools to install (JSON array, e.g., ["git", "nodejs@20", "claude-code"])
    tools JSONB DEFAULT '["git", "nodejs@20", "claude-code"]',

    -- Environment variables (JSON object)
    env_vars JSONB DEFAULT '{}',

    -- MCP server configurations (JSON array)
    -- Schema: [{"name": "playwright", "type": "stdio", "command": "npx", "args": ["@playwright/mcp@latest"], "env": {}}]
    mcp_servers JSONB DEFAULT '[]',

    -- Idle timeout before VM destruction (seconds), default 30 minutes
    idle_timeout_seconds INTEGER DEFAULT 1800,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT environments_name_unique UNIQUE (name)
);

-- Add environment_id and idle_since to workspaces table
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS environment_id UUID REFERENCES environments(id) ON DELETE SET NULL;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS idle_since TIMESTAMP WITH TIME ZONE;

-- Indexes for efficient queries
CREATE INDEX idx_environments_name ON environments(name);
CREATE INDEX idx_workspaces_environment ON workspaces(environment_id) WHERE environment_id IS NOT NULL;
CREATE INDEX idx_workspaces_idle ON workspaces(idle_since) WHERE status = 'ready' AND idle_since IS NOT NULL;

-- Grant permissions to aetherium user
GRANT ALL PRIVILEGES ON TABLE environments TO aetherium;
