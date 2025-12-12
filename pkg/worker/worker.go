package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/discovery"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/tools"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/google/uuid"
)

// Worker handles task execution
type Worker struct {
	// Core dependencies
	store            storage.Store
	orchestrator     vmm.VMOrchestrator
	toolInstaller    *tools.Installer
	workspaceService *service.WorkspaceService

	// Service discovery
	registry   discovery.ServiceRegistry
	workerInfo *discovery.WorkerInfo

	// Resource tracking
	mu             sync.RWMutex
	runningVMs     map[string]*vmResourceUsage
	tasksProcessed int

	// Heartbeat control
	heartbeatCancel context.CancelFunc
	heartbeatDone   chan struct{}
}

// vmResourceUsage tracks resource usage for a VM
type vmResourceUsage struct {
	VCPUs    int
	MemoryMB int64
}

// Config holds worker configuration
type Config struct {
	ID           string
	Hostname     string
	Address      string
	Zone         string
	Labels       map[string]string
	Capabilities []string

	// Resource capacity
	CPUCores int
	MemoryMB int64
	DiskGB   int64
	MaxVMs   int

	// Service discovery (optional)
	Registry discovery.ServiceRegistry
}

// New creates a new worker
func New(store storage.Store, orchestrator vmm.VMOrchestrator) *Worker {
	return &Worker{
		store:         store,
		orchestrator:  orchestrator,
		toolInstaller: tools.NewInstaller(orchestrator),
		runningVMs:    make(map[string]*vmResourceUsage),
	}
}

// NewWithConfig creates a new worker with configuration and service discovery
func NewWithConfig(store storage.Store, orchestrator vmm.VMOrchestrator, config *Config) (*Worker, error) {
	// Set defaults
	if config.ID == "" {
		config.ID = uuid.New().String()
	}
	if config.Hostname == "" {
		hostname, _ := os.Hostname()
		config.Hostname = hostname
	}
	if config.CPUCores == 0 {
		config.CPUCores = runtime.NumCPU()
	}
	if config.MaxVMs == 0 {
		config.MaxVMs = 100
	}

	worker := &Worker{
		store:         store,
		orchestrator:  orchestrator,
		toolInstaller: tools.NewInstaller(orchestrator),
		registry:      config.Registry,
		runningVMs:    make(map[string]*vmResourceUsage),
		workerInfo: &discovery.WorkerInfo{
			ID:           config.ID,
			Hostname:     config.Hostname,
			Address:      config.Address,
			Zone:         config.Zone,
			Labels:       config.Labels,
			Capabilities: config.Capabilities,
			Status:       discovery.WorkerStatusActive,
			StartedAt:    time.Now(),
			LastSeen:     time.Now(),
			Resources: discovery.WorkerResources{
				CPUCores: config.CPUCores,
				MemoryMB: config.MemoryMB,
				DiskGB:   config.DiskGB,
				MaxVMs:   config.MaxVMs,
			},
		},
	}

	return worker, nil
}

// Register registers the worker with service discovery and database
func (w *Worker) Register(ctx context.Context) error {
	// Register with service discovery if configured
	if w.registry != nil {
		if err := w.registry.Register(ctx, w.workerInfo); err != nil {
			return fmt.Errorf("failed to register with service discovery: %w", err)
		}
		log.Printf("Worker registered with service discovery: %s", w.workerInfo.ID)
	}

	// Register in database if worker info exists
	if w.workerInfo != nil {
		// Convert to storage.Worker
		dbWorker := w.workerInfoToStorage(w.workerInfo)
		if err := w.store.Workers().Create(ctx, dbWorker); err != nil {
			// If already exists, update instead
			if err := w.store.Workers().Update(ctx, dbWorker); err != nil {
				return fmt.Errorf("failed to register worker in database: %w", err)
			}
		}
		log.Printf("Worker registered in database: %s", w.workerInfo.ID)
	}

	return nil
}

// StartHeartbeat starts sending periodic heartbeats
func (w *Worker) StartHeartbeat(interval time.Duration) error {
	if w.registry == nil {
		log.Println("Service discovery not configured, skipping heartbeat")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.heartbeatCancel = cancel
	w.heartbeatDone = make(chan struct{})

	go func() {
		defer close(w.heartbeatDone)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := w.sendHeartbeat(context.Background()); err != nil {
					log.Printf("Heartbeat failed: %v", err)
				}
			}
		}
	}()

	log.Printf("Heartbeat started (interval: %v)", interval)
	return nil
}

// sendHeartbeat sends a heartbeat and updates resource usage
func (w *Worker) sendHeartbeat(ctx context.Context) error {
	// Update last seen time
	w.mu.Lock()
	w.workerInfo.LastSeen = time.Now()

	// Update resource usage
	w.workerInfo.Resources.UsedCPUCores = 0
	w.workerInfo.Resources.UsedMemoryMB = 0
	w.workerInfo.Resources.VMCount = len(w.runningVMs)

	for _, vm := range w.runningVMs {
		w.workerInfo.Resources.UsedCPUCores += vm.VCPUs
		w.workerInfo.Resources.UsedMemoryMB += vm.MemoryMB
	}
	w.mu.Unlock()

	// Send heartbeat to service discovery
	if err := w.registry.Heartbeat(ctx, w.workerInfo.ID); err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	// Update database
	if err := w.store.Workers().UpdateLastSeen(ctx, w.workerInfo.ID); err != nil {
		log.Printf("Warning: Failed to update last_seen in database: %v", err)
	}

	return nil
}

// Deregister removes the worker from service discovery
func (w *Worker) Deregister(ctx context.Context) error {
	// Stop heartbeat
	if w.heartbeatCancel != nil {
		w.heartbeatCancel()
		<-w.heartbeatDone
		log.Println("Heartbeat stopped")
	}

	// Deregister from service discovery
	if w.registry != nil {
		if err := w.registry.Deregister(ctx, w.workerInfo.ID); err != nil {
			return fmt.Errorf("failed to deregister from service discovery: %w", err)
		}
		log.Printf("Worker deregistered from service discovery: %s", w.workerInfo.ID)
	}

	// Update status in database
	if w.workerInfo != nil {
		if err := w.store.Workers().UpdateStatus(ctx, w.workerInfo.ID, string(discovery.WorkerStatusOffline)); err != nil {
			log.Printf("Warning: Failed to update worker status in database: %v", err)
		}
	}

	return nil
}

// GetWorkerInfo returns the worker's information
func (w *Worker) GetWorkerInfo() *discovery.WorkerInfo {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Return a copy to avoid race conditions
	info := *w.workerInfo
	return &info
}

// Helper: convert WorkerInfo to storage.Worker
func (w *Worker) workerInfoToStorage(info *discovery.WorkerInfo) *storage.Worker {
	// Convert capabilities to interface slice
	capabilities := make([]interface{}, len(info.Capabilities))
	for i, cap := range info.Capabilities {
		capabilities[i] = cap
	}

	// Convert labels to interface map
	labels := make(map[string]interface{})
	for k, v := range info.Labels {
		labels[k] = v
	}

	return &storage.Worker{
		ID:           info.ID,
		Hostname:     info.Hostname,
		Address:      info.Address,
		Status:       string(info.Status),
		LastSeen:     info.LastSeen,
		StartedAt:    info.StartedAt,
		Zone:         info.Zone,
		Labels:       labels,
		Capabilities: capabilities,
		CPUCores:     info.Resources.CPUCores,
		MemoryMB:     info.Resources.MemoryMB,
		DiskGB:       info.Resources.DiskGB,
		UsedCPUCores: info.Resources.UsedCPUCores,
		UsedMemoryMB: info.Resources.UsedMemoryMB,
		UsedDiskGB:   info.Resources.UsedDiskGB,
		VMCount:      info.Resources.VMCount,
		MaxVMs:       info.Resources.MaxVMs,
		Metadata:     make(map[string]interface{}),
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

	// Register workspace handlers (separate function)
	if err := w.RegisterWorkspaceHandlers(q); err != nil {
		return fmt.Errorf("failed to register workspace handlers: %w", err)
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
		RootFSPath: "", // Will be set by orchestrator.CreateVM() from template
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

	// Track VM resources
	w.mu.Lock()
	w.runningVMs[vm.ID] = &vmResourceUsage{
		VCPUs:    payload.VCPUs,
		MemoryMB: int64(payload.MemoryMB),
	}
	w.tasksProcessed++
	w.mu.Unlock()

	// Store VM in database
	vmUUID, _ := uuid.Parse(vm.ID)
	kernelPath := vmConfig.KernelPath
	rootfsPath := vmConfig.RootFSPath
	socketPath := vmConfig.SocketPath

	// Prepare worker_id if worker info exists
	var workerID *string
	if w.workerInfo != nil {
		workerID = &w.workerInfo.ID
	}

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
		WorkerID:     workerID,
		CreatedAt:    time.Now(),
		Metadata:     make(map[string]interface{}),
	}

	if err := w.store.VMs().Create(ctx, dbVM); err != nil {
		log.Printf("Warning: Failed to store VM in database: %v", err)
	}

	// Update worker resources in database
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
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

	// Convert args to JSONBArray
	args := make(storage.JSONBArray, len(payload.Args))
	for i, arg := range payload.Args {
		args[i] = arg
	}

	execution := &storage.Execution{
		ID:          uuid.New(),
		VMID:        &vmUUID,
		Command:     payload.Command,
		Args:        args,
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

	// Untrack VM resources
	w.mu.Lock()
	delete(w.runningVMs, vmID)
	w.mu.Unlock()

	// Delete from database
	vmUUID, _ := uuid.Parse(vmID)
	if err := w.store.VMs().Delete(ctx, vmUUID); err != nil {
		log.Printf("Warning: Failed to delete VM from database: %v", err)
	}

	// Update worker resources in database
	if w.workerInfo != nil {
		w.updateWorkerResources(ctx)
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

// updateWorkerResources updates worker resource usage in the database
func (w *Worker) updateWorkerResources(ctx context.Context) {
	w.mu.RLock()
	usedCPU := 0
	usedMemory := int64(0)
	vmCount := len(w.runningVMs)

	for _, vm := range w.runningVMs {
		usedCPU += vm.VCPUs
		usedMemory += vm.MemoryMB
	}
	w.mu.RUnlock()

	resources := map[string]interface{}{
		"used_cpu_cores": usedCPU,
		"used_memory_mb": usedMemory,
		"vm_count":       vmCount,
	}

	if err := w.store.Workers().UpdateResources(ctx, w.workerInfo.ID, resources); err != nil {
		log.Printf("Warning: Failed to update worker resources: %v", err)
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func intPtr(i int) *int {
	return &i
}
