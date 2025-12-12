package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type vmRepository struct {
	db *sqlx.DB
}

func (r *vmRepository) Create(ctx context.Context, vm *storage.VM) error {
	query := `
		INSERT INTO vms (
			id, name, orchestrator, status, kernel_path, rootfs_path, socket_path,
			vcpu_count, memory_mb, worker_id, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	// Marshal metadata to JSON
	var metadataJSON []byte
	var err error
	if vm.Metadata != nil {
		metadataJSON, err = json.Marshal(vm.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		vm.ID, vm.Name, vm.Orchestrator, vm.Status,
		vm.KernelPath, vm.RootFSPath, vm.SocketPath,
		vm.VCPUCount, vm.MemoryMB, vm.WorkerID, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	return nil
}

func (r *vmRepository) Get(ctx context.Context, id uuid.UUID) (*storage.VM, error) {
	var vm storage.VM
	query := `SELECT * FROM vms WHERE id = $1`

	err := r.db.GetContext(ctx, &vm, query, id)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("VM not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	return &vm, nil
}

func (r *vmRepository) GetByName(ctx context.Context, name string) (*storage.VM, error) {
	var vm storage.VM
	query := `SELECT * FROM vms WHERE name = $1`

	err := r.db.GetContext(ctx, &vm, query, name)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("VM not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	return &vm, nil
}

func (r *vmRepository) List(ctx context.Context, filters map[string]interface{}) ([]*storage.VM, error) {
	query := `SELECT * FROM vms WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if orchestrator, ok := filters["orchestrator"].(string); ok {
		query += fmt.Sprintf(" AND orchestrator = $%d", argIndex)
		args = append(args, orchestrator)
		argIndex++
	}

	if status, ok := filters["status"].(string); ok {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if workerID, ok := filters["worker_id"].(string); ok {
		query += fmt.Sprintf(" AND worker_id = $%d", argIndex)
		args = append(args, workerID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"].(int); ok && limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	var vms []*storage.VM
	err := r.db.SelectContext(ctx, &vms, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	return vms, nil
}

func (r *vmRepository) Update(ctx context.Context, vm *storage.VM) error {
	query := `
		UPDATE vms SET
			name = $2, orchestrator = $3, status = $4,
			kernel_path = $5, rootfs_path = $6, socket_path = $7,
			vcpu_count = $8, memory_mb = $9,
			started_at = $10, stopped_at = $11, metadata = $12
		WHERE id = $1`

	// Marshal metadata to JSON
	var metadataJSON []byte
	var err error
	if vm.Metadata != nil {
		metadataJSON, err = json.Marshal(vm.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		vm.ID, vm.Name, vm.Orchestrator, vm.Status,
		vm.KernelPath, vm.RootFSPath, vm.SocketPath,
		vm.VCPUCount, vm.MemoryMB,
		vm.StartedAt, vm.StoppedAt, metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to update VM: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("VM not found: %s", vm.ID)
	}

	return nil
}

func (r *vmRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM vms WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("VM not found: %s", id)
	}

	return nil
}
