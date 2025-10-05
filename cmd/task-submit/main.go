package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aetherium/aetherium/pkg/config"
	"github.com/aetherium/aetherium/pkg/queue"
	"github.com/aetherium/aetherium/pkg/queue/asynq"
	"github.com/google/uuid"
)

func main() {
	configPath := flag.String("config", "config/example.yaml", "Path to config file")
	taskType := flag.String("type", "vm:create", "Task type (vm:create, vm:execute, vm:delete)")
	vmName := flag.String("name", "", "VM name (for vm:create)")
	vmID := flag.String("vm-id", "", "VM ID (for vm:execute, vm:delete)")
	command := flag.String("cmd", "", "Command to execute (for vm:execute)")
	args := flag.String("args", "", "Command args as JSON array (for vm:execute)")
	flag.Parse()

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create task queue client
	taskQueue, err := asynq.NewQueue(asynq.Config{
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to create queue: %v", err)
	}

	ctx := context.Background()

	switch *taskType {
	case "vm:create":
		if *vmName == "" {
			log.Fatal("VM name is required for vm:create")
		}

		task := &queue.Task{
			ID:   uuid.New(),
			Type: queue.TaskTypeVMCreate,
			Payload: map[string]interface{}{
				"name":      *vmName,
				"vcpus":     1,
				"memory_mb": 256,
			},
		}

		if err := taskQueue.Enqueue(ctx, task, &queue.TaskOptions{
			MaxRetry: 3,
			Timeout:  5 * time.Minute,
			Queue:    "default",
		}); err != nil {
			log.Fatalf("Failed to enqueue task: %v", err)
		}

		fmt.Printf("✓ VM creation task submitted: %s\n", task.ID)
		fmt.Printf("  Name: %s\n", *vmName)
		fmt.Printf("  Task will create a Firecracker VM with 1 vCPU and 256MB RAM\n")

	case "vm:execute":
		if *vmID == "" || *command == "" {
			log.Fatal("VM ID and command are required for vm:execute")
		}

		var cmdArgs []string
		if *args != "" {
			// Simple parsing - just split by spaces for demo
			// In production, use proper JSON parsing
			cmdArgs = []string{*args}
		}

		task := &queue.Task{
			ID:   uuid.New(),
			Type: queue.TaskTypeVMExecute,
			Payload: map[string]interface{}{
				"vm_id":   *vmID,
				"command": *command,
				"args":    cmdArgs,
			},
		}

		if err := taskQueue.Enqueue(ctx, task, &queue.TaskOptions{
			MaxRetry: 2,
			Timeout:  10 * time.Minute,
			Queue:    "default",
		}); err != nil {
			log.Fatalf("Failed to enqueue task: %v", err)
		}

		fmt.Printf("✓ Command execution task submitted: %s\n", task.ID)
		fmt.Printf("  VM ID: %s\n", *vmID)
		fmt.Printf("  Command: %s %v\n", *command, cmdArgs)

	case "vm:delete":
		if *vmID == "" {
			log.Fatal("VM ID is required for vm:delete")
		}

		task := &queue.Task{
			ID:   uuid.New(),
			Type: queue.TaskTypeVMDelete,
			Payload: map[string]interface{}{
				"vm_id": *vmID,
			},
		}

		if err := taskQueue.Enqueue(ctx, task, &queue.TaskOptions{
			MaxRetry: 2,
			Timeout:  2 * time.Minute,
			Queue:    "default",
		}); err != nil {
			log.Fatalf("Failed to enqueue task: %v", err)
		}

		fmt.Printf("✓ VM deletion task submitted: %s\n", task.ID)
		fmt.Printf("  VM ID: %s\n", *vmID)

	default:
		log.Fatalf("Unknown task type: %s", *taskType)
	}

	fmt.Println("\nTask submitted to queue. Worker will process it shortly.")
}
