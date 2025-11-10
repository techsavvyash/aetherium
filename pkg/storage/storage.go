package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// VM represents a virtual machine in the database
type VM struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Orchestrator string     `db:"orchestrator" json:"orchestrator"`
	Status       string     `db:"status" json:"status"`
	KernelPath   *string    `db:"kernel_path" json:"kernel_path,omitempty"`
	RootFSPath   *string    `db:"rootfs_path" json:"rootfs_path,omitempty"`
	SocketPath   *string    `db:"socket_path" json:"socket_path,omitempty"`
	VCPUCount    *int       `db:"vcpu_count" json:"vcpu_count,omitempty"`
	MemoryMB     *int       `db:"memory_mb" json:"memory_mb,omitempty"`
	WorkerID     *string    `db:"worker_id" json:"worker_id,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	StoppedAt    *time.Time `db:"stopped_at" json:"stopped_at,omitempty"`
	Metadata     JSONB      `db:"metadata" json:"metadata"`
}

// Task represents a distributed task in the queue
type Task struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Type        string     `db:"type" json:"type"`
	Status      string     `db:"status" json:"status"`
	Priority    int        `db:"priority" json:"priority"`
	Payload     JSONB      `db:"payload" json:"payload"`
	Result      JSONB      `db:"result" json:"result,omitempty"`
	Error       *string    `db:"error" json:"error,omitempty"`
	VMID        *uuid.UUID `db:"vm_id" json:"vm_id,omitempty"`
	WorkerID    *string    `db:"worker_id" json:"worker_id,omitempty"`
	MaxRetries  int        `db:"max_retries" json:"max_retries"`
	RetryCount  int        `db:"retry_count" json:"retry_count"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	ScheduledAt time.Time  `db:"scheduled_at" json:"scheduled_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	Metadata    JSONB      `db:"metadata" json:"metadata"`
}

// Job represents an execution job
type Job struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	Status      string     `db:"status" json:"status"`
	VMID        *uuid.UUID `db:"vm_id" json:"vm_id,omitempty"`
	Commands    JSONBArray `db:"commands" json:"commands"`
	Results     JSONB      `db:"results" json:"results,omitempty"`
	Error       *string    `db:"error" json:"error,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	Metadata    JSONB      `db:"metadata" json:"metadata"`
}

// Execution represents a command execution
type Execution struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	JobID          *uuid.UUID `db:"job_id" json:"job_id,omitempty"`
	VMID           *uuid.UUID `db:"vm_id" json:"vm_id,omitempty"`
	Command        string     `db:"command" json:"command"`
	Args           JSONBArray `db:"args" json:"args,omitempty"`
	Env            JSONB      `db:"env" json:"env,omitempty"`
	SecretRedacted bool       `db:"secret_redacted" json:"secret_redacted"` // Indicates if transient secrets were used
	ExitCode       *int       `db:"exit_code" json:"exit_code,omitempty"`
	Stdout         *string    `db:"stdout" json:"stdout,omitempty"`
	Stderr         *string    `db:"stderr" json:"stderr,omitempty"`
	Error          *string    `db:"error" json:"error,omitempty"`
	StartedAt      time.Time  `db:"started_at" json:"started_at"`
	CompletedAt    *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS     *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	Metadata       JSONB      `db:"metadata" json:"metadata"`
}

// Worker represents a distributed worker node in the database
type Worker struct {
	ID       string    `db:"id" json:"id"`
	Hostname string    `db:"hostname" json:"hostname"`
	Address  string    `db:"address" json:"address"`

	// Status
	Status    string     `db:"status" json:"status"`
	LastSeen  time.Time  `db:"last_seen" json:"last_seen"`
	StartedAt time.Time  `db:"started_at" json:"started_at"`

	// Location
	Zone   string `db:"zone" json:"zone"`
	Labels JSONB  `db:"labels" json:"labels"`

	// Capabilities
	Capabilities JSONBArray `db:"capabilities" json:"capabilities"`

	// Resources
	CPUCores      int   `db:"cpu_cores" json:"cpu_cores"`
	MemoryMB      int64 `db:"memory_mb" json:"memory_mb"`
	DiskGB        int64 `db:"disk_gb" json:"disk_gb"`
	UsedCPUCores  int   `db:"used_cpu_cores" json:"used_cpu_cores"`
	UsedMemoryMB  int64 `db:"used_memory_mb" json:"used_memory_mb"`
	UsedDiskGB    int64 `db:"used_disk_gb" json:"used_disk_gb"`
	VMCount       int   `db:"vm_count" json:"vm_count"`
	MaxVMs        int   `db:"max_vms" json:"max_vms"`

	// Metadata
	Metadata  JSONB     `db:"metadata" json:"metadata"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// WorkerMetric represents worker metrics at a point in time
type WorkerMetric struct {
	ID             uuid.UUID `db:"id" json:"id"`
	WorkerID       string    `db:"worker_id" json:"worker_id"`
	Timestamp      time.Time `db:"timestamp" json:"timestamp"`

	// Resource metrics
	CPUUsage      float64 `db:"cpu_usage" json:"cpu_usage"`
	MemoryUsage   float64 `db:"memory_usage" json:"memory_usage"`
	DiskUsage     float64 `db:"disk_usage" json:"disk_usage"`

	// VM metrics
	VMCount        int `db:"vm_count" json:"vm_count"`
	TasksProcessed int `db:"tasks_processed" json:"tasks_processed"`

	// Network metrics (optional)
	NetworkInMB  *float64               `db:"network_in_mb" json:"network_in_mb,omitempty"`
	NetworkOutMB *float64               `db:"network_out_mb" json:"network_out_mb,omitempty"`

	Metadata map[string]interface{} `db:"metadata" json:"metadata"`
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

// WorkerRepository handles worker storage operations
type WorkerRepository interface {
	Create(ctx context.Context, worker *Worker) error
	Get(ctx context.Context, id string) (*Worker, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*Worker, error)
	Update(ctx context.Context, worker *Worker) error
	UpdateResources(ctx context.Context, id string, resources map[string]interface{}) error
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateLastSeen(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	ListByZone(ctx context.Context, zone string) ([]*Worker, error)
	ListActive(ctx context.Context) ([]*Worker, error)
}

// WorkerMetricRepository handles worker metrics storage operations
type WorkerMetricRepository interface {
	Create(ctx context.Context, metric *WorkerMetric) error
	Get(ctx context.Context, id uuid.UUID) (*WorkerMetric, error)
	ListByWorker(ctx context.Context, workerID string, limit int) ([]*WorkerMetric, error)
	ListByWorkerInTimeRange(ctx context.Context, workerID string, start, end time.Time) ([]*WorkerMetric, error)
	DeleteOlderThan(ctx context.Context, before time.Time) error
}

// Store provides access to all repositories
type Store interface {
	VMs() VMRepository
	Tasks() TaskRepository
	Jobs() JobRepository
	Executions() ExecutionRepository
	Workers() WorkerRepository
	WorkerMetrics() WorkerMetricRepository
	Close() error
}
