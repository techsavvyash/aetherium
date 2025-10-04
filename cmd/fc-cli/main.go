package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

func main() {
	fmt.Println("Aetherium Firecracker CLI")
	fmt.Println("=========================\n")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Initialize orchestrator
	config := map[string]interface{}{
		"kernel_path":      "/var/firecracker/vmlinux",
		"rootfs_template":  "/var/firecracker/rootfs.ext4",
		"socket_dir":       "/tmp",
		"default_vcpu":     2,
		"default_memory_mb": 512,
	}

	orch, err := firecracker.NewFirecrackerOrchestrator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create orchestrator: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	switch command {
	case "create":
		handleCreate(ctx, orch)
	case "start":
		handleStart(ctx, orch)
	case "stop":
		handleStop(ctx, orch)
	case "status":
		handleStatus(ctx, orch)
	case "delete":
		handleDelete(ctx, orch)
	case "list":
		handleList(ctx, orch)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleCreate(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	vmID := "test-vm"
	if len(os.Args) > 2 {
		vmID = os.Args[2]
	}

	socketPath := fmt.Sprintf("/tmp/firecracker-%s.sock", vmID)

	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: socketPath,
		VCPUCount:  2,
		MemoryMB:   512,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM created: %s\n", vm.ID)
	fmt.Printf("  Status: %s\n", vm.Status)
	fmt.Printf("  Socket: %s\n", vmConfig.SocketPath)
}

func handleStart(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: fc-cli start <vm-id>\n")
		os.Exit(1)
	}

	vmID := os.Args[2]

	fmt.Printf("Starting VM %s...\n", vmID)
	err := orch.StartVM(ctx, vmID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s started successfully\n", vmID)
}

func handleStop(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: fc-cli stop <vm-id> [--force]\n")
		os.Exit(1)
	}

	vmID := os.Args[2]
	force := len(os.Args) > 3 && os.Args[3] == "--force"

	fmt.Printf("Stopping VM %s (force: %v)...\n", vmID, force)
	err := orch.StopVM(ctx, vmID, force)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s stopped\n", vmID)
}

func handleStatus(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: fc-cli status <vm-id>\n")
		os.Exit(1)
	}

	vmID := os.Args[2]

	vm, err := orch.GetVMStatus(ctx, vmID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get VM status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("VM: %s\n", vm.ID)
	fmt.Printf("Status: %s\n", vm.Status)
	fmt.Printf("vCPUs: %d\n", vm.Config.VCPUCount)
	fmt.Printf("Memory: %d MB\n", vm.Config.MemoryMB)
	if vm.StartedAt != nil {
		fmt.Printf("Started: %s\n", vm.StartedAt.Format(time.RFC3339))
	}
}

func handleDelete(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: fc-cli delete <vm-id>\n")
		os.Exit(1)
	}

	vmID := os.Args[2]

	err := orch.DeleteVM(ctx, vmID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s deleted\n", vmID)
}

func handleList(ctx context.Context, orch *firecracker.FirecrackerOrchestrator) {
	vms, err := orch.ListVMs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list VMs: %v\n", err)
		os.Exit(1)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found")
		return
	}

	fmt.Printf("%-20s %-15s %-10s %-10s\n", "VM ID", "STATUS", "vCPUs", "Memory(MB)")
	fmt.Println(string(make([]byte, 60)))

	for _, vm := range vms {
		fmt.Printf("%-20s %-15s %-10d %-10d\n",
			vm.ID, vm.Status, vm.Config.VCPUCount, vm.Config.MemoryMB)
	}
}

func printUsage() {
	fmt.Println("Usage: fc-cli <command> [arguments]")
	fmt.Println("\nCommands:")
	fmt.Println("  create [vm-id]           Create a new VM")
	fmt.Println("  start <vm-id>            Start a VM")
	fmt.Println("  stop <vm-id> [--force]   Stop a VM")
	fmt.Println("  status <vm-id>           Get VM status")
	fmt.Println("  delete <vm-id>           Delete a VM")
	fmt.Println("  list                     List all VMs")
	fmt.Println("\nExample:")
	fmt.Println("  fc-cli create my-vm")
	fmt.Println("  fc-cli start my-vm")
	fmt.Println("  fc-cli status my-vm")
	fmt.Println("  fc-cli stop my-vm")
	fmt.Println("  fc-cli delete my-vm")
}
