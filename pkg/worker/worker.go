package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/tools"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/google/uuid"
)

// Worker handles task execution
type Worker struct {
	store        storage.Store
	orchestrator vmm.VMOrchestrator
	toolInstaller *tools.Installer
}

// New creates a new worker
func New(store storage.Store, orchestrator vmm.VMOrchestrator) *Worker {
	return &Worker{
		store:         store,
		orchestrator:  orchestrator,
		toolInstaller: tools.NewInstaller(orchestrator),
	}
}

// RegisterHandlers registers task handlers with the queue
func (w *Worker) RegisterHandlers(q queue.Queue) error {
	if err := q.RegisterHandler(queue.TaskTypeVMCreate, w.HandleVMCreate); err != nil {
		return fmt.Errorf("failed to register VM create handler: %w", err)
	}

	if err := q.RegisterHandler(queue.TaskTypeVMExecute, w.HandleVMExecute); err != nil {
		return fmt.Errorf("failed to register VM execute handler: %w", err)
	}

	if err := q.RegisterHandler(queue.TaskTypeVMDelete, w.HandleVMDelete); err != nil {
		return fmt.Errorf("failed to register VM delete handler: %w", err)
	}

	return nil
}

// VMCreatePayload represents VM creation task payload
type VMCreatePayload struct {
	Name            string            `json:"name"`
	VCPUs           int               `json:"vcpus"`
	MemoryMB        int               `json:"memory_mb"`
	AdditionalTools []string          `json:"additional_tools,omitempty"`
	ToolVersions    map[string]string `json:"tool_versions,omitempty"`
}

// VMExecutePayload represents command execution task payload
type VMExecutePayload struct {
	VMID    string   `json:"vm_id"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// HandleVMCreate handles VM creation tasks
func (w *Worker) HandleVMCreate(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	var payload VMCreatePayload
	if err := queue.UnmarshalPayload(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	log.Printf("Creating VM: %s (vcpu=%d, mem=%dMB)", payload.Name, payload.VCPUs, payload.MemoryMB)

	// Create VM config
	vmID := uuid.New().String()
	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: fmt.Sprintf("/tmp/aetherium-vm-%s.sock", vmID),
		VCPUCount:  payload.VCPUs,
		MemoryMB:   payload.MemoryMB,
	}

	// Create VM using orchestrator
	vm, err := w.orchestrator.CreateVM(ctx, vmConfig)
	if err != nil {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Start VM
	if err := w.orchestrator.StartVM(ctx, vm.ID); err != nil {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     fmt.Sprintf("failed to start VM: %v", err),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Wait for agent to be ready
	time.Sleep(5 * time.Second)

	// Install tools
	log.Printf("Installing tools in VM %s...", vm.ID)

	// Combine default tools with additional tools
	defaultTools := tools.GetDefaultTools()
	allTools := append(defaultTools, payload.AdditionalTools...)

	// Remove duplicates
	toolSet := make(map[string]bool)
	uniqueTools := []string{}
	for _, tool := range allTools {
		if !toolSet[tool] {
			toolSet[tool] = true
			uniqueTools = append(uniqueTools, tool)
		}
	}

	// Install tools with timeout (20 minutes)
	if len(uniqueTools) > 0 {
		toolVersions := payload.ToolVersions
		if toolVersions == nil {
			toolVersions = make(map[string]string)
		}

		if err := w.toolInstaller.InstallToolsWithTimeout(ctx, vm.ID, uniqueTools, toolVersions, 20*time.Minute); err != nil {
			log.Printf("Warning: Tool installation failed (VM still usable): %v", err)
			// Don't fail the task, but log the error
		} else {
			log.Printf("✓ All tools installed successfully in VM %s", vm.ID)
		}
	}

	// Store VM in database
	vmUUID, _ := uuid.Parse(vm.ID)
	kernelPath := vmConfig.KernelPath
	rootfsPath := vmConfig.RootFSPath
	socketPath := vmConfig.SocketPath

	dbVM := &storage.VM{
		ID:           vmUUID,
		Name:         payload.Name,
		Orchestrator: "firecracker",
		Status:       string(vm.Status),
		KernelPath:   &kernelPath,
		RootFSPath:   &rootfsPath,
		SocketPath:   &socketPath,
		VCPUCount:    &payload.VCPUs,
		MemoryMB:     &payload.MemoryMB,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	if err := w.store.VMs().Create(ctx, dbVM); err != nil {
		log.Printf("Warning: Failed to store VM in database: %v", err)
	}

	log.Printf("✓ VM created successfully: %s (id=%s)", payload.Name, vm.ID)

	result := map[string]interface{}{
		"vm_id":  vm.ID,
		"name":   payload.Name,
		"status": "running",
	}

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   true,
		Result:    result,
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

// HandleVMExecute handles command execution tasks
func (w *Worker) HandleVMExecute(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	var payload VMExecutePayload
	if err := queue.UnmarshalPayload(task.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	log.Printf("Executing command on VM %s: %s %v", payload.VMID, payload.Command, payload.Args)

	// Execute command
	cmd := &vmm.Command{
		Cmd:  payload.Command,
		Args: payload.Args,
	}

	execResult, err := w.orchestrator.ExecuteCommand(ctx, payload.VMID, cmd)
	if err != nil {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Store execution in database
	vmUUID, _ := uuid.Parse(payload.VMID)
	exitCode := execResult.ExitCode
	stdout := execResult.Stdout
	stderr := execResult.Stderr

	argsJSON, _ := json.Marshal(payload.Args)
	var argsInterface []interface{}
	json.Unmarshal(argsJSON, &argsInterface)

	execution := &storage.Execution{
		ID:          uuid.New(),
		VMID:        &vmUUID,
		Command:     payload.Command,
		Args:        argsInterface,
		ExitCode:    &exitCode,
		Stdout:      &stdout,
		Stderr:      &stderr,
		StartedAt:   startTime,
		CompletedAt: timePtr(time.Now()),
		DurationMS:  intPtr(int(time.Since(startTime).Milliseconds())),
		Metadata:    make(map[string]interface{}),
	}

	if err := w.store.Executions().Create(ctx, execution); err != nil {
		log.Printf("Warning: Failed to store execution: %v", err)
	}

	success := execResult.ExitCode == 0
	if success {
		log.Printf("✓ Command executed successfully on VM %s", payload.VMID)
	} else {
		log.Printf("✗ Command failed on VM %s (exit code: %d)", payload.VMID, execResult.ExitCode)
	}

	result := map[string]interface{}{
		"vm_id":     payload.VMID,
		"exit_code": execResult.ExitCode,
		"stdout":    execResult.Stdout,
		"stderr":    execResult.Stderr,
	}

	if !success {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     fmt.Sprintf("command failed with exit code %d", execResult.ExitCode),
			Result:    result,
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   true,
		Result:    result,
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

// HandleVMDelete handles VM deletion tasks
func (w *Worker) HandleVMDelete(ctx context.Context, task *queue.Task) (*queue.TaskResult, error) {
	startTime := time.Now()

	vmID, ok := task.Payload["vm_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid payload: missing vm_id")
	}

	log.Printf("Deleting VM: %s", vmID)

	// Delete VM using orchestrator
	if err := w.orchestrator.DeleteVM(ctx, vmID); err != nil {
		return &queue.TaskResult{
			TaskID:    task.ID,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(startTime),
			StartedAt: startTime,
		}, nil
	}

	// Delete from database
	vmUUID, _ := uuid.Parse(vmID)
	if err := w.store.VMs().Delete(ctx, vmUUID); err != nil {
		log.Printf("Warning: Failed to delete VM from database: %v", err)
	}

	log.Printf("✓ VM deleted: %s", vmID)

	return &queue.TaskResult{
		TaskID:    task.ID,
		Success:   true,
		Result:    map[string]interface{}{"vm_id": vmID},
		Duration:  time.Since(startTime),
		StartedAt: startTime,
	}, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}
