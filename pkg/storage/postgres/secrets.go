package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type secretsRepository struct {
	db *sqlx.DB
}

// Create creates a new secret
func (r *secretsRepository) Create(ctx context.Context, secret *storage.Secret) error {
	query := `
		INSERT INTO secrets (
			id, name, description, encrypted_value, encryption_key_id, encryption_version,
			scope, scope_id, created_by, tags, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		secret.ID,
		secret.Name,
		secret.Description,
		secret.EncryptedValue,
		secret.EncryptionKeyID,
		secret.EncryptionVersion,
		secret.Scope,
		secret.ScopeID,
		secret.CreatedBy,
		secret.Tags,
		secret.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	return nil
}

// Get retrieves a secret by ID
func (r *secretsRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Secret, error) {
	var secret storage.Secret

	query := `
		SELECT id, name, description, encrypted_value, encryption_key_id, encryption_version,
		       scope, scope_id, created_at, updated_at, expires_at, rotated_at,
		       created_by, updated_by, last_accessed_at, access_count, tags, metadata
		FROM secrets
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &secret, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("secret not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return &secret, nil
}

// GetByName retrieves a secret by name
func (r *secretsRepository) GetByName(ctx context.Context, name string) (*storage.Secret, error) {
	var secret storage.Secret

	query := `
		SELECT id, name, description, encrypted_value, encryption_key_id, encryption_version,
		       scope, scope_id, created_at, updated_at, expires_at, rotated_at,
		       created_by, updated_by, last_accessed_at, access_count, tags, metadata
		FROM secrets
		WHERE name = $1
	`

	err := r.db.GetContext(ctx, &secret, query, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("secret not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return &secret, nil
}

// List retrieves secrets with optional filters
func (r *secretsRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.Secret, error) {
	query := `
		SELECT id, name, description, encrypted_value, encryption_key_id, encryption_version,
		       scope, scope_id, created_at, updated_at, expires_at, rotated_at,
		       created_by, updated_by, last_accessed_at, access_count, tags, metadata
		FROM secrets
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Add filters
	if scope, ok := filters["scope"].(string); ok {
		query += fmt.Sprintf(" AND scope = $%d", argPos)
		args = append(args, scope)
		argPos++
	}

	if scopeID, ok := filters["scope_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND scope_id = $%d", argPos)
		args = append(args, scopeID)
		argPos++
	}

	if expired, ok := filters["expired"].(bool); ok {
		if expired {
			query += " AND expires_at IS NOT NULL AND expires_at < NOW()"
		} else {
			query += " AND (expires_at IS NULL OR expires_at >= NOW())"
		}
	}

	query += " ORDER BY created_at DESC"

	// Add limit
	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, limit)
	}

	var secrets []*storage.Secret
	err := r.db.SelectContext(ctx, &secrets, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	return secrets, nil
}

// Update updates an existing secret
func (r *secretsRepository) Update(ctx context.Context, secret *storage.Secret) error {
	query := `
		UPDATE secrets
		SET description = $2,
		    encrypted_value = $3,
		    encryption_version = $4,
		    expires_at = $5,
		    updated_by = $6,
		    tags = $7,
		    metadata = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		secret.ID,
		secret.Description,
		secret.EncryptedValue,
		secret.EncryptionVersion,
		secret.ExpiresAt,
		secret.UpdatedBy,
		secret.Tags,
		secret.Metadata,
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

// Delete deletes a secret
func (r *secretsRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM secrets WHERE id = $1`

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

// RecordAccess increments the access count and updates last accessed time
func (r *secretsRepository) RecordAccess(ctx context.Context, secretID uuid.UUID) error {
	query := `
		UPDATE secrets
		SET access_count = access_count + 1,
		    last_accessed_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, secretID)
	if err != nil {
		return fmt.Errorf("failed to record access: %w", err)
	}

	return nil
}

// CreateAuditLog creates an audit log entry
func (r *secretsRepository) CreateAuditLog(ctx context.Context, log *storage.SecretAuditLog) error {
	query := `
		INSERT INTO secret_audit_logs (
			id, secret_id, secret_name, action, actor, actor_ip,
			vm_id, execution_id, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.ID,
		log.SecretID,
		log.SecretName,
		log.Action,
		log.Actor,
		log.ActorIP,
		log.VMID,
		log.ExecutionID,
		log.Metadata,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs for a secret
func (r *secretsRepository) GetAuditLogs(ctx context.Context, secretID uuid.UUID, limit int) ([]*storage.SecretAuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, secret_id, secret_name, action, actor, actor_ip,
		       vm_id, execution_id, timestamp, metadata
		FROM secret_audit_logs
		WHERE secret_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	var logs []*storage.SecretAuditLog
	err := r.db.SelectContext(ctx, &logs, query, secretID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs: %w", err)
	}

	return logs, nil
}
