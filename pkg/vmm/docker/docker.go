package docker

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
)

// DockerOrchestrator implements vmm.VMOrchestrator using Docker
// This is a simpler alternative to Firecracker for testing
type DockerOrchestrator struct {
	config *Config
	vms    map[string]*vmHandle
}

type Config struct {
	Network string
	Image   string
}

type vmHandle struct {
	containerID string
	vm          *types.VM
}

// NewDockerOrchestrator creates a new Docker-based orchestrator
func NewDockerOrchestrator(configMap map[string]interface{}) (*DockerOrchestrator, error) {
	config := &Config{
		Network: getStringOrDefault(configMap, "network", "bridge"),
		Image:   getStringOrDefault(configMap, "image", "ubuntu:22.04"),
	}

	return &DockerOrchestrator{
		config: config,
		vms:    make(map[string]*vmHandle),
	}, nil
}

// CreateVM creates a new Docker container
func (d *DockerOrchestrator) CreateVM(ctx context.Context, config *types.VMConfig) (*types.VM, error) {
	// Use docker run with sleep infinity to keep container alive
	cmd := exec.CommandContext(ctx, "docker", "run",
		"-d",                     // Detached
		"--name", config.ID,      // Container name
		"--network", d.config.Network,
		d.config.Image,
		"sleep", "infinity") // Keep alive

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w, output: %s", err, string(output))
	}

	containerID := strings.TrimSpace(string(output))

	vm := &types.VM{
		ID:        config.ID,
		Status:    types.VMStatusCreated,
		Config:    *config,
		CreatedAt: time.Now(),
	}

	d.vms[config.ID] = &vmHandle{
		containerID: containerID,
		vm:          vm,
	}

	return vm, nil
}

// StartVM starts a Docker container (already started in CreateVM)
func (d *DockerOrchestrator) StartVM(ctx context.Context, vmID string) error {
	handle, exists := d.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Container is already running, just update status
	handle.vm.Status = types.VMStatusRunning
	now := time.Now()
	handle.vm.StartedAt = &now

	return nil
}

// StopVM stops a Docker container
func (d *DockerOrchestrator) StopVM(ctx context.Context, vmID string, force bool) error {
	handle, exists := d.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	var cmd *exec.Cmd
	if force {
		cmd = exec.CommandContext(ctx, "docker", "kill", handle.containerID)
	} else {
		cmd = exec.CommandContext(ctx, "docker", "stop", handle.containerID)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	handle.vm.Status = types.VMStatusStopped
	now := time.Now()
	handle.vm.StoppedAt = &now

	return nil
}

// GetVMStatus returns the current status of a VM
func (d *DockerOrchestrator) GetVMStatus(ctx context.Context, vmID string) (*types.VM, error) {
	handle, exists := d.vms[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Check actual container status
	cmd := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Status}}", handle.containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container status: %w", err)
	}

	status := strings.TrimSpace(string(output))
	switch status {
	case "running":
		handle.vm.Status = types.VMStatusRunning
	case "exited":
		handle.vm.Status = types.VMStatusStopped
	default:
		handle.vm.Status = types.VMStatus(strings.ToUpper(status))
	}

	return handle.vm, nil
}

// DeleteVM removes a Docker container
func (d *DockerOrchestrator) DeleteVM(ctx context.Context, vmID string) error {
	handle, exists := d.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", handle.containerID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	delete(d.vms, vmID)
	return nil
}

// ListVMs returns all VMs
func (d *DockerOrchestrator) ListVMs(ctx context.Context) ([]*types.VM, error) {
	vms := make([]*types.VM, 0, len(d.vms))
	for _, handle := range d.vms {
		vms = append(vms, handle.vm)
	}
	return vms, nil
}

// StreamLogs streams logs from a container
func (d *DockerOrchestrator) StreamLogs(ctx context.Context, vmID string) (<-chan string, error) {
	handle, exists := d.vms[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", handle.containerID)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start log streaming: %w", err)
	}

	logChan := make(chan string, 100)

	go func() {
		defer close(logChan)
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				logChan <- string(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	return logChan, nil
}

// ExecuteCommand executes a command in a Docker container
func (d *DockerOrchestrator) ExecuteCommand(ctx context.Context, vmID string, cmd *vmm.Command) (*vmm.ExecResult, error) {
	_, exists := d.vms[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	// Build docker exec command - use VM ID (container name) instead of container ID
	args := []string{"exec", vmID, cmd.Cmd}
	args = append(args, cmd.Args...)

	execCmd := exec.CommandContext(ctx, "docker", args...)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &vmm.ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// ExecuteCommandStream executes a command in a Docker container with streaming output
// This is a simple wrapper around ExecuteCommand that emits a single chunk
func (d *DockerOrchestrator) ExecuteCommandStream(ctx context.Context, vmID string, cmd *vmm.Command, handler vmm.StreamHandler) error {
	result, err := d.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return err
	}

	// Send stdout if present
	if result.Stdout != "" {
		handler(&vmm.ExecStreamChunk{Stdout: result.Stdout})
	}

	// Send stderr if present
	if result.Stderr != "" {
		handler(&vmm.ExecStreamChunk{Stderr: result.Stderr})
	}

	// Send exit code
	handler(&vmm.ExecStreamChunk{ExitCode: &result.ExitCode})

	return nil
}

// Health returns the health status of the orchestrator
func (d *DockerOrchestrator) Health(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "info")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available: %w", err)
	}
	return nil
}

func getStringOrDefault(m map[string]interface{}, key, defaultVal string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultVal
}

// Ensure DockerOrchestrator implements vmm.VMOrchestrator
var _ vmm.VMOrchestrator = (*DockerOrchestrator)(nil)
