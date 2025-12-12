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
	ID          uuid.UUID  `db:"id" json:"id"`
	JobID       *uuid.UUID `db:"job_id" json:"job_id,omitempty"`
	VMID        *uuid.UUID `db:"vm_id" json:"vm_id,omitempty"`
	Command     string     `db:"command" json:"command"`
	Args        JSONBArray `db:"args" json:"args,omitempty"`
	Env         JSONB      `db:"env" json:"env,omitempty"`
	ExitCode    *int       `db:"exit_code" json:"exit_code,omitempty"`
	Stdout      *string    `db:"stdout" json:"stdout,omitempty"`
	Stderr      *string    `db:"stderr" json:"stderr,omitempty"`
	Error       *string    `db:"error" json:"error,omitempty"`
	StartedAt   time.Time  `db:"started_at" json:"started_at"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS  *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	Metadata    JSONB      `db:"metadata" json:"metadata"`
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

// Workspace represents an AI workspace that extends a VM
type Workspace struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	Name              string     `db:"name" json:"name"`
	Description       *string    `db:"description" json:"description,omitempty"`
	VMID              *uuid.UUID `db:"vm_id" json:"vm_id,omitempty"`
	EnvironmentID     *uuid.UUID `db:"environment_id" json:"environment_id,omitempty"`
	Status            string     `db:"status" json:"status"`
	AIAssistant       string     `db:"ai_assistant" json:"ai_assistant"`
	AIAssistantConfig JSONB      `db:"ai_assistant_config" json:"ai_assistant_config"`
	WorkingDirectory  string     `db:"working_directory" json:"working_directory"`
	IdleSince         *time.Time `db:"idle_since" json:"idle_since,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	ReadyAt           *time.Time `db:"ready_at" json:"ready_at,omitempty"`
	StoppedAt         *time.Time `db:"stopped_at" json:"stopped_at,omitempty"`
	Metadata          JSONB      `db:"metadata" json:"metadata"`
}

// WorkspaceSecret represents an encrypted secret for a workspace
type WorkspaceSecret struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	WorkspaceID     *uuid.UUID `db:"workspace_id" json:"workspace_id,omitempty"`
	Name            string     `db:"name" json:"name"`
	Description     *string    `db:"description" json:"description,omitempty"`
	SecretType      string     `db:"secret_type" json:"secret_type"`
	EncryptedValue  []byte     `db:"encrypted_value" json:"-"` // Never expose in JSON
	EncryptionKeyID string     `db:"encryption_key_id" json:"-"`
	Nonce           []byte     `db:"nonce" json:"-"`
	Scope           string     `db:"scope" json:"scope"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// PrepStep represents a workspace preparation step
type PrepStep struct {
	ID          uuid.UUID  `db:"id" json:"id"`
	WorkspaceID uuid.UUID  `db:"workspace_id" json:"workspace_id"`
	StepType    string     `db:"step_type" json:"step_type"`
	StepOrder   int        `db:"step_order" json:"step_order"`
	Config      JSONB      `db:"config" json:"config"`
	Status      string     `db:"status" json:"status"`
	ExitCode    *int       `db:"exit_code" json:"exit_code,omitempty"`
	Stdout      *string    `db:"stdout" json:"stdout,omitempty"`
	Stderr      *string    `db:"stderr" json:"stderr,omitempty"`
	Error       *string    `db:"error" json:"error,omitempty"`
	StartedAt   *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS  *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	Metadata    JSONB      `db:"metadata" json:"metadata"`
}

// PrepStepResult holds execution results for a prep step
type PrepStepResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Error      string
	DurationMS int
}

// PromptTask represents a queued prompt for execution
type PromptTask struct {
	ID               uuid.UUID  `db:"id" json:"id"`
	WorkspaceID      uuid.UUID  `db:"workspace_id" json:"workspace_id"`
	Prompt           string     `db:"prompt" json:"prompt"`
	SystemPrompt     *string    `db:"system_prompt" json:"system_prompt,omitempty"`
	WorkingDirectory *string    `db:"working_directory" json:"working_directory,omitempty"`
	Environment      JSONB      `db:"environment" json:"environment"`
	Priority         int        `db:"priority" json:"priority"`
	Status           string     `db:"status" json:"status"`
	ExitCode         *int       `db:"exit_code" json:"exit_code,omitempty"`
	Stdout           *string    `db:"stdout" json:"stdout,omitempty"`
	Stderr           *string    `db:"stderr" json:"stderr,omitempty"`
	Error            *string    `db:"error" json:"error,omitempty"`
	CreatedAt        time.Time  `db:"created_at" json:"created_at"`
	ScheduledAt      time.Time  `db:"scheduled_at" json:"scheduled_at"`
	StartedAt        *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt      *time.Time `db:"completed_at" json:"completed_at,omitempty"`
	DurationMS       *int       `db:"duration_ms" json:"duration_ms,omitempty"`
	Metadata         JSONB      `db:"metadata" json:"metadata"`
}

// PromptResult holds execution results for a prompt
type PromptResult struct {
	ExitCode   int
	Stdout     string
	Stderr     string
	Error      string
	DurationMS int
}

// WorkspaceSession represents a WebSocket session
type WorkspaceSession struct {
	ID             uuid.UUID  `db:"id" json:"id"`
	WorkspaceID    uuid.UUID  `db:"workspace_id" json:"workspace_id"`
	Status         string     `db:"status" json:"status"`
	ClientIP       *string    `db:"client_ip" json:"client_ip,omitempty"`
	UserAgent      *string    `db:"user_agent" json:"user_agent,omitempty"`
	ConnectedAt    time.Time  `db:"connected_at" json:"connected_at"`
	LastActivity   time.Time  `db:"last_activity" json:"last_activity"`
	DisconnectedAt *time.Time `db:"disconnected_at" json:"disconnected_at,omitempty"`
	Metadata       JSONB      `db:"metadata" json:"metadata"`
}

// SessionMessage represents a message in an interactive session
type SessionMessage struct {
	ID          uuid.UUID `db:"id" json:"id"`
	SessionID   uuid.UUID `db:"session_id" json:"session_id"`
	MessageType string    `db:"message_type" json:"message_type"`
	Content     string    `db:"content" json:"content"`
	ExitCode    *int      `db:"exit_code" json:"exit_code,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	Metadata    JSONB     `db:"metadata" json:"metadata"`
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

// WorkspaceRepository handles workspace storage operations
type WorkspaceRepository interface {
	Create(ctx context.Context, workspace *Workspace) error
	Get(ctx context.Context, id uuid.UUID) (*Workspace, error)
	GetByName(ctx context.Context, name string) (*Workspace, error)
	GetByVMID(ctx context.Context, vmID uuid.UUID) (*Workspace, error)
	List(ctx context.Context, filters map[string]interface{}) ([]*Workspace, error)
	ListByEnvironment(ctx context.Context, environmentID uuid.UUID) ([]*Workspace, error)
	ListIdleWithVMs(ctx context.Context) ([]*Workspace, error)
	Update(ctx context.Context, workspace *Workspace) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateIdleSince(ctx context.Context, id uuid.UUID, idleSince *time.Time) error
	SetVMID(ctx context.Context, id uuid.UUID, vmID uuid.UUID) error
	ClearVMID(ctx context.Context, id uuid.UUID) error
	SetEnvironmentID(ctx context.Context, id uuid.UUID, environmentID uuid.UUID) error
	SetReady(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SecretRepository handles workspace secret storage operations
type SecretRepository interface {
	Create(ctx context.Context, secret *WorkspaceSecret) error
	Get(ctx context.Context, id uuid.UUID) (*WorkspaceSecret, error)
	GetByName(ctx context.Context, workspaceID *uuid.UUID, name string) (*WorkspaceSecret, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*WorkspaceSecret, error)
	ListGlobal(ctx context.Context) ([]*WorkspaceSecret, error)
	Update(ctx context.Context, secret *WorkspaceSecret) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PrepStepRepository handles workspace preparation step storage operations
type PrepStepRepository interface {
	Create(ctx context.Context, step *PrepStep) error
	CreateBatch(ctx context.Context, steps []*PrepStep) error
	Get(ctx context.Context, id uuid.UUID) (*PrepStep, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*PrepStep, error)
	GetNextPending(ctx context.Context, workspaceID uuid.UUID) (*PrepStep, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, result *PrepStepResult) error
}

// PromptTaskRepository handles prompt task storage operations
type PromptTaskRepository interface {
	Create(ctx context.Context, task *PromptTask) error
	Get(ctx context.Context, id uuid.UUID) (*PromptTask, error)
	ListByWorkspace(ctx context.Context, workspaceID uuid.UUID, limit int) ([]*PromptTask, error)
	GetNextPending(ctx context.Context, workspaceID uuid.UUID) (*PromptTask, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string, result *PromptResult) error
	Cancel(ctx context.Context, id uuid.UUID) error
}

// SessionRepository handles workspace session storage operations
type SessionRepository interface {
	Create(ctx context.Context, session *WorkspaceSession) error
	Get(ctx context.Context, id uuid.UUID) (*WorkspaceSession, error)
	GetActiveByWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]*WorkspaceSession, error)
	UpdateLastActivity(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SessionMessageRepository handles session message storage operations
type SessionMessageRepository interface {
	Create(ctx context.Context, message *SessionMessage) error
	ListBySession(ctx context.Context, sessionID uuid.UUID, limit int) ([]*SessionMessage, error)
}

// Store provides access to all repositories
type Store interface {
	VMs() VMRepository
	Tasks() TaskRepository
	Jobs() JobRepository
	Executions() ExecutionRepository
	Workers() WorkerRepository
	WorkerMetrics() WorkerMetricRepository
	Environments() EnvironmentRepository
	Workspaces() WorkspaceRepository
	Secrets() SecretRepository
	PrepSteps() PrepStepRepository
	PromptTasks() PromptTaskRepository
	Sessions() SessionRepository
	SessionMessages() SessionMessageRepository
	Close() error
}
