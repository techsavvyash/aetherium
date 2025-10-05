package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aetherium/aetherium/pkg/config"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/google/uuid"
)

func main() {
	configPath := flag.String("config", "config/example.yaml", "Path to config file")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create storage
	store, err := postgres.NewStore(postgres.Config{
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		User:         cfg.Database.User,
		Password:     cfg.Database.Password,
		Database:     cfg.Database.Database,
		SSLMode:      cfg.Database.SSLMode,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create task queue
	taskQueue, err := asynq.NewQueue(asynq.Config{
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
		Concurrency:   cfg.Queue.Concurrency,
		Queues:        cfg.Queue.Queues,
	})
	if err != nil {
		log.Fatalf("Failed to create queue: %v", err)
	}

	// Create VMM orchestrator
	orchestrator, err := firecracker.NewFirecrackerOrchestrator(map[string]interface{}{
		"kernel_path":       cfg.VMM.Firecracker.KernelPath,
		"rootfs_template":   cfg.VMM.Firecracker.RootFSTemplate,
		"socket_dir":        cfg.VMM.Firecracker.SocketDir,
		"default_vcpu":      cfg.VMM.Firecracker.DefaultVCPU,
		"default_memory_mb": cfg.VMM.Firecracker.DefaultMemoryMB,
	})
	if err != nil {
		log.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Create worker
	worker := NewWorker(store, orchestrator)

	// Register task handlers
	if err := taskQueue.RegisterHandler(queue.TaskTypeVMCreate, worker.HandleVMCreate); err != nil {
		log.Fatalf("Failed to register VM create handler: %v", err)
	}

	if err := taskQueue.RegisterHandler(queue.TaskTypeVMExecute, worker.HandleVMExecute); err != nil {
		log.Fatalf("Failed to register VM execute handler: %v", err)
	}

	if err := taskQueue.RegisterHandler(queue.TaskTypeVMDelete, worker.HandleVMDelete); err != nil {
		log.Fatalf("Failed to register VM delete handler: %v", err)
	}

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Worker starting...")
		if err := taskQueue.Start(ctx); err != nil {
			log.Printf("Worker error: %v", err)
		}
	}()

	// Wait for interrupt
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	log.Println("Shutting down...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := taskQueue.Stop(shutdownCtx); err != nil {
		log.Printf("Error stopping queue: %v", err)
	}

	log.Println("Worker stopped")
}

// Worker handles task execution
type Worker struct {
	store        storage.Store
	orchestrator vmm.VMOrchestrator
}

func NewWorker(store storage.Store, orchestrator vmm.VMOrchestrator) *Worker {
	return &Worker{
		store:        store,
		orchestrator: orchestrator,
	}
}

// VMCreatePayload represents VM creation task payload
type VMCreatePayload struct {
	Name     string `json:"name"`
	VCPUs    int    `json:"vcpus"`
	MemoryMB int    `json:"memory_mb"`
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

	result := map[string]interface{}{
		"vm_id": vm.ID,
		"name":  payload.Name,
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

	result := map[string]interface{}{
		"vm_id":     payload.VMID,
		"exit_code": execResult.ExitCode,
		"stdout":    execResult.Stdout,
		"stderr":    execResult.Stderr,
	}

	success := execResult.ExitCode == 0
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
