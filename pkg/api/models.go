package api

import (
	"time"

	"github.com/google/uuid"
)

// CreateVMRequest represents a VM creation request
type CreateVMRequest struct {
	Name            string            `json:"name" binding:"required"`
	VCPUs           int               `json:"vcpus" binding:"required,min=1"`
	MemoryMB        int               `json:"memory_mb" binding:"required,min=128"`
	AdditionalTools []string          `json:"additional_tools,omitempty"`
	ToolVersions    map[string]string `json:"tool_versions,omitempty"`
}

// CreateVMResponse represents a VM creation response
type CreateVMResponse struct {
	TaskID uuid.UUID `json:"task_id"`
	VMID   string    `json:"vm_id,omitempty"`
	Status string    `json:"status"`
}

// VMResponse represents a VM information response
type VMResponse struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	VCPUCount    *int              `json:"vcpu_count,omitempty"`
	MemoryMB     *int              `json:"memory_mb,omitempty"`
	KernelPath   *string           `json:"kernel_path,omitempty"`
	RootFSPath   *string           `json:"rootfs_path,omitempty"`
	SocketPath   *string           `json:"socket_path,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	StoppedAt    *time.Time        `json:"stopped_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ListVMsResponse represents a list of VMs
type ListVMsResponse struct {
	VMs   []*VMResponse `json:"vms"`
	Total int           `json:"total"`
}

// ExecuteCommandRequest represents a command execution request
type ExecuteCommandRequest struct {
	Command          string            `json:"command" binding:"required"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`              // Environment variables (persisted to DB)
	TransientSecrets map[string]string `json:"transient_secrets,omitempty"` // Secrets (NOT persisted to DB)
}

// ExecuteCommandResponse represents a command execution response
type ExecuteCommandResponse struct {
	TaskID uuid.UUID `json:"task_id"`
	VMID   string    `json:"vm_id"`
	Status string    `json:"status"`
}

// ExecutionResponse represents a command execution result
// Note: Env field is NOT included for security reasons (may contain secrets)
type ExecutionResponse struct {
	ID             uuid.UUID              `json:"id"`
	VMID           *uuid.UUID             `json:"vm_id,omitempty"`
	Command        string                 `json:"command"`
	Args           []interface{}          `json:"args,omitempty"`
	SecretRedacted bool                   `json:"secret_redacted"` // Indicates if transient secrets were used
	ExitCode       *int                   `json:"exit_code,omitempty"`
	Stdout         *string                `json:"stdout,omitempty"`
	Stderr         *string                `json:"stderr,omitempty"`
	Error          *string                `json:"error,omitempty"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	DurationMS     *int                   `json:"duration_ms,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ListExecutionsResponse represents a list of executions
type ListExecutionsResponse struct {
	Executions []*ExecutionResponse `json:"executions"`
	Total      int                  `json:"total"`
}

// TaskResponse represents a task status response
type TaskResponse struct {
	ID        uuid.UUID              `json:"id"`
	Type      string                 `json:"type"`
	Status    string                 `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// SmartExecuteRequest represents a smart command execution request
type SmartExecuteRequest struct {
	Command          string            `json:"command" binding:"required"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`              // Environment variables (persisted to DB)
	TransientSecrets map[string]string `json:"transient_secrets,omitempty"` // Secrets (NOT persisted to DB)
	VMName           string            `json:"vm_name,omitempty"`          // Optional: specific VM name
	RequiredTools    []string          `json:"required_tools,omitempty"`   // Optional: tools needed for command
	PreferExisting   bool              `json:"prefer_existing"`            // Default true: reuse existing VMs
	VCPUs            int               `json:"vcpus,omitempty"`            // For new VM if needed
	MemoryMB         int               `json:"memory_mb,omitempty"`        // For new VM if needed
}

// SmartExecuteResponse represents a smart command execution response
type SmartExecuteResponse struct {
	ExecutionID uuid.UUID `json:"execution_id"`
	VMID        uuid.UUID `json:"vm_id"`
	VMName      string    `json:"vm_name"`
	VMCreated   bool      `json:"vm_created"`    // true if new VM was created
	VMReused    bool      `json:"vm_reused"`     // true if existing VM was reused
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
}

// LogQueryRequest represents a log query request
type LogQueryRequest struct {
	VMID       string `json:"vm_id,omitempty"`
	TaskID     string `json:"task_id,omitempty"`
	Level      string `json:"level,omitempty"`
	SearchText string `json:"search_text,omitempty"`
	StartTime  *int64 `json:"start_time,omitempty"`
	EndTime    *int64 `json:"end_time,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	TaskID    string                 `json:"task_id,omitempty"`
	VMID      string                 `json:"vm_id,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// LogQueryResponse represents log query results
type LogQueryResponse struct {
	Logs  []*LogEntry `json:"logs"`
	Total int         `json:"total"`
}

// WebhookRequest represents an integration webhook request
type WebhookRequest struct {
	Integration string                 `json:"integration"`
	EventType   string                 `json:"event_type"`
	Data        map[string]interface{} `json:"data"`
	Signature   string                 `json:"signature,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status     string            `json:"status"`
	Components map[string]string `json:"components"`
	Timestamp  time.Time         `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
