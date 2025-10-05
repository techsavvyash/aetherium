package integration

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/pkg/worker"
	"github.com/google/uuid"
)

// TestThreeVMClone tests distributed task execution by creating 3 VMs
// and cloning 3 different GitHub repositories in parallel
func TestThreeVMClone(t *testing.T) {
	ctx := context.Background()

	// Repositories to clone
	repos := []struct {
		name string
		url  string
	}{
		{"vync", "https://github.com/techsavvyash/vync"},
		{"veil", "https://github.com/try-veil/veil"},
		{"web", "https://github.com/try-veil/web"},
	}

	log.Println("=== Aetherium Integration Test: 3 VM Clone ===")

	// Initialize PostgreSQL store
	log.Println("Connecting to PostgreSQL...")
	store, err := postgres.NewStore(postgres.Config{
		Host:         "localhost",
		Port:         5432,
		User:         "aetherium",
		Password:     "aetherium",
		Database:     "aetherium",
		SSLMode:      "disable",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Initialize Redis queue
	log.Println("Connecting to Redis...")
	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: "localhost:6379",
		Queues: map[string]int{
			"default": 10,
		},
	})
	if err != nil {
		t.Fatalf("Failed to initialize queue: %v", err)
	}

	// Initialize Firecracker orchestrator
	log.Println("Initializing Firecracker orchestrator...")
	orchestrator, err := firecracker.NewFirecrackerOrchestrator(map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 256,
	})
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Create and start worker
	log.Println("Starting worker...")
	w := worker.New(store, orchestrator)
	if err := w.RegisterHandlers(queue); err != nil {
		t.Fatalf("Failed to register handlers: %v", err)
	}

	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()

	if err := queue.Start(workerCtx); err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		queue.Stop(stopCtx)
	}()

	log.Println("Worker started successfully")

	// Create task service
	taskService := service.NewTaskService(queue, store)

	// Track created VMs for cleanup
	var vmIDs []string
	defer func() {
		log.Println("\n=== Cleanup: Deleting VMs ===")
		for _, vmID := range vmIDs {
			taskID, err := taskService.DeleteVMTask(ctx, vmID)
			if err != nil {
				log.Printf("Failed to submit delete task for VM %s: %v", vmID, err)
			} else {
				log.Printf("Submitted delete task %s for VM %s", taskID, vmID)
			}
		}
		// Wait for cleanup
		time.Sleep(10 * time.Second)
	}()

	// Step 1: Create 3 VMs in parallel
	log.Println("\n=== Step 1: Creating 3 VMs ===")
	vmCreateTasks := make([]uuid.UUID, 3)

	for i, repo := range repos {
		vmName := fmt.Sprintf("clone-%s", repo.name)
		taskID, err := taskService.CreateVMTask(ctx, vmName, 1, 256)
		if err != nil {
			t.Fatalf("Failed to create VM task for %s: %v", vmName, err)
		}
		vmCreateTasks[i] = taskID
		log.Printf("Created VM task %s for %s", taskID, vmName)
	}

	// Wait for VM creation to complete
	log.Println("Waiting for VMs to be created...")
	time.Sleep(30 * time.Second) // Give time for VMs to start and agent to initialize

	// Get VM IDs from database
	for _, repo := range repos {
		vmName := fmt.Sprintf("clone-%s", repo.name)
		vm, err := taskService.GetVMByName(ctx, vmName)
		if err != nil {
			t.Fatalf("Failed to get VM %s: %v", vmName, err)
		}
		vmIDs = append(vmIDs, vm.ID.String())
		log.Printf("✓ VM created: %s (ID: %s)", vmName, vm.ID)
	}

	// Step 2: Install git in all VMs (in case not present in rootfs)
	log.Println("\n=== Step 2: Installing git in VMs ===")
	for i, vmID := range vmIDs {
		// Update package list
		taskID, err := taskService.ExecuteCommandTask(ctx, vmID, "apt-get", []string{"update"})
		if err != nil {
			log.Printf("Warning: Failed to create apt-get update task for VM %s: %v", vmID, err)
		} else {
			log.Printf("Submitted apt-get update task %s for VM %d", taskID, i+1)
		}
	}

	// Wait for updates
	time.Sleep(30 * time.Second)

	for i, vmID := range vmIDs {
		// Install git
		taskID, err := taskService.ExecuteCommandTask(ctx, vmID, "apt-get", []string{"install", "-y", "git"})
		if err != nil {
			log.Printf("Warning: Failed to create git install task for VM %s: %v", vmID, err)
		} else {
			log.Printf("Submitted git install task %s for VM %d", taskID, i+1)
		}
	}

	// Wait for git installation
	log.Println("Waiting for git installation...")
	time.Sleep(60 * time.Second)

	// Step 3: Clone repositories in parallel
	log.Println("\n=== Step 3: Cloning repositories ===")
	cloneTasks := make([]uuid.UUID, 3)

	for i, repo := range repos {
		vmID := vmIDs[i]
		taskID, err := taskService.ExecuteCommandTask(ctx, vmID, "git", []string{"clone", repo.url})
		if err != nil {
			t.Fatalf("Failed to create clone task for %s: %v", repo.url, err)
		}
		cloneTasks[i] = taskID
		log.Printf("Submitted clone task %s for %s on VM %d", taskID, repo.url, i+1)
	}

	// Wait for cloning to complete
	log.Println("Waiting for repositories to clone...")
	time.Sleep(90 * time.Second) // Give time for cloning

	// Step 4: Verify clones by listing directories
	log.Println("\n=== Step 4: Verifying clones ===")
	for i, repo := range repos {
		vmID := vmIDs[i]
		taskID, err := taskService.ExecuteCommandTask(ctx, vmID, "ls", []string{"-la", repo.name})
		if err != nil {
			t.Fatalf("Failed to create ls task for %s: %v", repo.name, err)
		}
		log.Printf("Submitted verification task %s for %s", taskID, repo.name)
	}

	// Wait for verification
	time.Sleep(10 * time.Second)

	// Step 5: Retrieve execution results
	log.Println("\n=== Step 5: Execution Results ===")
	for i, repo := range repos {
		vmUUID, _ := uuid.Parse(vmIDs[i])
		executions, err := taskService.GetExecutions(ctx, vmUUID)
		if err != nil {
			log.Printf("Warning: Failed to get executions for VM %s: %v", vmIDs[i], err)
			continue
		}

		log.Printf("\nVM %d (%s) - %d executions:", i+1, repo.name, len(executions))
		for _, exec := range executions {
			status := "✓"
			if exec.ExitCode != nil && *exec.ExitCode != 0 {
				status = "✗"
			}
			log.Printf("  %s %s %v (exit: %v)", status, exec.Command, exec.Args, getIntValue(exec.ExitCode))
			if exec.Stderr != nil && *exec.Stderr != "" {
				log.Printf("    stderr: %s", *exec.Stderr)
			}
		}
	}

	// List all VMs
	log.Println("\n=== Final VM State ===")
	vms, err := taskService.ListVMs(ctx)
	if err != nil {
		log.Printf("Warning: Failed to list VMs: %v", err)
	} else {
		for _, vm := range vms {
			log.Printf("VM: %s (ID: %s, Status: %s)", vm.Name, vm.ID, vm.Status)
		}
	}

	log.Println("\n=== Test Completed Successfully ===")
}

func getIntValue(ptr *int) int {
	if ptr == nil {
		return -1
	}
	return *ptr
}
