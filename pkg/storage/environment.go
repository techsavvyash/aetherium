package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MCPServerType defines the transport type for MCP servers
type MCPServerType string

const (
	// MCPServerTypeStdio is for MCP servers that communicate via stdin/stdout
	MCPServerTypeStdio MCPServerType = "stdio"
	// MCPServerTypeHTTP is for MCP servers that communicate via HTTP
	MCPServerTypeHTTP MCPServerType = "http"
)

// MCPServerConfig defines an MCP server configuration
type MCPServerConfig struct {
	Name string        `json:"name"`
	Type MCPServerType `json:"type"`

	// For stdio type
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`

	// For http type
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// Common - environment variables for the MCP server
	Env map[string]string `json:"env,omitempty"`
}

// Environment represents a reusable workspace template
type Environment struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`

	// VM Configuration
	VCPUs    int `db:"vcpus" json:"vcpus"`
	MemoryMB int `db:"memory_mb" json:"memory_mb"`

	// Repository
	GitRepoURL string `db:"git_repo_url" json:"git_repo_url"`
	GitBranch  string `db:"git_branch" json:"git_branch"`

	// Workspace
	WorkingDirectory string `db:"working_directory" json:"working_directory"`

	// Tools (stored as JSONB array in DB)
	Tools []string `json:"tools"`

	// Environment variables (stored as JSONB object in DB)
	EnvVars map[string]string `json:"env_vars"`

	// MCP Servers (stored as JSONB array in DB)
	MCPServers []MCPServerConfig `json:"mcp_servers"`

	// Idle timeout in seconds before VM is destroyed
	IdleTimeoutSeconds int `db:"idle_timeout_seconds" json:"idle_timeout_seconds"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// EnvironmentRepository defines environment data access operations
type EnvironmentRepository interface {
	// Create creates a new environment
	Create(ctx context.Context, env *Environment) error

	// Get retrieves an environment by ID
	Get(ctx context.Context, id uuid.UUID) (*Environment, error)

	// GetByName retrieves an environment by name
	GetByName(ctx context.Context, name string) (*Environment, error)

	// List retrieves all environments
	List(ctx context.Context) ([]*Environment, error)

	// Update updates an existing environment
	Update(ctx context.Context, env *Environment) error

	// Delete deletes an environment by ID
	Delete(ctx context.Context, id uuid.UUID) error
}
