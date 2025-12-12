package postgres

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/jmoiron/sqlx"
)

type workerRepository struct {
	db *sqlx.DB
}

func (r *workerRepository) Create(ctx context.Context, worker *storage.Worker) error {
	query := `
		INSERT INTO workers (
			id, hostname, address, status, last_seen, started_at,
			zone, labels, capabilities,
			cpu_cores, memory_mb, disk_gb,
			used_cpu_cores, used_memory_mb, used_disk_gb,
			vm_count, max_vms, metadata
		) VALUES (
			:id, :hostname, :address, :status, :last_seen, :started_at,
			:zone, :labels, :capabilities,
			:cpu_cores, :memory_mb, :disk_gb,
			:used_cpu_cores, :used_memory_mb, :used_disk_gb,
			:vm_count, :max_vms, :metadata
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, worker)
	if err != nil {
		return fmt.Errorf("failed to create worker: %w", err)
	}
	return nil
}

func (r *workerRepository) Get(ctx context.Context, id string) (*storage.Worker, error) {
	var worker storage.Worker
	query := `
		SELECT id, hostname, address, status, last_seen, started_at,
		       zone, labels, capabilities,
		       cpu_cores, memory_mb, disk_gb,
		       used_cpu_cores, used_memory_mb, used_disk_gb,
		       vm_count, max_vms, metadata, created_at, updated_at
		FROM workers
		WHERE id = $1
	`
	if err := r.db.GetContext(ctx, &worker, query, id); err != nil {
		return nil, fmt.Errorf("failed to get worker: %w", err)
	}
	return &worker, nil
}

func (r *workerRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.Worker, error) {
	query := `
		SELECT id, hostname, address, status, last_seen, started_at,
		       zone, labels, capabilities,
		       cpu_cores, memory_mb, disk_gb,
		       used_cpu_cores, used_memory_mb, used_disk_gb,
		       vm_count, max_vms, metadata, created_at, updated_at
		FROM workers
		ORDER BY created_at DESC
	`

	var workers []*storage.Worker
	if err := r.db.SelectContext(ctx, &workers, query); err != nil {
		return nil, fmt.Errorf("failed to list workers: %w", err)
	}
	return workers, nil
}

func (r *workerRepository) Update(ctx context.Context, worker *storage.Worker) error {
	query := `
		UPDATE workers SET
			hostname = :hostname,
			address = :address,
			status = :status,
			last_seen = :last_seen,
			zone = :zone,
			labels = :labels,
			capabilities = :capabilities,
			cpu_cores = :cpu_cores,
			memory_mb = :memory_mb,
			disk_gb = :disk_gb,
			used_cpu_cores = :used_cpu_cores,
			used_memory_mb = :used_memory_mb,
			used_disk_gb = :used_disk_gb,
			vm_count = :vm_count,
			max_vms = :max_vms,
			metadata = :metadata
		WHERE id = :id
	`
	result, err := r.db.NamedExecContext(ctx, query, worker)
	if err != nil {
		return fmt.Errorf("failed to update worker: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", worker.ID)
	}

	return nil
}

func (r *workerRepository) UpdateResources(ctx context.Context, id string, resources map[string]interface{}) error {
	query := `
		UPDATE workers SET
			used_cpu_cores = COALESCE($2, used_cpu_cores),
			used_memory_mb = COALESCE($3, used_memory_mb),
			used_disk_gb = COALESCE($4, used_disk_gb),
			vm_count = COALESCE($5, vm_count),
			last_seen = NOW()
		WHERE id = $1
	`

	usedCPU, _ := resources["used_cpu_cores"]
	usedMemory, _ := resources["used_memory_mb"]
	usedDisk, _ := resources["used_disk_gb"]
	vmCount, _ := resources["vm_count"]

	result, err := r.db.ExecContext(ctx, query, id, usedCPU, usedMemory, usedDisk, vmCount)
	if err != nil {
		return fmt.Errorf("failed to update worker resources: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", id)
	}

	return nil
}

func (r *workerRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE workers SET
			status = $2,
			last_seen = NOW()
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update worker status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", id)
	}

	return nil
}

func (r *workerRepository) UpdateLastSeen(ctx context.Context, id string) error {
	query := `UPDATE workers SET last_seen = NOW() WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update worker last_seen: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", id)
	}

	return nil
}

func (r *workerRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workers WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete worker: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("worker not found: %s", id)
	}

	return nil
}

func (r *workerRepository) ListByZone(ctx context.Context, zone string) ([]*storage.Worker, error) {
	query := `
		SELECT id, hostname, address, status, last_seen, started_at,
		       zone, labels, capabilities,
		       cpu_cores, memory_mb, disk_gb,
		       used_cpu_cores, used_memory_mb, used_disk_gb,
		       vm_count, max_vms, metadata, created_at, updated_at
		FROM workers
		WHERE zone = $1
		ORDER BY created_at DESC
	`

	var workers []*storage.Worker
	if err := r.db.SelectContext(ctx, &workers, query, zone); err != nil {
		return nil, fmt.Errorf("failed to list workers by zone: %w", err)
	}
	return workers, nil
}

func (r *workerRepository) ListActive(ctx context.Context) ([]*storage.Worker, error) {
	query := `
		SELECT id, hostname, address, status, last_seen, started_at,
		       zone, labels, capabilities,
		       cpu_cores, memory_mb, disk_gb,
		       used_cpu_cores, used_memory_mb, used_disk_gb,
		       vm_count, max_vms, metadata, created_at, updated_at
		FROM workers
		WHERE status = 'active'
		ORDER BY created_at DESC
	`

	var workers []*storage.Worker
	if err := r.db.SelectContext(ctx, &workers, query); err != nil {
		return nil, fmt.Errorf("failed to list active workers: %w", err)
	}
	return workers, nil
}
