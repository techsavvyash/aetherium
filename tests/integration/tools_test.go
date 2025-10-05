package integration

import (
	"context"
	"testing"
	"time"

	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/aetherium/aetherium/pkg/service"
	"github.com/aetherium/aetherium/pkg/storage/postgres"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/pkg/worker"
)

// TestVMToolInstallation tests the VM tool installation feature
func TestVMToolInstallation(t *testing.T) {
	ctx := context.Background()

	// Setup infrastructure
	store, err := postgres.NewStore(postgres.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "aetherium",
		Password: "aetherium",
		Database: "aetherium",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: "localhost:6379",
		Queues: map[string]int{
			"default": 10,
		},
	})
	if err != nil {
		t.Fatalf("Failed to initialize queue: %v", err)
	}

	orchestrator, err := firecracker.NewFirecrackerOrchestrator(map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 512,
	})
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Start worker
	w := worker.New(store, orchestrator)
	if err := w.RegisterHandlers(queue); err != nil {
		t.Fatalf("Failed to register handlers: %v", err)
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := queue.Start(workerCtx); err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		queue.Stop(stopCtx)
	}()

	taskService := service.NewTaskService(queue, store)

	t.Run("CreateVMWithAdditionalTools", func(t *testing.T) {
		// Create VM with Go and Python
		additionalTools := []string{"go", "python"}
		toolVersions := map[string]string{
			"go":     "1.23.0",
			"python": "3.11",
		}

		taskID, err := taskService.CreateVMTaskWithTools(
			ctx,
			"tools-test-vm",
			1,
			512,
			additionalTools,
			toolVersions,
		)
		if err != nil {
			t.Fatalf("Failed to create VM task: %v", err)
		}

		t.Logf("Created VM task: %s", taskID)

		// Wait for VM creation (including tool installation)
		time.Sleep(5 * time.Minute) // Tools take time to install

		// Get VM
		vm, err := taskService.GetVMByName(ctx, "tools-test-vm")
		if err != nil {
			t.Fatalf("Failed to get VM: %v", err)
		}

		t.Logf("VM created: %s (Status: %s)", vm.ID, vm.Status)

		// Verify tools are installed
		// Check Node.js
		taskID, err = taskService.ExecuteCommandTask(ctx, vm.ID.String(), "node", []string{"--version"})
		if err != nil {
			t.Errorf("Failed to check Node.js: %v", err)
		}

		time.Sleep(5 * time.Second)

		// Check Bun
		taskID, err = taskService.ExecuteCommandTask(ctx, vm.ID.String(), "bash", []string{"-c", "~/.bun/bin/bun --version"})
		if err != nil {
			t.Errorf("Failed to check Bun: %v", err)
		}

		time.Sleep(5 * time.Second)

		// Check Go
		taskID, err = taskService.ExecuteCommandTask(ctx, vm.ID.String(), "/usr/local/go/bin/go", []string{"version"})
		if err != nil {
			t.Errorf("Failed to check Go: %v", err)
		}

		time.Sleep(5 * time.Second)

		// Check Python
		taskID, err = taskService.ExecuteCommandTask(ctx, vm.ID.String(), "python3", []string{"--version"})
		if err != nil {
			t.Errorf("Failed to check Python: %v", err)
		}

		time.Sleep(5 * time.Second)

		// Get executions to verify
		executions, err := taskService.GetExecutions(ctx, vm.ID)
		if err != nil {
			t.Fatalf("Failed to get executions: %v", err)
		}

		t.Logf("Executions:")
		for _, exec := range executions {
			exitCode := -1
			if exec.ExitCode != nil {
				exitCode = *exec.ExitCode
			}
			t.Logf("  - %s %v (exit: %d)", exec.Command, exec.Args, exitCode)
			if exec.Stdout != nil {
				t.Logf("    stdout: %s", *exec.Stdout)
			}
		}

		// Cleanup
		_, err = taskService.DeleteVMTask(ctx, vm.ID.String())
		if err != nil {
			t.Errorf("Failed to delete VM: %v", err)
		}
	})
}
