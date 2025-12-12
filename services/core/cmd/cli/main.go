package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aetherium/aetherium/services/core/pkg/queue/asynq"
	"github.com/aetherium/aetherium/services/core/pkg/service"
	"github.com/aetherium/aetherium/services/core/pkg/storage/postgres"
	"github.com/google/uuid"
)

func main() {
	// Define flags
	taskType := flag.String("type", "", "Task type: vm:create, vm:execute, vm:delete")
	vmName := flag.String("name", "", "VM name (for vm:create)")
	vmID := flag.String("vm-id", "", "VM ID (for vm:execute, vm:delete)")
	cmd := flag.String("cmd", "", "Command to execute (for vm:execute)")
	args := flag.String("args", "", "Command arguments, comma-separated (for vm:execute)")
	vcpus := flag.Int("vcpus", 1, "Number of vCPUs (for vm:create)")
	memory := flag.Int("memory", 256, "Memory in MB (for vm:create)")
	tools := flag.String("tools", "", "Additional tools to install, comma-separated (for vm:create)")
	toolVersions := flag.String("tool-versions", "", "Tool versions as key=value pairs, comma-separated (for vm:create)")

	flag.Parse()

	if *taskType == "" {
		fmt.Println("Error: -type is required")
		flag.Usage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Initialize PostgreSQL store
	store, err := postgres.NewStore(postgres.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "aetherium",
		Password: "aetherium",
		Database: "aetherium",
		SSLMode:  "disable",
	})
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	// Initialize Redis queue
	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: "localhost:6379",
	})
	if err != nil {
		log.Fatalf("Failed to initialize queue: %v", err)
	}

	// Create task service
	taskService := service.NewTaskService(queue, store)

	var taskID uuid.UUID

	switch *taskType {
	case "vm:create":
		if *vmName == "" {
			log.Fatal("Error: -name is required for vm:create")
		}

		// Parse additional tools
		var additionalTools []string
		if *tools != "" {
			additionalTools = strings.Split(*tools, ",")
			for i := range additionalTools {
				additionalTools[i] = strings.TrimSpace(additionalTools[i])
			}
		}

		// Parse tool versions
		versions := make(map[string]string)
		if *toolVersions != "" {
			pairs := strings.Split(*toolVersions, ",")
			for _, pair := range pairs {
				parts := strings.Split(pair, "=")
				if len(parts) == 2 {
					versions[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
			}
		}

		taskID, err = taskService.CreateVMTaskWithTools(ctx, *vmName, *vcpus, *memory, additionalTools, versions)
		if err != nil {
			log.Fatalf("Failed to create VM task: %v", err)
		}
		fmt.Printf("✓ VM creation task submitted: %s\n", taskID)
		fmt.Printf("  Name: %s\n", *vmName)
		fmt.Printf("  vCPUs: %d\n", *vcpus)
		fmt.Printf("  Memory: %dMB\n", *memory)
		if len(additionalTools) > 0 {
			fmt.Printf("  Additional Tools: %v\n", additionalTools)
		}
		if len(versions) > 0 {
			fmt.Printf("  Tool Versions: %v\n", versions)
		}
		fmt.Printf("  Default Tools: git, nodejs, bun, claude-code (installed automatically)\n")

	case "vm:execute":
		if *vmID == "" {
			log.Fatal("Error: -vm-id is required for vm:execute")
		}
		if *cmd == "" {
			log.Fatal("Error: -cmd is required for vm:execute")
		}

		var cmdArgs []string
		if *args != "" {
			cmdArgs = strings.Split(*args, ",")
		}

		taskID, err = taskService.ExecuteCommandTask(ctx, *vmID, *cmd, cmdArgs)
		if err != nil {
			log.Fatalf("Failed to create execute task: %v", err)
		}
		fmt.Printf("✓ Command execution task submitted: %s\n", taskID)
		fmt.Printf("  VM ID: %s\n", *vmID)
		fmt.Printf("  Command: %s %v\n", *cmd, cmdArgs)

	case "vm:delete":
		if *vmID == "" {
			log.Fatal("Error: -vm-id is required for vm:delete")
		}
		taskID, err = taskService.DeleteVMTask(ctx, *vmID)
		if err != nil {
			log.Fatalf("Failed to create delete task: %v", err)
		}
		fmt.Printf("✓ VM deletion task submitted: %s\n", taskID)
		fmt.Printf("  VM ID: %s\n", *vmID)

	default:
		log.Fatalf("Unknown task type: %s", *taskType)
	}
}
