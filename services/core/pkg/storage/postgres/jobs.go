package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type jobRepository struct {
	db *sqlx.DB
}

func (r *jobRepository) Create(ctx context.Context, job *storage.Job) error {
	query := `
		INSERT INTO jobs (id, name, status, vm_id, commands, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Status, job.VMID, job.Commands, job.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

func (r *jobRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Job, error) {
	var job storage.Job
	query := `SELECT * FROM jobs WHERE id = $1`

	err := r.db.GetContext(ctx, &job, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

func (r *jobRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.Job, error) {
	query := `SELECT * FROM jobs WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if vmID, ok := filters["vm_id"].(uuid.UUID); ok {
		query += fmt.Sprintf(" AND vm_id = $%d", argIndex)
		args = append(args, vmID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	var jobs []*storage.Job
	err := r.db.SelectContext(ctx, &jobs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}

	return jobs, nil
}

func (r *jobRepository) Update(ctx context.Context, job *storage.Job) error {
	query := `
		UPDATE jobs SET
			name = $2, status = $3, vm_id = $4, commands = $5,
			results = $6, error = $7, started_at = $8, completed_at = $9, metadata = $10
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		job.ID, job.Name, job.Status, job.VMID, job.Commands,
		job.Results, job.Error, job.StartedAt, job.CompletedAt, job.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", job.ID)
	}

	return nil
}

func (r *jobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM jobs WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("job not found: %s", id)
	}

	return nil
}
