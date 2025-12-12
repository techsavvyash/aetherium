package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// environmentRepository implements storage.EnvironmentRepository
type environmentRepository struct {
	db *sqlx.DB
}

// environmentRow represents a database row for environments
type environmentRow struct {
	ID                 uuid.UUID      `db:"id"`
	Name               string         `db:"name"`
	Description        sql.NullString `db:"description"`
	VCPUs              int            `db:"vcpus"`
	MemoryMB           int            `db:"memory_mb"`
	GitRepoURL         sql.NullString `db:"git_repo_url"`
	GitBranch          string         `db:"git_branch"`
	WorkingDirectory   string         `db:"working_directory"`
	Tools              []byte         `db:"tools"`
	EnvVars            []byte         `db:"env_vars"`
	MCPServers         []byte         `db:"mcp_servers"`
	IdleTimeoutSeconds int            `db:"idle_timeout_seconds"`
	CreatedAt          time.Time      `db:"created_at"`
	UpdatedAt          time.Time      `db:"updated_at"`
}

// toEnvironment converts a database row to an Environment
func (r *environmentRow) toEnvironment() (*storage.Environment, error) {
	env := &storage.Environment{
		ID:                 r.ID,
		Name:               r.Name,
		VCPUs:              r.VCPUs,
		MemoryMB:           r.MemoryMB,
		GitBranch:          r.GitBranch,
		WorkingDirectory:   r.WorkingDirectory,
		IdleTimeoutSeconds: r.IdleTimeoutSeconds,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}

	if r.Description.Valid {
		env.Description = &r.Description.String
	}

	if r.GitRepoURL.Valid {
		env.GitRepoURL = r.GitRepoURL.String
	}

	// Parse tools JSON array
	if len(r.Tools) > 0 {
		if err := json.Unmarshal(r.Tools, &env.Tools); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
		}
	} else {
		env.Tools = []string{}
	}

	// Parse env_vars JSON object
	if len(r.EnvVars) > 0 {
		if err := json.Unmarshal(r.EnvVars, &env.EnvVars); err != nil {
			return nil, fmt.Errorf("failed to unmarshal env_vars: %w", err)
		}
	} else {
		env.EnvVars = make(map[string]string)
	}

	// Parse mcp_servers JSON array
	if len(r.MCPServers) > 0 {
		if err := json.Unmarshal(r.MCPServers, &env.MCPServers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal mcp_servers: %w", err)
		}
	} else {
		env.MCPServers = []storage.MCPServerConfig{}
	}

	return env, nil
}

// Create creates a new environment
func (r *environmentRepository) Create(ctx context.Context, env *storage.Environment) error {
	if env.ID == uuid.Nil {
		env.ID = uuid.New()
	}

	// Marshal JSON fields with defaults
	toolsJSON := []byte("[]")
	if env.Tools != nil && len(env.Tools) > 0 {
		var err error
		toolsJSON, err = json.Marshal(env.Tools)
		if err != nil {
			return fmt.Errorf("failed to marshal tools: %w", err)
		}
	}

	envVarsJSON := []byte("{}")
	if env.EnvVars != nil && len(env.EnvVars) > 0 {
		var err error
		envVarsJSON, err = json.Marshal(env.EnvVars)
		if err != nil {
			return fmt.Errorf("failed to marshal env_vars: %w", err)
		}
	}

	mcpServersJSON := []byte("[]")
	if env.MCPServers != nil && len(env.MCPServers) > 0 {
		var err error
		mcpServersJSON, err = json.Marshal(env.MCPServers)
		if err != nil {
			return fmt.Errorf("failed to marshal mcp_servers: %w", err)
		}
	}

	// Set defaults
	if env.VCPUs <= 0 {
		env.VCPUs = 2
	}
	if env.MemoryMB <= 0 {
		env.MemoryMB = 2048
	}
	if env.GitBranch == "" {
		env.GitBranch = "main"
	}
	if env.WorkingDirectory == "" {
		env.WorkingDirectory = "/workspace"
	}
	if env.IdleTimeoutSeconds <= 0 {
		env.IdleTimeoutSeconds = 1800 // 30 minutes
	}

	query := `
		INSERT INTO environments (
			id, name, description, vcpus, memory_mb,
			git_repo_url, git_branch, working_directory,
			tools, env_vars, mcp_servers, idle_timeout_seconds,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			NOW(), NOW()
		)
		RETURNING created_at, updated_at
	`

	err := r.db.QueryRowContext(ctx, query,
		env.ID,
		env.Name,
		toNullString(env.Description),
		env.VCPUs,
		env.MemoryMB,
		toNullString(&env.GitRepoURL),
		env.GitBranch,
		env.WorkingDirectory,
		toolsJSON,
		envVarsJSON,
		mcpServersJSON,
		env.IdleTimeoutSeconds,
	).Scan(&env.CreatedAt, &env.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create environment: %w", err)
	}

	return nil
}

// Get retrieves an environment by ID
func (r *environmentRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Environment, error) {
	query := `
		SELECT id, name, description, vcpus, memory_mb,
			   git_repo_url, git_branch, working_directory,
			   tools, env_vars, mcp_servers, idle_timeout_seconds,
			   created_at, updated_at
		FROM environments
		WHERE id = $1
	`

	var row environmentRow
	if err := r.db.GetContext(ctx, &row, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("environment not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return row.toEnvironment()
}

// GetByName retrieves an environment by name
func (r *environmentRepository) GetByName(ctx context.Context, name string) (*storage.Environment, error) {
	query := `
		SELECT id, name, description, vcpus, memory_mb,
			   git_repo_url, git_branch, working_directory,
			   tools, env_vars, mcp_servers, idle_timeout_seconds,
			   created_at, updated_at
		FROM environments
		WHERE name = $1
	`

	var row environmentRow
	if err := r.db.GetContext(ctx, &row, query, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("environment not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return row.toEnvironment()
}

// List retrieves all environments
func (r *environmentRepository) List(ctx context.Context) ([]*storage.Environment, error) {
	query := `
		SELECT id, name, description, vcpus, memory_mb,
			   git_repo_url, git_branch, working_directory,
			   tools, env_vars, mcp_servers, idle_timeout_seconds,
			   created_at, updated_at
		FROM environments
		ORDER BY created_at DESC
	`

	var rows []environmentRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	environments := make([]*storage.Environment, 0, len(rows))
	for _, row := range rows {
		env, err := row.toEnvironment()
		if err != nil {
			return nil, err
		}
		environments = append(environments, env)
	}

	return environments, nil
}

// Update updates an existing environment
func (r *environmentRepository) Update(ctx context.Context, env *storage.Environment) error {
	// Marshal JSON fields
	toolsJSON, err := json.Marshal(env.Tools)
	if err != nil {
		return fmt.Errorf("failed to marshal tools: %w", err)
	}

	envVarsJSON, err := json.Marshal(env.EnvVars)
	if err != nil {
		return fmt.Errorf("failed to marshal env_vars: %w", err)
	}

	mcpServersJSON, err := json.Marshal(env.MCPServers)
	if err != nil {
		return fmt.Errorf("failed to marshal mcp_servers: %w", err)
	}

	query := `
		UPDATE environments
		SET name = $2,
			description = $3,
			vcpus = $4,
			memory_mb = $5,
			git_repo_url = $6,
			git_branch = $7,
			working_directory = $8,
			tools = $9,
			env_vars = $10,
			mcp_servers = $11,
			idle_timeout_seconds = $12,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.db.QueryRowContext(ctx, query,
		env.ID,
		env.Name,
		toNullString(env.Description),
		env.VCPUs,
		env.MemoryMB,
		toNullString(&env.GitRepoURL),
		env.GitBranch,
		env.WorkingDirectory,
		toolsJSON,
		envVarsJSON,
		mcpServersJSON,
		env.IdleTimeoutSeconds,
	).Scan(&env.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("environment not found: %s", env.ID)
		}
		return fmt.Errorf("failed to update environment: %w", err)
	}

	return nil
}

// Delete deletes an environment by ID
func (r *environmentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM environments WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("environment not found: %s", id)
	}

	return nil
}
