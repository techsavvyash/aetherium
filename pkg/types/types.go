package types

import "time"

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusRunning    TaskStatus = "RUNNING"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
	TaskStatusTerminated TaskStatus = "TERMINATED"
)

// Task represents an agent execution task
type Task struct {
	ID           string                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	ParentTaskID *string                `json:"parent_task_id,omitempty"`
	Type         string                 `json:"type"` // Type of task (e.g., "code_review", "deploy", etc.)
	Status       TaskStatus             `json:"status"`
	Description  string                 `json:"description,omitempty"`
	AgentType    string                 `json:"agent_type"`
	Prompt       string                 `json:"prompt"`
	ContainerID  *string                `json:"container_id,omitempty"`
	VMID         *string                `json:"vm_id,omitempty"`
	PullRequest  *string                `json:"pull_request,omitempty"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
}

// ProjectStatus represents the current state of a project
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "ACTIVE"
	ProjectStatusArchived ProjectStatus = "ARCHIVED"
	ProjectStatusPaused   ProjectStatus = "PAUSED"
)

// Project represents a workspace for organizing tasks
type Project struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	RepoURL     string        `json:"repo_url"`
	Status      ProjectStatus `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// VMConfig represents configuration for a VM instance
type VMConfig struct {
	ID              string            `json:"id"`
	KernelPath      string            `json:"kernel_path"`
	RootFSPath      string            `json:"rootfs_path"`
	SocketPath      string            `json:"socket_path"`
	VCPUCount       int               `json:"vcpu_count"`
	MemoryMB        int               `json:"memory_mb"`
	Env             map[string]string `json:"env,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	DefaultTools    []string          `json:"default_tools,omitempty"`    // Tools installed in all VMs (e.g., nodejs, bun, claude-code)
	AdditionalTools []string          `json:"additional_tools,omitempty"` // Per-request tools (e.g., go, python)
	ToolVersions    map[string]string `json:"tool_versions,omitempty"`    // Tool version specifications
}

// VMStatus represents the current state of a VM
type VMStatus string

const (
	VMStatusCreated  VMStatus = "CREATED"
	VMStatusStarting VMStatus = "STARTING"
	VMStatusRunning  VMStatus = "RUNNING"
	VMStatusStopping VMStatus = "STOPPING"
	VMStatusStopped  VMStatus = "STOPPED"
	VMStatusFailed   VMStatus = "FAILED"
)

// VM represents a virtual machine instance
type VM struct {
	ID         string            `json:"id"`
	Status     VMStatus          `json:"status"`
	Config     VMConfig          `json:"config"`
	CreatedAt  time.Time         `json:"created_at"`
	StartedAt  *time.Time        `json:"started_at,omitempty"`
	StoppedAt  *time.Time        `json:"stopped_at,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

// LogEntry represents a single log message
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Message   string                 `json:"message"`
	TaskID    string                 `json:"task_id,omitempty"`
	VMID      string                 `json:"vm_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Event represents a system event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Notification represents a notification to be sent via integrations
type Notification struct {
	Type    string                 `json:"type"`
	Target  string                 `json:"target"` // channel, user, etc.
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// Artifact represents an output artifact from a task
type Artifact struct {
	Type    string                 `json:"type"` // pull_request, file, etc.
	Content map[string]interface{} `json:"content"`
}

// TaskFilter represents filters for querying tasks
type TaskFilter struct {
	ProjectID string
	Status    TaskStatus
	Type      string
}

// ProjectFilter represents filters for querying projects
type ProjectFilter struct {
	Status ProjectStatus
}

// VMFilter represents filters for querying VMs
type VMFilter struct {
	Status VMStatus
}
