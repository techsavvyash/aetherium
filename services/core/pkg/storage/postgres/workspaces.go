package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type workspaceRepository struct {
	db *sqlx.DB
}

func (r *workspaceRepository) Create(ctx context.Context, workspace *storage.Workspace) error {
	query := `
		INSERT INTO workspaces (
			id, name, description, vm_id, status, ai_assistant, ai_assistant_config,
			working_directory, environment_id, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)`

	// Initialize with empty JSON object as default for JSONB columns
	configJSON := []byte("{}")
	metadataJSON := []byte("{}")
	var err error

	if workspace.AIAssistantConfig != nil {
		configJSON, err = json.Marshal(workspace.AIAssistantConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal ai_assistant_config: %w", err)
		}
	}

	if workspace.Metadata != nil {
		metadataJSON, err = json.Marshal(workspace.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		workspace.ID, workspace.Name, workspace.Description, workspace.VMID,
		workspace.Status, workspace.AIAssistant, configJSON,
		workspace.WorkingDirectory, workspace.EnvironmentID, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	return nil
}

func (r *workspaceRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Workspace, error) {
	var workspace storage.Workspace
	query := `SELECT * FROM workspaces WHERE id = $1`

	err := r.db.GetContext(ctx, &workspace, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return &workspace, nil
}

func (r *workspaceRepository) GetByName(ctx context.Context, name string) (*storage.Workspace, error) {
	var workspace storage.Workspace
	query := `SELECT * FROM workspaces WHERE name = $1`

	err := r.db.GetContext(ctx, &workspace, query, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return &workspace, nil
}

func (r *workspaceRepository) GetByVMID(ctx context.Context, vmID uuid.UUID) (*storage.Workspace, error) {
	var workspace storage.Workspace
	query := `SELECT * FROM workspaces WHERE vm_id = $1`

	err := r.db.GetContext(ctx, &workspace, query, vmID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("workspace not found for VM: %s", vmID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace: %w", err)
	}

	return &workspace, nil
}

func (r *workspaceRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.Workspace, error) {
	query := `SELECT * FROM workspaces WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if aiAssistant, ok := filters["ai_assistant"].(string); ok {
		query += fmt.Sprintf(" AND ai_assistant = $%d", argIndex)
		args = append(args, aiAssistant)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	var workspaces []*storage.Workspace
	err := r.db.SelectContext(ctx, &workspaces, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces: %w", err)
	}

	return workspaces, nil
}

func (r *workspaceRepository) Update(ctx context.Context, workspace *storage.Workspace) error {
	query := `
		UPDATE workspaces SET
			name = $2, description = $3, vm_id = $4, status = $5,
			ai_assistant = $6, ai_assistant_config = $7, working_directory = $8,
			ready_at = $9, stopped_at = $10, metadata = $11
		WHERE id = $1`

	// Initialize with empty JSON object as default for JSONB columns
	configJSON := []byte("{}")
	metadataJSON := []byte("{}")
	var err error

	if workspace.AIAssistantConfig != nil {
		configJSON, err = json.Marshal(workspace.AIAssistantConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal ai_assistant_config: %w", err)
		}
	}

	if workspace.Metadata != nil {
		metadataJSON, err = json.Marshal(workspace.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		workspace.ID, workspace.Name, workspace.Description, workspace.VMID,
		workspace.Status, workspace.AIAssistant, configJSON, workspace.WorkingDirectory,
		workspace.ReadyAt, workspace.StoppedAt, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update workspace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", workspace.ID)
	}

	return nil
}

func (r *workspaceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE workspaces SET status = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update workspace status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) SetVMID(ctx context.Context, id uuid.UUID, vmID uuid.UUID) error {
	query := `UPDATE workspaces SET vm_id = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, vmID)
	if err != nil {
		return fmt.Errorf("failed to set workspace VM ID: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) SetReady(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE workspaces SET status = 'ready', ready_at = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set workspace ready: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workspaces WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) ListByEnvironment(ctx context.Context, environmentID uuid.UUID) ([]*storage.Workspace, error) {
	query := `SELECT * FROM workspaces WHERE environment_id = $1 ORDER BY created_at DESC`

	var workspaces []*storage.Workspace
	err := r.db.SelectContext(ctx, &workspaces, query, environmentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspaces by environment: %w", err)
	}

	return workspaces, nil
}

func (r *workspaceRepository) ListIdleWithVMs(ctx context.Context) ([]*storage.Workspace, error) {
	query := `
		SELECT * FROM workspaces
		WHERE status = 'ready'
		AND vm_id IS NOT NULL
		AND idle_since IS NOT NULL
		ORDER BY idle_since ASC`

	var workspaces []*storage.Workspace
	err := r.db.SelectContext(ctx, &workspaces, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list idle workspaces: %w", err)
	}

	return workspaces, nil
}

func (r *workspaceRepository) UpdateIdleSince(ctx context.Context, id uuid.UUID, idleSince *time.Time) error {
	query := `UPDATE workspaces SET idle_since = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, idleSince)
	if err != nil {
		return fmt.Errorf("failed to update workspace idle_since: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) ClearVMID(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE workspaces SET vm_id = NULL WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to clear workspace VM ID: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

func (r *workspaceRepository) SetEnvironmentID(ctx context.Context, id uuid.UUID, environmentID uuid.UUID) error {
	query := `UPDATE workspaces SET environment_id = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, environmentID)
	if err != nil {
		return fmt.Errorf("failed to set workspace environment ID: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("workspace not found: %s", id)
	}

	return nil
}

// secretRepository implements storage.SecretRepository
type secretRepository struct {
	db *sqlx.DB
}

func (r *secretRepository) Create(ctx context.Context, secret *storage.WorkspaceSecret) error {
	query := `
		INSERT INTO workspace_secrets (
			id, workspace_id, name, description, secret_type,
			encrypted_value, encryption_key_id, nonce, scope
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(ctx, query,
		secret.ID, secret.WorkspaceID, secret.Name, secret.Description,
		secret.SecretType, secret.EncryptedValue, secret.EncryptionKeyID,
		secret.Nonce, secret.Scope,
	)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

func (r *secretRepository) Get(ctx context.Context, id uuid.UUID) (*storage.WorkspaceSecret, error) {
	var secret storage.WorkspaceSecret
	query := `SELECT * FROM workspace_secrets WHERE id = $1`

	err := r.db.GetContext(ctx, &secret, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("secret not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return &secret, nil
}

func (r *secretRepository) GetByName(ctx context.Context, workspaceID *uuid.UUID, name string) (*storage.WorkspaceSecret, error) {
	var secret storage.WorkspaceSecret
	var query string
	var args []interface{}

	if workspaceID == nil {
		query = `SELECT * FROM workspace_secrets WHERE workspace_id IS NULL AND name = $1`
		args = []interface{}{name}
	} else {
		query = `SELECT * FROM workspace_secrets WHERE workspace_id = $1 AND name = $2`
		args = []interface{}{workspaceID, name}
	}

	err := r.db.GetContext(ctx, &secret, query, args...)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("secret not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return &secret, nil
}

func (r *secretRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*storage.WorkspaceSecret, error) {
	query := `SELECT * FROM workspace_secrets WHERE workspace_id = $1 ORDER BY name`

	var secrets []*storage.WorkspaceSecret
	err := r.db.SelectContext(ctx, &secrets, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets, nil
}

func (r *secretRepository) ListGlobal(ctx context.Context) ([]*storage.WorkspaceSecret, error) {
	query := `SELECT * FROM workspace_secrets WHERE scope = 'global' ORDER BY name`

	var secrets []*storage.WorkspaceSecret
	err := r.db.SelectContext(ctx, &secrets, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list global secrets: %w", err)
	}

	return secrets, nil
}

func (r *secretRepository) Update(ctx context.Context, secret *storage.WorkspaceSecret) error {
	query := `
		UPDATE workspace_secrets SET
			name = $2, description = $3, secret_type = $4,
			encrypted_value = $5, encryption_key_id = $6, nonce = $7,
			scope = $8, updated_at = $9
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		secret.ID, secret.Name, secret.Description, secret.SecretType,
		secret.EncryptedValue, secret.EncryptionKeyID, secret.Nonce,
		secret.Scope, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("secret not found: %s", secret.ID)
	}

	return nil
}

func (r *secretRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workspace_secrets WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("secret not found: %s", id)
	}

	return nil
}

// prepStepRepository implements storage.PrepStepRepository
type prepStepRepository struct {
	db *sqlx.DB
}

func (r *prepStepRepository) Create(ctx context.Context, step *storage.PrepStep) error {
	query := `
		INSERT INTO workspace_prep_steps (
			id, workspace_id, step_type, step_order, config, status, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	// Explicitly marshal JSONB fields to []byte for PostgreSQL
	configJSON := []byte("{}")
	metadataJSON := []byte("{}")
	var err error

	if step.Config != nil {
		configJSON, err = json.Marshal(step.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	if step.Metadata != nil {
		metadataJSON, err = json.Marshal(step.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		step.ID, step.WorkspaceID, step.StepType, step.StepOrder,
		configJSON, step.Status, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create prep step: %w", err)
	}

	return nil
}

func (r *prepStepRepository) CreateBatch(ctx context.Context, steps []*storage.PrepStep) error {
	for _, step := range steps {
		if err := r.Create(ctx, step); err != nil {
			return err
		}
	}
	return nil
}

func (r *prepStepRepository) Get(ctx context.Context, id uuid.UUID) (*storage.PrepStep, error) {
	var step storage.PrepStep
	query := `SELECT * FROM workspace_prep_steps WHERE id = $1`

	err := r.db.GetContext(ctx, &step, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prep step not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prep step: %w", err)
	}

	return &step, nil
}

func (r *prepStepRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*storage.PrepStep, error) {
	query := `SELECT * FROM workspace_prep_steps WHERE workspace_id = $1 ORDER BY step_order`

	var steps []*storage.PrepStep
	err := r.db.SelectContext(ctx, &steps, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list prep steps: %w", err)
	}

	return steps, nil
}

func (r *prepStepRepository) GetNextPending(ctx context.Context, workspaceID uuid.UUID) (*storage.PrepStep, error) {
	var step storage.PrepStep
	query := `SELECT * FROM workspace_prep_steps WHERE workspace_id = $1 AND status = 'pending' ORDER BY step_order LIMIT 1`

	err := r.db.GetContext(ctx, &step, query, workspaceID)
	if err == sql.ErrNoRows {
		return nil, nil // No pending steps
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending prep step: %w", err)
	}

	return &step, nil
}

func (r *prepStepRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, result *storage.PrepStepResult) error {
	var query string
	var args []interface{}

	if result != nil {
		query = `
			UPDATE workspace_prep_steps SET
				status = $2, exit_code = $3, stdout = $4, stderr = $5,
				error = $6, completed_at = $7, duration_ms = $8
			WHERE id = $1`
		args = []interface{}{
			id, status, result.ExitCode, result.Stdout, result.Stderr,
			result.Error, time.Now(), result.DurationMS,
		}
	} else {
		query = `UPDATE workspace_prep_steps SET status = $2, started_at = $3 WHERE id = $1`
		args = []interface{}{id, status, time.Now()}
	}

	result2, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update prep step status: %w", err)
	}

	rows, err := result2.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("prep step not found: %s", id)
	}

	return nil
}

// promptTaskRepository implements storage.PromptTaskRepository
type promptTaskRepository struct {
	db *sqlx.DB
}

func (r *promptTaskRepository) Create(ctx context.Context, task *storage.PromptTask) error {
	query := `
		INSERT INTO prompt_tasks (
			id, workspace_id, prompt, system_prompt, working_directory,
			environment, priority, status, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	// Initialize with empty JSON object as default for JSONB columns
	envJSON := []byte("{}")
	metadataJSON := []byte("{}")
	var err error

	if task.Environment != nil {
		envJSON, err = json.Marshal(task.Environment)
		if err != nil {
			return fmt.Errorf("failed to marshal environment: %w", err)
		}
	}

	if task.Metadata != nil {
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		task.ID, task.WorkspaceID, task.Prompt, task.SystemPrompt,
		task.WorkingDirectory, envJSON, task.Priority, task.Status, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create prompt task: %w", err)
	}

	return nil
}

func (r *promptTaskRepository) Get(ctx context.Context, id uuid.UUID) (*storage.PromptTask, error) {
	var task storage.PromptTask
	query := `SELECT * FROM prompt_tasks WHERE id = $1`

	err := r.db.GetContext(ctx, &task, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("prompt task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt task: %w", err)
	}

	return &task, nil
}

func (r *promptTaskRepository) ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int) ([]*storage.PromptTask, error) {
	query := `SELECT * FROM prompt_tasks WHERE workspace_id = $1 ORDER BY created_at DESC`
	args := []interface{}{workspaceID}

	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	var tasks []*storage.PromptTask
	err := r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list prompt tasks: %w", err)
	}

	return tasks, nil
}

func (r *promptTaskRepository) GetNextPending(ctx context.Context, workspaceID uuid.UUID) (*storage.PromptTask, error) {
	var task storage.PromptTask
	query := `
		SELECT * FROM prompt_tasks
		WHERE workspace_id = $1 AND status = 'pending'
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1`

	err := r.db.GetContext(ctx, &task, query, workspaceID)
	if err == sql.ErrNoRows {
		return nil, nil // No pending tasks
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending prompt task: %w", err)
	}

	return &task, nil
}

func (r *promptTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, result *storage.PromptResult) error {
	var query string
	var args []interface{}

	if result != nil {
		query = `
			UPDATE prompt_tasks SET
				status = $2, exit_code = $3, stdout = $4, stderr = $5,
				error = $6, completed_at = $7, duration_ms = $8
			WHERE id = $1`
		args = []interface{}{
			id, status, result.ExitCode, result.Stdout, result.Stderr,
			result.Error, time.Now(), result.DurationMS,
		}
	} else {
		query = `UPDATE prompt_tasks SET status = $2, started_at = $3 WHERE id = $1`
		args = []interface{}{id, status, time.Now()}
	}

	result2, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update prompt task status: %w", err)
	}

	rows, err := result2.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("prompt task not found: %s", id)
	}

	return nil
}

func (r *promptTaskRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE prompt_tasks SET status = 'cancelled' WHERE id = $1 AND status = 'pending'`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to cancel prompt task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("prompt task not found or not pending: %s", id)
	}

	return nil
}

// sessionRepository implements storage.SessionRepository
type sessionRepository struct {
	db *sqlx.DB
}

func (r *sessionRepository) Create(ctx context.Context, session *storage.WorkspaceSession) error {
	query := `
		INSERT INTO workspace_sessions (
			id, workspace_id, status, client_ip, user_agent, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	var metadataJSON []byte
	var err error

	if session.Metadata != nil {
		metadataJSON, err = json.Marshal(session.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		session.ID, session.WorkspaceID, session.Status,
		session.ClientIP, session.UserAgent, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *sessionRepository) Get(ctx context.Context, id uuid.UUID) (*storage.WorkspaceSession, error) {
	var session storage.WorkspaceSession
	query := `SELECT * FROM workspace_sessions WHERE id = $1`

	err := r.db.GetContext(ctx, &session, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

func (r *sessionRepository) GetActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*storage.WorkspaceSession, error) {
	query := `SELECT * FROM workspace_sessions WHERE workspace_id = $1 AND status = 'active' ORDER BY connected_at DESC`

	var sessions []*storage.WorkspaceSession
	err := r.db.SelectContext(ctx, &sessions, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sessions: %w", err)
	}

	return sessions, nil
}

func (r *sessionRepository) UpdateLastActivity(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE workspace_sessions SET last_activity = $2 WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update session last activity: %w", err)
	}

	return nil
}

func (r *sessionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	var query string
	if status == "disconnected" || status == "terminated" {
		query = `UPDATE workspace_sessions SET status = $2, disconnected_at = $3 WHERE id = $1`
		_, err := r.db.ExecContext(ctx, query, id, status, time.Now())
		if err != nil {
			return fmt.Errorf("failed to update session status: %w", err)
		}
	} else {
		query = `UPDATE workspace_sessions SET status = $2 WHERE id = $1`
		_, err := r.db.ExecContext(ctx, query, id, status)
		if err != nil {
			return fmt.Errorf("failed to update session status: %w", err)
		}
	}

	return nil
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM workspace_sessions WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found: %s", id)
	}

	return nil
}

// sessionMessageRepository implements storage.SessionMessageRepository
type sessionMessageRepository struct {
	db *sqlx.DB
}

func (r *sessionMessageRepository) Create(ctx context.Context, message *storage.SessionMessage) error {
	query := `
		INSERT INTO session_messages (
			id, session_id, message_type, content, exit_code, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)`

	var metadataJSON []byte
	var err error

	if message.Metadata != nil {
		metadataJSON, err = json.Marshal(message.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		message.ID, message.SessionID, message.MessageType,
		message.Content, message.ExitCode, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create session message: %w", err)
	}

	return nil
}

func (r *sessionMessageRepository) ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]*storage.SessionMessage, error) {
	query := `SELECT * FROM session_messages WHERE session_id = $1 ORDER BY created_at`
	args := []interface{}{sessionID}

	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	var messages []*storage.SessionMessage
	err := r.db.SelectContext(ctx, &messages, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list session messages: %w", err)
	}

	return messages, nil
}
