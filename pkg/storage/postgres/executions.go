package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type executionRepository struct {
	db *sqlx.DB
}

func (r *executionRepository) Create(ctx context.Context, execution *storage.Execution) error {
	query := `
		INSERT INTO executions (
			id, job_id, vm_id, command, args, env,
			exit_code, stdout, stderr, error,
			started_at, completed_at, duration_ms, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)`

	_, err := r.db.ExecContext(ctx, query,
		execution.ID, execution.JobID, execution.VMID, execution.Command,
		execution.Args, execution.Env, execution.ExitCode, execution.Stdout,
		execution.Stderr, execution.Error, execution.StartedAt, execution.CompletedAt,
		execution.DurationMS, execution.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (r *executionRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Execution, error) {
	var execution storage.Execution
	query := `SELECT * FROM executions WHERE id = $1`

	err := r.db.GetContext(ctx, &execution, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

func (r *executionRepository) ListByJob(ctx context.Context, jobID uuid.UUID) ([]*storage.Execution, error) {
	var executions []*storage.Execution
	query := `SELECT * FROM executions WHERE job_id = $1 ORDER BY started_at ASC`

	err := r.db.SelectContext(ctx, &executions, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions by job: %w", err)
	}

	return executions, nil
}

func (r *executionRepository) ListByVM(ctx context.Context, vmID uuid.UUID) ([]*storage.Execution, error) {
	var executions []*storage.Execution
	query := `SELECT * FROM executions WHERE vm_id = $1 ORDER BY started_at DESC`

	err := r.db.SelectContext(ctx, &executions, query, vmID)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions by VM: %w", err)
	}

	return executions, nil
}
