package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// VM represents a virtual machine in the database
type VM struct {
	ID           uuid.UUID              `db:"id" json:"id"`
	Name         string                 `db:"name" json:"name"`
	Orchestrator string                 `db:"orchestrator" json:"orchestrator"`
	Status       string                 `db:"status" json:"status"`
	KernelPath   *string                `db:"kernel_path" json:"kernel_path,omitempty"`
	RootFSPath   *string                `db:"rootfs_path" json:"rootfs_path,omitempty"`
	SocketPath   *string                `db:"socket_path" json:"socket_path,omitempty"`
	VCPUCount    *int                   `db:"vcpu_count" json:"vcpu_count,omitempty"`
	MemoryMB     *int                   `db:"memory_mb" json:"memory_mb,omitempty"`
	CreatedAt    time.Time              `db:"created_at" json:"created_at"`
	StartedAt    *time.Time             `db:"started_at" json:"started_at,omitempty"`
	StoppedAt    *time.Time             `db:"stopped_at" json:"stopped_at,omitempty"`
	Metadata     map[string]interface{} `db:"metadata" json:"metadata"`
}

// Task represents a distributed task in the queue
type Task struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	Type        string                 `db:"type" json:"type"`
	Status      string                 `db:"status" json:"status"`
	Priority    int                    `db:"priority" json:"priority"`
	Payload     map[string]interface{} `db:"payload" json:"payload"`
	Result      map[string]interface{} `db:"result" json:"result,omitempty"`
	Error       *string                `db:"error" json:"error,omitempty"`
	VMID        *uuid.UUID             `db:"vm_id" json:"vm_id,omitempty"`
	WorkerID    *string                `db:"worker_id" json:"worker_id,omitempty"`
	MaxRetries  int                    `db:"max_retries" json:"max_retries"`
	RetryCount  int                    `db:"retry_count" json:"retry_count"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	ScheduledAt time.Time              `db:"scheduled_at" json:"scheduled_at"`
	StartedAt   *time.Time             `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata"`
}

// Job represents an execution job
type Job struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	Name        string                 `db:"name" json:"name"`
	Status      string                 `db:"status" json:"status"`
	VMID        *uuid.UUID             `db:"vm_id" json:"vm_id,omitempty"`
	Commands    []interface{}          `db:"commands" json:"commands"`
	Results     map[string]interface{} `db:"results" json:"results,omitempty"`
	Error       *string                `db:"error" json:"error,omitempty"`
	CreatedAt   time.Time              `db:"created_at" json:"created_at"`
	StartedAt   *time.Time             `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata"`
}

// Execution represents a command execution
type Execution struct {
	ID          uuid.UUID              `db:"id" json:"id"`
	JobID       *uuid.UUID             `db:"job_id" json:"job_id,omitempty"`
	VMID        *uuid.UUID             `db:"vm_id" json:"vm_id,omitempty"`
	Command     string                 `db:"command" json:"command"`
	Args        []interface{}          `db:"args" json:"args,omitempty"`
	Env         map[string]interface{} `db:"env" json:"env,omitempty"`
	ExitCode    *int                   `db:"exit_code" json:"exit_code,omitempty"`
	Stdout      *string                `db:"stdout" json:"stdout,omitempty"`
	Stderr      *string                `db:"stderr" json:"stderr,omitempty"`
	Error       *string                `db:"error" json:"error,omitempty"`
	StartedAt   time.Time              `db:"started_at" json:"started_at"`
	CompletedAt *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS  *int                   `db:"duration_ms" json:"duration_ms,omitempty"`
	Metadata    map[string]interface{} `db:"metadata" json:"metadata"`
}

// VMRepository handles VM storage operations
type VMRepository interface {
	Create(ctx context.Context, vm *VM) error
	Get(ctx context.Context, id uuid.UUID) (*VM, error)
	GetByName(ctx context.Context, name string) (*VM, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*VM, error)
	Update(ctx context.Context, vm *VM) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TaskRepository handles task storage operations
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	Get(ctx context.Context, id uuid.UUID) (*Task, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetNextPending(ctx context.Context) (*Task, error)
	MarkProcessing(ctx context.Context, id uuid.UUID, workerID string) error
	MarkCompleted(ctx context.Context, id uuid.UUID, result map[string]interface{}) error
	MarkFailed(ctx context.Context, id uuid.UUID, err error) error
}

// JobRepository handles job storage operations
type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	Get(ctx context.Context, id uuid.UUID) (*Job, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*Job, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// ExecutionRepository handles execution storage operations
type ExecutionRepository interface {
	Create(ctx context.Context, execution *Execution) error
	Get(ctx context.Context, id uuid.UUID) (*Execution, error)
	ListByJob(ctx context.Context, jobID uuid.UUID) ([]*Execution, error)
	ListByVM(ctx context.Context, vmID uuid.UUID) ([]*Execution, error)
}

// Store provides access to all repositories
type Store interface {
	VMs() VMRepository
	Tasks() TaskRepository
	Jobs() JobRepository
	Executions() ExecutionRepository
	Close() error
}
