package vmm

import (
	"context"

	"github.com/aetherium/aetherium/pkg/types"
)

// VMOrchestrator defines the interface for VM orchestration implementations
type VMOrchestrator interface {
	// CreateVM creates a new VM with the given configuration
	CreateVM(ctx context.Context, config *types.VMConfig) (*types.VM, error)

	// StartVM boots a created VM
	StartVM(ctx context.Context, vmID string) error

	// StopVM gracefully shuts down a VM
	// If force is true, forcefully terminates the VM
	StopVM(ctx context.Context, vmID string, force bool) error

	// GetVMStatus returns the current status of a VM
	GetVMStatus(ctx context.Context, vmID string) (*types.VM, error)

	// StreamLogs streams logs from a VM
	StreamLogs(ctx context.Context, vmID string) (<-chan string, error)

	// ExecuteCommand executes a command inside a VM
	ExecuteCommand(ctx context.Context, vmID string, cmd *Command) (*ExecResult, error)

	// DeleteVM destroys a VM and cleans up resources
	DeleteVM(ctx context.Context, vmID string) error

	// ListVMs returns all VMs
	ListVMs(ctx context.Context) ([]*types.VM, error)

	// Health returns the health status of the orchestrator
	Health(ctx context.Context) error
}

// Command represents a command to execute in a VM
type Command struct {
	Cmd              string            `json:"cmd"`
	Args             []string          `json:"args,omitempty"`
	Env              map[string]string `json:"env,omitempty"`
	TransientSecrets map[string]string `json:"transient_secrets,omitempty"` // Secrets that won't be persisted to DB
}

// ExecResult represents the result of a command execution
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Config represents VMM configuration
type Config struct {
	// Provider-specific configuration
	Options map[string]interface{}
}
