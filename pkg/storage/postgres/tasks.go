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

type taskRepository struct {
	db *sqlx.DB
}

func (r *taskRepository) Create(ctx context.Context, task *storage.Task) error {
	query := `
		INSERT INTO tasks (
			id, type, status, priority, payload, vm_id, worker_id,
			max_retries, retry_count, scheduled_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.Type, task.Status, task.Priority, task.Payload,
		task.VMID, task.WorkerID, task.MaxRetries, task.RetryCount,
		task.ScheduledAt, task.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (r *taskRepository) Get(ctx context.Context, id uuid.UUID) (*storage.Task, error) {
	var task storage.Task
	query := `SELECT * FROM tasks WHERE id = $1`

	err := r.db.GetContext(ctx, &task, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &task, nil
}

func (r *taskRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.Task, error) {
	query := `SELECT * FROM tasks WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if taskType, ok := filters["type"].(string); ok {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, taskType)
		argIndex++
	}

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

	query += " ORDER BY priority DESC, scheduled_at ASC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	var tasks []*storage.Task
	err := r.db.SelectContext(ctx, &tasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, nil
}

func (r *taskRepository) Update(ctx context.Context, task *storage.Task) error {
	query := `
		UPDATE tasks SET
			type = $2, status = $3, priority = $4, payload = $5,
			result = $6, error = $7, vm_id = $8, worker_id = $9,
			max_retries = $10, retry_count = $11, scheduled_at = $12,
			started_at = $13, completed_at = $14, metadata = $15
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		task.ID, task.Type, task.Status, task.Priority, task.Payload,
		task.Result, task.Error, task.VMID, task.WorkerID,
		task.MaxRetries, task.RetryCount, task.ScheduledAt,
		task.StartedAt, task.CompletedAt, task.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", task.ID)
	}

	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

func (r *taskRepository) GetNextPending(ctx context.Context) (*storage.Task, error) {
	var task storage.Task
	query := `
		SELECT * FROM tasks
		WHERE status = 'pending' AND scheduled_at <= NOW()
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	err := r.db.GetContext(ctx, &task, query)
	if err == sql.ErrNoRows {
		return nil, nil // No pending tasks
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get next pending task: %w", err)
	}

	return &task, nil
}

func (r *taskRepository) MarkProcessing(ctx context.Context, id uuid.UUID, workerID string) error {
	query := `
		UPDATE tasks SET
			status = 'processing',
			worker_id = $2,
			started_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, workerID)
	if err != nil {
		return fmt.Errorf("failed to mark task as processing: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

func (r *taskRepository) MarkCompleted(ctx context.Context, id uuid.UUID, result map[string]interface{}) error {
	query := `
		UPDATE tasks SET
			status = 'completed',
			result = $2,
			completed_at = NOW()
		WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id, result)
	if err != nil {
		return fmt.Errorf("failed to mark task as completed: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("task not found: %s", id)
	}

	return nil
}

func (r *taskRepository) MarkFailed(ctx context.Context, id uuid.UUID, taskErr error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get current retry count
	var task storage.Task
	query := `SELECT * FROM tasks WHERE id = $1 FOR UPDATE`
	err = tx.GetContext(ctx, &task, query, id)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	task.RetryCount++
	errorMsg := taskErr.Error()
	task.Error = &errorMsg

	// Determine new status
	newStatus := "failed"
	var completedAt *time.Time
	now := time.Now()

	if task.RetryCount < task.MaxRetries {
		newStatus = "retrying"
		// Schedule retry with exponential backoff
		delay := time.Duration(task.RetryCount*task.RetryCount) * time.Second
		scheduledAt := now.Add(delay)
		task.ScheduledAt = scheduledAt
	} else {
		completedAt = &now
	}

	// Update task
	updateQuery := `
		UPDATE tasks SET
			status = $2,
			error = $3,
			retry_count = $4,
			scheduled_at = $5,
			completed_at = $6
		WHERE id = $1`

	_, err = tx.ExecContext(ctx, updateQuery,
		id, newStatus, task.Error, task.RetryCount, task.ScheduledAt, completedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
