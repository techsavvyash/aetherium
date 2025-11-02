package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type workerMetricRepository struct {
	db *sqlx.DB
}

func (r *workerMetricRepository) Create(ctx context.Context, metric *storage.WorkerMetric) error {
	if metric.ID == uuid.Nil {
		metric.ID = uuid.New()
	}

	query := `
		INSERT INTO worker_metrics (
			id, worker_id, timestamp,
			cpu_usage, memory_usage, disk_usage,
			vm_count, tasks_processed,
			network_in_mb, network_out_mb,
			metadata
		) VALUES (
			:id, :worker_id, :timestamp,
			:cpu_usage, :memory_usage, :disk_usage,
			:vm_count, :tasks_processed,
			:network_in_mb, :network_out_mb,
			:metadata
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, metric)
	if err != nil {
		return fmt.Errorf("failed to create worker metric: %w", err)
	}
	return nil
}

func (r *workerMetricRepository) Get(ctx context.Context, id uuid.UUID) (*storage.WorkerMetric, error) {
	var metric storage.WorkerMetric
	query := `
		SELECT id, worker_id, timestamp,
		       cpu_usage, memory_usage, disk_usage,
		       vm_count, tasks_processed,
		       network_in_mb, network_out_mb,
		       metadata
		FROM worker_metrics
		WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &metric, query, id); err != nil {
		return nil, fmt.Errorf("failed to get worker metric: %w", err)
	}
	return &metric, nil
}

func (r *workerMetricRepository) ListByWorker(ctx context.Context, workerID string, limit int) ([]*storage.WorkerMetric, error) {
	if limit <= 0 {
		limit = 100 // Default limit
	}

	query := `
		SELECT id, worker_id, timestamp,
		       cpu_usage, memory_usage, disk_usage,
		       vm_count, tasks_processed,
		       network_in_mb, network_out_mb,
		       metadata
		FROM worker_metrics
		WHERE worker_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	var metrics []*storage.WorkerMetric
	if err := r.db.SelectContext(ctx, &metrics, query, workerID, limit); err != nil {
		return nil, fmt.Errorf("failed to list worker metrics: %w", err)
	}
	return metrics, nil
}

func (r *workerMetricRepository) ListByWorkerInTimeRange(ctx context.Context, workerID string, start, end time.Time) ([]*storage.WorkerMetric, error) {
	query := `
		SELECT id, worker_id, timestamp,
		       cpu_usage, memory_usage, disk_usage,
		       vm_count, tasks_processed,
		       network_in_mb, network_out_mb,
		       metadata
		FROM worker_metrics
		WHERE worker_id = $1 AND timestamp >= $2 AND timestamp <= $3
		ORDER BY timestamp ASC
	`

	var metrics []*storage.WorkerMetric
	if err := r.db.SelectContext(ctx, &metrics, query, workerID, start, end); err != nil {
		return nil, fmt.Errorf("failed to list worker metrics in time range: %w", err)
	}
	return metrics, nil
}

func (r *workerMetricRepository) DeleteOlderThan(ctx context.Context, before time.Time) error {
	query := `DELETE FROM worker_metrics WHERE timestamp < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return fmt.Errorf("failed to delete old worker metrics: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	// Log number of rows deleted (could use proper logging here)
	_ = rows

	return nil
}
