package service

import (
	"context"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/google/uuid"
)

// TaskService handles task operations
type TaskService struct {
	queue queue.Queue
	store storage.Store
}

// NewTaskService creates a new task service
func NewTaskService(q queue.Queue, s storage.Store) *TaskService {
	return &TaskService{
		queue: q,
		store: s,
	}
}

// CreateVMTask submits a VM creation task
func (s *TaskService) CreateVMTask(ctx context.Context, name string, vcpus, memoryMB int) (uuid.UUID, error) {
	return s.CreateVMTaskWithTools(ctx, name, vcpus, memoryMB, nil, nil)
}

// CreateVMTaskWithTools submits a VM creation task with additional tools
func (s *TaskService) CreateVMTaskWithTools(ctx context.Context, name string, vcpus, memoryMB int, additionalTools []string, toolVersions map[string]string) (uuid.UUID, error) {
	payload := map[string]interface{}{
		"name":      name,
		"vcpus":     vcpus,
		"memory_mb": memoryMB,
	}

	if len(additionalTools) > 0 {
		payload["additional_tools"] = additionalTools
	}

	if len(toolVersions) > 0 {
		payload["tool_versions"] = toolVersions
	}

	task := &queue.Task{
		ID:      uuid.New(),
		Type:    queue.TaskTypeVMCreate,
		Payload: payload,
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 3,
		Timeout:  25 * time.Minute, // Increased for tool installation
		Queue:    "default",
		Priority: 5,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to enqueue VM creation task: %w", err)
	}

	return task.ID, nil
}

// ExecuteCommandTask submits a command execution task
func (s *TaskService) ExecuteCommandTask(ctx context.Context, vmID, command string, args []string) (uuid.UUID, error) {
	task := &queue.Task{
		ID:   uuid.New(),
		Type: queue.TaskTypeVMExecute,
		Payload: map[string]interface{}{
			"vm_id":   vmID,
			"command": command,
			"args":    args,
		},
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 2,
		Timeout:  10 * time.Minute,
		Queue:    "default",
		Priority: 5,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to enqueue execution task: %w", err)
	}

	return task.ID, nil
}

// DeleteVMTask submits a VM deletion task
func (s *TaskService) DeleteVMTask(ctx context.Context, vmID string) (uuid.UUID, error) {
	task := &queue.Task{
		ID:   uuid.New(),
		Type: queue.TaskTypeVMDelete,
		Payload: map[string]interface{}{
			"vm_id": vmID,
		},
	}

	if err := s.queue.Enqueue(ctx, task, &queue.TaskOptions{
		MaxRetry: 2,
		Timeout:  2 * time.Minute,
		Queue:    "default",
		Priority: 5,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("failed to enqueue deletion task: %w", err)
	}

	return task.ID, nil
}

// GetVM retrieves VM information from storage
func (s *TaskService) GetVM(ctx context.Context, vmID uuid.UUID) (*storage.VM, error) {
	return s.store.VMs().Get(ctx, vmID)
}

// GetVMByName retrieves VM by name
func (s *TaskService) GetVMByName(ctx context.Context, name string) (*storage.VM, error) {
	return s.store.VMs().GetByName(ctx, name)
}

// ListVMs lists all VMs
func (s *TaskService) ListVMs(ctx context.Context) ([]*storage.VM, error) {
	return s.store.VMs().List(ctx, map[string]interface{}{})
}

// GetExecutions retrieves execution history for a VM
func (s *TaskService) GetExecutions(ctx context.Context, vmID uuid.UUID) ([]*storage.Execution, error) {
	return s.store.Executions().ListByVM(ctx, vmID)
}
