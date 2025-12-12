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
	Command string   `json:"command" binding:"required"`
	Args    []string `json:"args,omitempty"`
}

// ExecuteCommandResponse represents a command execution response
type ExecuteCommandResponse struct {
	TaskID uuid.UUID `json:"task_id"`
	VMID   string    `json:"vm_id"`
	Status string    `json:"status"`
}

// ExecutionResponse represents a command execution result
type ExecutionResponse struct {
	ID          uuid.UUID              `json:"id"`
	VMID        *uuid.UUID             `json:"vm_id,omitempty"`
	Command     string                 `json:"command"`
	Args        []interface{}          `json:"args,omitempty"`
	ExitCode    *int                   `json:"exit_code,omitempty"`
	Stdout      *string                `json:"stdout,omitempty"`
	Stderr      *string                `json:"stderr,omitempty"`
	Error       *string                `json:"error,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	DurationMS  *int                   `json:"duration_ms,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	Command         string            `json:"command" binding:"required"`
	Args            []string          `json:"args,omitempty"`
	VMName          string            `json:"vm_name,omitempty"`          // Optional: specific VM name
	RequiredTools   []string          `json:"required_tools,omitempty"`   // Optional: tools needed for command
	PreferExisting  bool              `json:"prefer_existing"`            // Default true: reuse existing VMs
	VCPUs           int               `json:"vcpus,omitempty"`            // For new VM if needed
	MemoryMB        int               `json:"memory_mb,omitempty"`        // For new VM if needed
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

// Proxy-related models

// UpdateWhitelistRequest represents a request to update global whitelist
type UpdateWhitelistRequest struct {
	Domains []string `json:"domains" binding:"required"`
}

// UpdateVMWhitelistRequest represents a request to update VM-specific whitelist
type UpdateVMWhitelistRequest struct {
	Domains []string `json:"domains" binding:"required"`
}

// ProxyStatsResponse represents proxy statistics
type ProxyStatsResponse struct {
	TotalRequests   int64   `json:"total_requests"`
	BlockedRequests int64   `json:"blocked_requests"`
	CacheHitRate    float64 `json:"cache_hit_rate"`
	BytesServed     int64   `json:"bytes_served"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
}

// BlockedRequestResponse represents a blocked HTTP request
type BlockedRequestResponse struct {
	Timestamp string `json:"timestamp"`
	ClientIP  string `json:"client_ip"`
	Method    string `json:"method"`
	URL       string `json:"url"`
	Domain    string `json:"domain"`
	Reason    string `json:"reason"`
}

// BlockedRequestsResponse represents a list of blocked requests
type BlockedRequestsResponse struct {
	Requests []*BlockedRequestResponse `json:"requests"`
	Total    int                       `json:"total"`
}

// ========================================
// Workspace-related models
// ========================================

// PrepStepRequest represents a preparation step in workspace creation
type PrepStepRequest struct {
	Type   string                 `json:"type" binding:"required"` // git_clone, script, env_var
	Order  int                    `json:"order"`
	Config map[string]interface{} `json:"config" binding:"required"`
	// git_clone config: {url, branch, dest_path, ssh_key_secret_name}
	// script config: {content, interpreter, args}
	// env_var config: {key, value, secret_name, is_secret}
}

// SecretRequest represents a secret in workspace creation
type SecretRequest struct {
	Name        string `json:"name" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type"` // api_key, ssh_key, token, password, other
	Description string `json:"description,omitempty"`
}

// CreateWorkspaceRequest represents a workspace creation request
type CreateWorkspaceRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Description       string                 `json:"description,omitempty"`
	EnvironmentID     string                 `json:"environment_id,omitempty"` // Optional: reference to an environment template
	VCPUs             int                    `json:"vcpus,omitempty"`          // Optional if environment_id is set
	MemoryMB          int                    `json:"memory_mb,omitempty"`      // Optional if environment_id is set
	AIAssistant       string                 `json:"ai_assistant,omitempty"`   // claude-code or ampcode
	AIAssistantConfig map[string]interface{} `json:"ai_assistant_config,omitempty"`
	WorkingDirectory  string                 `json:"working_directory,omitempty"` // default: /workspace
	Secrets           []SecretRequest        `json:"secrets,omitempty"`
	PrepSteps         []PrepStepRequest      `json:"prep_steps,omitempty"`
	AdditionalTools   []string               `json:"additional_tools,omitempty"`
	ToolVersions      map[string]string      `json:"tool_versions,omitempty"`
}

// CreateWorkspaceResponse represents a workspace creation response
type CreateWorkspaceResponse struct {
	TaskID      uuid.UUID `json:"task_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Status      string    `json:"status"`
}

// PrepStepResponse represents a preparation step response
type PrepStepResponse struct {
	ID          uuid.UUID              `json:"id"`
	Type        string                 `json:"type"`
	Order       int                    `json:"order"`
	Config      map[string]interface{} `json:"config"`
	Status      string                 `json:"status"`
	ExitCode    *int                   `json:"exit_code,omitempty"`
	Stdout      *string                `json:"stdout,omitempty"`
	Stderr      *string                `json:"stderr,omitempty"`
	Error       *string                `json:"error,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	DurationMS  *int                   `json:"duration_ms,omitempty"`
}

// SecretResponse represents a secret response (value is never returned)
type SecretResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description,omitempty"`
	Scope       string    `json:"scope"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WorkspaceResponse represents workspace information
type WorkspaceResponse struct {
	ID                uuid.UUID              `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	EnvironmentID     *uuid.UUID             `json:"environment_id,omitempty"`
	VMID              *uuid.UUID             `json:"vm_id,omitempty"`
	VMName            string                 `json:"vm_name,omitempty"`
	Status            string                 `json:"status"`
	AIAssistant       string                 `json:"ai_assistant"`
	AIAssistantConfig map[string]interface{} `json:"ai_assistant_config,omitempty"`
	WorkingDirectory  string                 `json:"working_directory"`
	CreatedAt         time.Time              `json:"created_at"`
	ReadyAt           *time.Time             `json:"ready_at,omitempty"`
	StoppedAt         *time.Time             `json:"stopped_at,omitempty"`
	IdleSince         *time.Time             `json:"idle_since,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	// Nested data (included on detail view)
	PrepSteps []PrepStepResponse `json:"prep_steps,omitempty"`
	Secrets   []SecretResponse   `json:"secrets,omitempty"`
}

// ListWorkspacesResponse represents a list of workspaces
type ListWorkspacesResponse struct {
	Workspaces []*WorkspaceResponse `json:"workspaces"`
	Total      int                  `json:"total"`
}

// SubmitPromptRequest represents a prompt submission request
type SubmitPromptRequest struct {
	Prompt           string                 `json:"prompt" binding:"required"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	WorkingDirectory string                 `json:"working_directory,omitempty"`
	Environment      map[string]interface{} `json:"environment,omitempty"`
	Priority         int                    `json:"priority,omitempty"` // 0-10, default 5
}

// SubmitPromptResponse represents a prompt submission response
type SubmitPromptResponse struct {
	PromptID    uuid.UUID `json:"prompt_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Status      string    `json:"status"`
	Position    int       `json:"position,omitempty"` // Position in queue
}

// PromptResponse represents a prompt task response
type PromptResponse struct {
	ID               uuid.UUID              `json:"id"`
	WorkspaceID      uuid.UUID              `json:"workspace_id"`
	Prompt           string                 `json:"prompt"`
	SystemPrompt     string                 `json:"system_prompt,omitempty"`
	WorkingDirectory string                 `json:"working_directory,omitempty"`
	Environment      map[string]interface{} `json:"environment,omitempty"`
	Priority         int                    `json:"priority"`
	Status           string                 `json:"status"`
	ExitCode         *int                   `json:"exit_code,omitempty"`
	Stdout           *string                `json:"stdout,omitempty"`
	Stderr           *string                `json:"stderr,omitempty"`
	Error            *string                `json:"error,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	ScheduledAt      time.Time              `json:"scheduled_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	DurationMS       *int                   `json:"duration_ms,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ListPromptsResponse represents a list of prompts
type ListPromptsResponse struct {
	Prompts []*PromptResponse `json:"prompts"`
	Total   int               `json:"total"`
}

// SessionResponse represents an interactive session response
type SessionResponse struct {
	ID             uuid.UUID              `json:"id"`
	WorkspaceID    uuid.UUID              `json:"workspace_id"`
	Status         string                 `json:"status"`
	ClientIP       string                 `json:"client_ip,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
	ConnectedAt    time.Time              `json:"connected_at"`
	LastActivity   time.Time              `json:"last_activity"`
	DisconnectedAt *time.Time             `json:"disconnected_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// SessionMessageResponse represents a session message response
type SessionMessageResponse struct {
	ID          uuid.UUID              `json:"id"`
	SessionID   uuid.UUID              `json:"session_id"`
	MessageType string                 `json:"message_type"` // prompt, response, system, error
	Content     string                 `json:"content"`
	ExitCode    *int                   `json:"exit_code,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ListSessionMessagesResponse represents a list of session messages
type ListSessionMessagesResponse struct {
	Messages []*SessionMessageResponse `json:"messages"`
	Total    int                       `json:"total"`
}

// AddSecretRequest represents a request to add a secret to an existing workspace
type AddSecretRequest struct {
	Name        string `json:"name" binding:"required"`
	Value       string `json:"value" binding:"required"`
	Type        string `json:"type"` // api_key, ssh_key, token, password, other
	Description string `json:"description,omitempty"`
	Scope       string `json:"scope,omitempty"` // workspace (default) or global
}

// AddSecretResponse represents a response after adding a secret
type AddSecretResponse struct {
	SecretID uuid.UUID `json:"secret_id"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
}

// ListSecretsResponse represents a list of secrets
type ListSecretsResponse struct {
	Secrets []*SecretResponse `json:"secrets"`
	Total   int               `json:"total"`
}

// ========================================
// Environment-related models
// ========================================

// MCPServerRequest represents an MCP server configuration in requests
type MCPServerRequest struct {
	Name    string            `json:"name" binding:"required"`
	Type    string            `json:"type" binding:"required"` // stdio or http
	Command string            `json:"command,omitempty"`       // for stdio
	Args    []string          `json:"args,omitempty"`          // for stdio
	URL     string            `json:"url,omitempty"`           // for http
	Headers map[string]string `json:"headers,omitempty"`       // for http
	Env     map[string]string `json:"env,omitempty"`
}

// CreateEnvironmentRequest represents an environment creation request
type CreateEnvironmentRequest struct {
	Name               string             `json:"name" binding:"required"`
	Description        string             `json:"description,omitempty"`
	VCPUs              int                `json:"vcpus,omitempty"`
	MemoryMB           int                `json:"memory_mb,omitempty"`
	GitRepoURL         string             `json:"git_repo_url,omitempty"`
	GitBranch          string             `json:"git_branch,omitempty"`
	WorkingDirectory   string             `json:"working_directory,omitempty"`
	Tools              []string           `json:"tools,omitempty"`
	EnvVars            map[string]string  `json:"env_vars,omitempty"`
	MCPServers         []MCPServerRequest `json:"mcp_servers,omitempty"`
	IdleTimeoutSeconds int                `json:"idle_timeout_seconds,omitempty"`
}

// UpdateEnvironmentRequest represents an environment update request
type UpdateEnvironmentRequest struct {
	Name               string             `json:"name,omitempty"`
	Description        string             `json:"description,omitempty"`
	VCPUs              int                `json:"vcpus,omitempty"`
	MemoryMB           int                `json:"memory_mb,omitempty"`
	GitRepoURL         string             `json:"git_repo_url,omitempty"`
	GitBranch          string             `json:"git_branch,omitempty"`
	WorkingDirectory   string             `json:"working_directory,omitempty"`
	Tools              []string           `json:"tools,omitempty"`
	EnvVars            map[string]string  `json:"env_vars,omitempty"`
	MCPServers         []MCPServerRequest `json:"mcp_servers,omitempty"`
	IdleTimeoutSeconds int                `json:"idle_timeout_seconds,omitempty"`
}

// MCPServerResponse represents an MCP server configuration in responses
type MCPServerResponse struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// EnvironmentResponse represents environment information
type EnvironmentResponse struct {
	ID                 uuid.UUID           `json:"id"`
	Name               string              `json:"name"`
	Description        string              `json:"description,omitempty"`
	VCPUs              int                 `json:"vcpus"`
	MemoryMB           int                 `json:"memory_mb"`
	GitRepoURL         string              `json:"git_repo_url,omitempty"`
	GitBranch          string              `json:"git_branch"`
	WorkingDirectory   string              `json:"working_directory"`
	Tools              []string            `json:"tools"`
	EnvVars            map[string]string   `json:"env_vars,omitempty"`
	MCPServers         []MCPServerResponse `json:"mcp_servers,omitempty"`
	IdleTimeoutSeconds int                 `json:"idle_timeout_seconds"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

// ListEnvironmentsResponse represents a list of environments
type ListEnvironmentsResponse struct {
	Environments []*EnvironmentResponse `json:"environments"`
	Total        int                    `json:"total"`
}
