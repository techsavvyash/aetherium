package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/types"
)

// MemoryStore is an in-memory implementation of StateStore for testing
type MemoryStore struct {
	projects map[string]*types.Project
	tasks    map[string]*types.Task
	vms      map[string]*types.VM
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory state store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects: make(map[string]*types.Project),
		tasks:    make(map[string]*types.Task),
		vms:      make(map[string]*types.VM),
	}
}

// Project operations

func (m *MemoryStore) CreateProject(ctx context.Context, project *types.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.projects[project.ID]; exists {
		return fmt.Errorf("project with ID '%s' already exists", project.ID)
	}

	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	m.projects[project.ID] = project
	return nil
}

func (m *MemoryStore) GetProject(ctx context.Context, id string) (*types.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	project, exists := m.projects[id]
	if !exists {
		return nil, fmt.Errorf("project with ID '%s' not found", id)
	}

	return project, nil
}

func (m *MemoryStore) UpdateProject(ctx context.Context, project *types.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.projects[project.ID]; !exists {
		return fmt.Errorf("project with ID '%s' not found", project.ID)
	}

	project.UpdatedAt = time.Now()
	m.projects[project.ID] = project
	return nil
}

func (m *MemoryStore) DeleteProject(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.projects[id]; !exists {
		return fmt.Errorf("project with ID '%s' not found", id)
	}

	delete(m.projects, id)
	return nil
}

func (m *MemoryStore) ListProjects(ctx context.Context, filters *storage.ProjectFilters) ([]*types.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	projects := make([]*types.Project, 0, len(m.projects))
	for _, project := range m.projects {
		projects = append(projects, project)
	}

	// Apply limit/offset if specified
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(projects) {
			projects = projects[filters.Offset:]
		}
		if filters.Limit > 0 && filters.Limit < len(projects) {
			projects = projects[:filters.Limit]
		}
	}

	return projects, nil
}

// Task operations

func (m *MemoryStore) CreateTask(ctx context.Context, task *types.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID '%s' already exists", task.ID)
	}

	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()
	m.tasks[task.ID] = task
	return nil
}

func (m *MemoryStore) GetTask(ctx context.Context, id string) (*types.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task with ID '%s' not found", id)
	}

	return task, nil
}

func (m *MemoryStore) UpdateTask(ctx context.Context, task *types.Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[task.ID]; !exists {
		return fmt.Errorf("task with ID '%s' not found", task.ID)
	}

	task.UpdatedAt = time.Now()
	m.tasks[task.ID] = task
	return nil
}

func (m *MemoryStore) DeleteTask(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.tasks[id]; !exists {
		return fmt.Errorf("task with ID '%s' not found", id)
	}

	delete(m.tasks, id)
	return nil
}

func (m *MemoryStore) ListTasks(ctx context.Context, filters *storage.TaskFilters) ([]*types.Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*types.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		if filters != nil {
			if filters.ProjectID != nil && task.ProjectID != *filters.ProjectID {
				continue
			}
			if filters.ParentTaskID != nil && (task.ParentTaskID == nil || *task.ParentTaskID != *filters.ParentTaskID) {
				continue
			}
			if filters.Status != nil && task.Status != *filters.Status {
				continue
			}
			if filters.AgentType != nil && task.AgentType != *filters.AgentType {
				continue
			}
		}
		tasks = append(tasks, task)
	}

	// Apply limit/offset if specified
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(tasks) {
			tasks = tasks[filters.Offset:]
		}
		if filters.Limit > 0 && filters.Limit < len(tasks) {
			tasks = tasks[:filters.Limit]
		}
	}

	return tasks, nil
}

// VM operations

func (m *MemoryStore) CreateVM(ctx context.Context, vm *types.VM) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[vm.ID]; exists {
		return fmt.Errorf("VM with ID '%s' already exists", vm.ID)
	}

	vm.CreatedAt = time.Now()
	m.vms[vm.ID] = vm
	return nil
}

func (m *MemoryStore) GetVM(ctx context.Context, id string) (*types.VM, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vm, exists := m.vms[id]
	if !exists {
		return nil, fmt.Errorf("VM with ID '%s' not found", id)
	}

	return vm, nil
}

func (m *MemoryStore) UpdateVM(ctx context.Context, vm *types.VM) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[vm.ID]; !exists {
		return fmt.Errorf("VM with ID '%s' not found", vm.ID)
	}

	m.vms[vm.ID] = vm
	return nil
}

func (m *MemoryStore) DeleteVM(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.vms[id]; !exists {
		return fmt.Errorf("VM with ID '%s' not found", id)
	}

	delete(m.vms, id)
	return nil
}

func (m *MemoryStore) ListVMs(ctx context.Context, filters *storage.VMFilters) ([]*types.VM, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	vms := make([]*types.VM, 0, len(m.vms))
	for _, vm := range m.vms {
		if filters != nil && filters.Status != nil && vm.Status != *filters.Status {
			continue
		}
		vms = append(vms, vm)
	}

	// Apply limit/offset if specified
	if filters != nil {
		if filters.Offset > 0 && filters.Offset < len(vms) {
			vms = vms[filters.Offset:]
		}
		if filters.Limit > 0 && filters.Limit < len(vms) {
			vms = vms[:filters.Limit]
		}
	}

	return vms, nil
}

// Health checks if the store is operational
func (m *MemoryStore) Health(ctx context.Context) error {
	return nil
}

// Close closes the store (no-op for memory store)
func (m *MemoryStore) Close() error {
	return nil
}

// Ensure MemoryStore implements StateStore interface
var _ storage.StateStore = (*MemoryStore)(nil)
