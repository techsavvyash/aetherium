package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetherium/aetherium/pkg/config"
	"github.com/aetherium/aetherium/pkg/container"
	"github.com/aetherium/aetherium/pkg/container/factories"
	"github.com/aetherium/aetherium/pkg/storage"
	"github.com/aetherium/aetherium/pkg/types"
)

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   DI Container Demo                    ║")
	fmt.Println("║   In-Memory Providers                  ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()

	// Create configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		TaskQueue: config.ProviderConfig{
			Provider: "memory",
			Config: map[string]interface{}{
				"buffer_size": 100,
				"workers":     2,
			},
		},
		Storage: config.ProviderConfig{
			Provider: "memory",
			Config:   map[string]interface{}{},
		},
		Logging: config.ProviderConfig{
			Provider: "stdout",
			Config: map[string]interface{}{
				"colorize": true,
			},
		},
		VMM: config.ProviderConfig{
			Provider: "docker",
			Config: map[string]interface{}{
				"network": "bridge",
				"image":   "ubuntu:22.04",
			},
		},
		EventBus: config.ProviderConfig{
			Provider: "memory",
			Config:   map[string]interface{}{},
		},
		Integrations: config.IntegrationsConfig{
			Enabled: []string{},
			Configs: map[string]map[string]interface{}{},
		},
	}

	// Create container
	fmt.Println("1. Creating DI container...")
	cont := container.New(cfg)

	// Register factories
	fmt.Println("2. Registering component factories...")
	cont.RegisterTaskQueueFactory(factories.NewQueueFactory())
	cont.RegisterStateStoreFactory(factories.NewStorageFactory())
	cont.RegisterLoggerFactory(factories.NewLoggerFactory())
	cont.RegisterVMOrchestratorFactory(factories.NewVMMFactory())
	cont.RegisterEventBusFactory(factories.NewEventBusFactory())
	fmt.Println("   ✓ All factories registered")
	fmt.Println()

	// Initialize all components
	fmt.Println("3. Initializing all components...")
	if err := cont.Initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to initialize: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   ✓ All components initialized")
	fmt.Println()

	// Demonstrate each component
	demonstrateLogger(ctx, cont)
	demonstrateStorage(ctx, cont)
	demonstrateEventBus(ctx, cont)
	demonstrateVMOrchestrator(ctx, cont)
	demonstrateTaskQueue(ctx, cont)

	// Shutdown
	fmt.Println("\n6. Shutting down container...")
	if err := cont.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Shutdown error: %v\n", err)
	} else {
		fmt.Println("   ✓ Clean shutdown complete")
	}

	fmt.Println("\n╔════════════════════════════════════════╗")
	fmt.Println("║   ✓ Demo Complete!                     ║")
	fmt.Println("╚════════════════════════════════════════╝")
}

func demonstrateLogger(ctx context.Context, cont *container.Container) {
	fmt.Println("4. Testing Logger (stdout)...")
	logger := cont.GetLogger()

	logger.Log(ctx, types.LogLevelInfo, "Logger initialized successfully", map[string]interface{}{
		"component": "demo",
		"timestamp": time.Now().Unix(),
	})

	logger.Log(ctx, types.LogLevelDebug, "This is a debug message", nil)
	logger.Log(ctx, types.LogLevelWarn, "This is a warning message", nil)
	logger.Log(ctx, types.LogLevelError, "This is an error message", nil)

	fmt.Println("   ✓ Logger working")
	fmt.Println()
}

func demonstrateStorage(ctx context.Context, cont *container.Container) {
	fmt.Println("5. Testing StateStore (memory)...")
	store := cont.GetStateStore()

	// Create a project
	project := &types.Project{
		ID:          "proj-001",
		Name:        "Demo Project",
		Description: "A test project",
		Status:      types.ProjectStatusActive,
	}

	if err := store.CreateProject(ctx, project); err != nil {
		fmt.Printf("   ✗ Failed to create project: %v\n", err)
		return
	}
	fmt.Println("   ✓ Created project: proj-001")

	// Create a task
	task := &types.Task{
		ID:          "task-001",
		ProjectID:   "proj-001",
		Type:        "code_review",
		Description: "Review pull request",
		Status:      types.TaskStatusPending,
	}

	if err := store.CreateTask(ctx, task); err != nil {
		fmt.Printf("   ✗ Failed to create task: %v\n", err)
		return
	}
	fmt.Println("   ✓ Created task: task-001")

	// Query tasks
	projID := "proj-001"
	tasks, err := store.ListTasks(ctx, &storage.TaskFilters{ProjectID: &projID})
	if err != nil {
		fmt.Printf("   ✗ Failed to list tasks: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Found %d task(s) for project\n", len(tasks))
	fmt.Println()
}

func demonstrateEventBus(ctx context.Context, cont *container.Container) {
	fmt.Println("6. Testing EventBus (memory)...")
	bus := cont.GetEventBus()

	// Subscribe to events
	eventReceived := false
	subID, err := bus.Subscribe(ctx, "demo.test", func(ctx context.Context, event *types.Event) error {
		fmt.Printf("   → Received event: %s (type: %s)\n", event.ID, event.Type)
		eventReceived = true
		return nil
	})

	if err != nil {
		fmt.Printf("   ✗ Failed to subscribe: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Subscribed to 'demo.test' (id: %s)\n", subID)

	// Publish an event
	event := &types.Event{
		ID:        "evt-001",
		Type:      "task.created",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"task_id": "task-001",
		},
	}

	if err := bus.Publish(ctx, "demo.test", event); err != nil {
		fmt.Printf("   ✗ Failed to publish: %v\n", err)
		return
	}
	fmt.Println("   ✓ Published event to 'demo.test'")

	// Give handlers time to process
	time.Sleep(100 * time.Millisecond)

	if eventReceived {
		fmt.Println("   ✓ Event successfully received by subscriber")
	}
	fmt.Println()
}

func demonstrateVMOrchestrator(ctx context.Context, cont *container.Container) {
	fmt.Println("7. Testing VMOrchestrator (docker)...")
	orch := cont.GetVMOrchestrator()

	// Check health
	if err := orch.Health(ctx); err != nil {
		fmt.Printf("   ⚠ Docker not available: %v\n", err)
		fmt.Println("   Skipping VM orchestrator demo")
		fmt.Println()
		return
	}

	// Create a VM
	vm, err := orch.CreateVM(ctx, &types.VMConfig{
		ID:       "demo-vm",
		VCPUCount: 1,
		MemoryMB: 256,
	})

	if err != nil {
		fmt.Printf("   ✗ Failed to create VM: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Created VM: %s (status: %s)\n", vm.ID, vm.Status)

	// Cleanup
	defer func() {
		orch.DeleteVM(ctx, vm.ID)
		fmt.Println("   ✓ Cleaned up VM")
	}()

	// List VMs
	vms, err := orch.ListVMs(ctx)
	if err != nil {
		fmt.Printf("   ✗ Failed to list VMs: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Listed %d VM(s)\n", len(vms))
	fmt.Println()
}

func demonstrateTaskQueue(ctx context.Context, cont *container.Container) {
	fmt.Println("8. Testing TaskQueue (memory)...")
	queue := cont.GetTaskQueue()

	// Register a handler
	taskProcessed := false
	err := queue.RegisterHandler("demo_task", func(ctx context.Context, task *types.Task) error {
		fmt.Printf("   → Processing task: %s\n", task.ID)
		taskProcessed = true
		return nil
	})

	if err != nil {
		fmt.Printf("   ✗ Failed to register handler: %v\n", err)
		return
	}
	fmt.Println("   ✓ Registered handler for 'demo_task'")

	// Start workers
	if err := queue.Start(ctx); err != nil {
		fmt.Printf("   ✗ Failed to start queue: %v\n", err)
		return
	}
	fmt.Println("   ✓ Started task queue workers")

	// Enqueue a task
	task := &types.Task{
		ID:     "task-demo-001",
		Type:   "demo_task",
		Status: types.TaskStatusPending,
	}

	if err := queue.Enqueue(ctx, task); err != nil {
		fmt.Printf("   ✗ Failed to enqueue task: %v\n", err)
		return
	}
	fmt.Println("   ✓ Enqueued task to queue")

	// Give workers time to process
	time.Sleep(500 * time.Millisecond)

	if taskProcessed {
		fmt.Println("   ✓ Task successfully processed by worker")
	}

	// Stop queue
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := queue.Stop(stopCtx); err != nil {
		fmt.Printf("   ⚠ Queue stop warning: %v\n", err)
	} else {
		fmt.Println("   ✓ Stopped task queue")
	}
	fmt.Println()
}
