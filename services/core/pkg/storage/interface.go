package storage

import (
	"context"

	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// StateStore defines the interface for state persistence implementations
type StateStore interface {
	// Project operations
	CreateProject(ctx context.Context, project *types.Project) error
	GetProject(ctx context.Context, id string) (*types.Project, error)
	UpdateProject(ctx context.Context, project *types.Project) error
	DeleteProject(ctx context.Context, id string) error
	ListProjects(ctx context.Context, filters *ProjectFilters) ([]*types.Project, error)

	// Task operations
	CreateTask(ctx context.Context, task *types.Task) error
	GetTask(ctx context.Context, id string) (*types.Task, error)
	UpdateTask(ctx context.Context, task *types.Task) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context, filters *TaskFilters) ([]*types.Task, error)

	// VM operations
	CreateVM(ctx context.Context, vm *types.VM) error
	GetVM(ctx context.Context, id string) (*types.VM, error)
	UpdateVM(ctx context.Context, vm *types.VM) error
	DeleteVM(ctx context.Context, id string) error
	ListVMs(ctx context.Context, filters *VMFilters) ([]*types.VM, error)

	// Health check
	Health(ctx context.Context) error

	// Close connection
	Close() error
}

// ProjectFilters represents filters for listing projects
type ProjectFilters struct {
	Limit  int
	Offset int
}

// TaskFilters represents filters for listing tasks
type TaskFilters struct {
	ProjectID    *string
	ParentTaskID *string
	Status       *types.TaskStatus
	AgentType    *string
	Limit        int
	Offset       int
}

// VMFilters represents filters for listing VMs
type VMFilters struct {
	Status *types.VMStatus
	Limit  int
	Offset int
}

// Config represents storage configuration
type Config struct {
	// Provider-specific configuration
	Options map[string]interface{}
}
