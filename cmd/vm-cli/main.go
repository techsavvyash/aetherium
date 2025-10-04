package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/docker"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

var (
	currentOrch vmm.VMOrchestrator
	orchType    string
	ctx         = context.Background()
)

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   Aetherium VM Command CLI             ║")
	fmt.Println("║   Interactive MicroVM Manager          ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "init":
		handleInit()
	case "create":
		handleCreate()
	case "start":
		handleStart()
	case "exec":
		handleExec()
	case "status":
		handleStatus()
	case "list":
		handleList()
	case "stop":
		handleStop()
	case "delete":
		handleDelete()
	case "shell":
		handleShell()
	case "demo":
		handleDemo()
	case "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleInit() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli init <docker|firecracker>")
		os.Exit(1)
	}

	orchType = os.Args[2]
	var err error

	fmt.Printf("Initializing %s orchestrator...\n", orchType)

	switch orchType {
	case "docker":
		config := map[string]interface{}{
			"network": "bridge",
			"image":   "ubuntu:22.04",
		}
		currentOrch, err = docker.NewDockerOrchestrator(config)
	case "firecracker":
		config := map[string]interface{}{
			"kernel_path":       "/var/firecracker/vmlinux",
			"rootfs_template":   "/var/firecracker/rootfs.ext4",
			"socket_dir":        "/tmp",
			"default_vcpu":      2,
			"default_memory_mb": 512,
		}
		currentOrch, err = firecracker.NewFirecrackerOrchestrator(config)
	default:
		fmt.Fprintf(os.Stderr, "Unknown orchestrator type: %s\n", orchType)
		fmt.Println("Supported: docker, firecracker")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize orchestrator: %v\n", err)
		os.Exit(1)
	}

	// Health check
	if err := currentOrch.Health(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Orchestrator health check failed: %v\n", err)
		if orchType == "docker" {
			fmt.Println("\nTroubleshooting:")
			fmt.Println("  • Ensure Docker is running: sudo systemctl start docker")
			fmt.Println("  • Check permissions: sudo usermod -aG docker $USER")
		}
		os.Exit(1)
	}

	fmt.Printf("✓ %s orchestrator initialized successfully\n", orchType)
}

func handleCreate() {
	ensureInitialized()

	vmID := "my-vm"
	if len(os.Args) > 2 {
		vmID = os.Args[2]
	}

	fmt.Printf("Creating VM: %s\n", vmID)

	vmConfig := &types.VMConfig{
		ID:       vmID,
		VCPUCount: 2,
		MemoryMB: 512,
	}

	if orchType == "firecracker" {
		vmConfig.KernelPath = "/var/firecracker/vmlinux"
		vmConfig.RootFSPath = "/var/firecracker/rootfs.ext4"
		vmConfig.SocketPath = fmt.Sprintf("/tmp/firecracker-%s.sock", vmID)
	}

	vm, err := currentOrch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM created: %s\n", vm.ID)
	fmt.Printf("  Status: %s\n", vm.Status)
	if orchType == "docker" {
		fmt.Printf("  Container: %s\n", vm.ID)
	}
}

func handleStart() {
	ensureInitialized()

	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli start <vm-id>")
		os.Exit(1)
	}

	vmID := os.Args[2]
	fmt.Printf("Starting VM: %s\n", vmID)

	if err := currentOrch.StartVM(ctx, vmID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s started\n", vmID)

	// Give it a moment to fully start
	time.Sleep(500 * time.Millisecond)
}

func handleExec() {
	ensureInitialized()

	if len(os.Args) < 4 {
		fmt.Println("Usage: vm-cli exec <vm-id> <command> [args...]")
		fmt.Println("\nExamples:")
		fmt.Println("  vm-cli exec my-vm echo \"Hello World\"")
		fmt.Println("  vm-cli exec my-vm ls -la /")
		fmt.Println("  vm-cli exec my-vm pwd")
		os.Exit(1)
	}

	vmID := os.Args[2]
	cmdName := os.Args[3]
	cmdArgs := os.Args[4:]

	fmt.Printf("Executing in %s: %s %s\n", vmID, cmdName, strings.Join(cmdArgs, " "))

	result, err := currentOrch.ExecuteCommand(ctx, vmID, &vmm.Command{
		Cmd:  cmdName,
		Args: cmdArgs,
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n─── Output ───")
	if result.Stdout != "" {
		fmt.Print(result.Stdout)
		if !strings.HasSuffix(result.Stdout, "\n") {
			fmt.Println()
		}
	}

	if result.Stderr != "" {
		fmt.Println("\n─── Errors ───")
		fmt.Print(result.Stderr)
		if !strings.HasSuffix(result.Stderr, "\n") {
			fmt.Println()
		}
	}

	fmt.Printf("\n─── Exit Code: %d ───\n", result.ExitCode)

	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
}

func handleStatus() {
	ensureInitialized()

	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli status <vm-id>")
		os.Exit(1)
	}

	vmID := os.Args[2]

	vm, err := currentOrch.GetVMStatus(ctx, vmID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get status: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Printf("║ VM Status: %-27s ║\n", vmID)
	fmt.Println("╠════════════════════════════════════════╣")
	fmt.Printf("║ Status:     %-26s ║\n", vm.Status)
	fmt.Printf("║ vCPUs:      %-26d ║\n", vm.Config.VCPUCount)
	fmt.Printf("║ Memory:     %-23d MB ║\n", vm.Config.MemoryMB)
	if vm.CreatedAt.Unix() > 0 {
		fmt.Printf("║ Created:    %-26s ║\n", vm.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	if vm.StartedAt != nil {
		fmt.Printf("║ Started:    %-26s ║\n", vm.StartedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("╚════════════════════════════════════════╝")
}

func handleList() {
	ensureInitialized()

	vms, err := currentOrch.ListVMs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list VMs: %v\n", err)
		os.Exit(1)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found")
		return
	}

	fmt.Println("╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         VM List                                ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Printf("║ %-20s %-15s %-10s %-10s ║\n", "VM ID", "STATUS", "vCPUs", "Memory(MB)")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")

	for _, vm := range vms {
		fmt.Printf("║ %-20s %-15s %-10d %-10d ║\n",
			truncate(vm.ID, 20), vm.Status, vm.Config.VCPUCount, vm.Config.MemoryMB)
	}

	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("Total: %d VM(s)\n", len(vms))
}

func handleStop() {
	ensureInitialized()

	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli stop <vm-id> [--force]")
		os.Exit(1)
	}

	vmID := os.Args[2]
	force := len(os.Args) > 3 && os.Args[3] == "--force"

	fmt.Printf("Stopping VM: %s", vmID)
	if force {
		fmt.Print(" (force)")
	}
	fmt.Println()

	if err := currentOrch.StopVM(ctx, vmID, force); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s stopped\n", vmID)
}

func handleDelete() {
	ensureInitialized()

	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli delete <vm-id>")
		os.Exit(1)
	}

	vmID := os.Args[2]
	fmt.Printf("Deleting VM: %s\n", vmID)

	if err := currentOrch.DeleteVM(ctx, vmID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to delete VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ VM %s deleted\n", vmID)
}

func handleShell() {
	ensureInitialized()

	if len(os.Args) < 3 {
		fmt.Println("Usage: vm-cli shell <vm-id>")
		fmt.Println("\nStarts an interactive shell session in the VM")
		os.Exit(1)
	}

	vmID := os.Args[2]
	fmt.Printf("Starting shell in %s...\n", vmID)
	fmt.Println("Type 'exit' to quit")
	fmt.Println()

	result, err := currentOrch.ExecuteCommand(ctx, vmID, &vmm.Command{
		Cmd:  "bash",
		Args: []string{},
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start shell: %v\n", err)
		fmt.Println("\nNote: Interactive shells require TTY support.")
		fmt.Println("Use 'vm-cli exec' for individual commands instead.")
		os.Exit(1)
	}

	if result.Stdout != "" {
		fmt.Print(result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprintf(os.Stderr, "%s", result.Stderr)
	}
}

func handleDemo() {
	fmt.Println("Running interactive demo...")
	fmt.Println()

	// Step 1: Initialize
	fmt.Println("Step 1: Initializing Docker orchestrator")
	fmt.Println("Command: vm-cli init docker")
	fmt.Println()
	orchType = "docker"
	config := map[string]interface{}{
		"network": "bridge",
		"image":   "ubuntu:22.04",
	}
	var err error
	currentOrch, err = docker.NewDockerOrchestrator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Orchestrator initialized")
	pause()

	// Step 2: Create VM
	fmt.Println("\nStep 2: Creating VM 'demo-vm'")
	fmt.Println("Command: vm-cli create demo-vm")
	fmt.Println()
	vmConfig := &types.VMConfig{
		ID:       "demo-vm",
		VCPUCount: 2,
		MemoryMB: 512,
	}
	vm, err := currentOrch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ VM created: %s (Status: %s)\n", vm.ID, vm.Status)
	pause()

	// Cleanup on exit
	defer func() {
		fmt.Println("\nCleaning up...")
		currentOrch.DeleteVM(ctx, "demo-vm")
		fmt.Println("✓ Demo VM deleted")
	}()

	// Step 3: Start VM
	fmt.Println("\nStep 3: Starting VM")
	fmt.Println("Command: vm-cli start demo-vm")
	fmt.Println()
	if err := currentOrch.StartVM(ctx, "demo-vm"); err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		return
	}
	fmt.Println("✓ VM started")
	time.Sleep(1 * time.Second)
	pause()

	// Step 4: Execute commands
	fmt.Println("\nStep 4: Executing commands")
	commands := []struct {
		name string
		cmd  string
		args []string
	}{
		{"Echo", "echo", []string{"Hello from VM!"}},
		{"List files", "ls", []string{"-la", "/home"}},
		{"Show user", "whoami", []string{}},
		{"Current directory", "pwd", []string{}},
	}

	for i, c := range commands {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(commands), c.name)
		fmt.Printf("Command: vm-cli exec demo-vm %s %s\n", c.cmd, strings.Join(c.args, " "))
		fmt.Println()

		result, err := currentOrch.ExecuteCommand(ctx, "demo-vm", &vmm.Command{
			Cmd:  c.cmd,
			Args: c.args,
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
			continue
		}

		fmt.Println("Output:")
		fmt.Print(result.Stdout)
		if !strings.HasSuffix(result.Stdout, "\n") {
			fmt.Println()
		}
		if i < len(commands)-1 {
			pause()
		}
	}

	// Step 5: Check status
	fmt.Println("\nStep 5: Checking VM status")
	fmt.Println("Command: vm-cli status demo-vm")
	fmt.Println()
	status, _ := currentOrch.GetVMStatus(ctx, "demo-vm")
	fmt.Printf("Status: %s\n", status.Status)
	pause()

	// Step 6: Stop VM
	fmt.Println("\nStep 6: Stopping VM")
	fmt.Println("Command: vm-cli stop demo-vm")
	fmt.Println()
	currentOrch.StopVM(ctx, "demo-vm", false)
	fmt.Println("✓ VM stopped")

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Demo complete!")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("\nYou can now use these commands:")
	fmt.Println("  vm-cli init docker")
	fmt.Println("  vm-cli create my-vm")
	fmt.Println("  vm-cli start my-vm")
	fmt.Println("  vm-cli exec my-vm echo 'Hello!'")
	fmt.Println("  vm-cli stop my-vm")
	fmt.Println("  vm-cli delete my-vm")
}

func ensureInitialized() {
	if currentOrch == nil {
		fmt.Fprintln(os.Stderr, "Error: Orchestrator not initialized")
		fmt.Fprintln(os.Stderr, "Please run: vm-cli init <docker|firecracker>")
		os.Exit(1)
	}
}

func pause() {
	fmt.Print("\n[Press Enter to continue...]")
	fmt.Scanln()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func printUsage() {
	fmt.Println("Usage: vm-cli <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  init <type>              Initialize orchestrator (docker|firecracker)")
	fmt.Println("  create [vm-id]           Create a new VM")
	fmt.Println("  start <vm-id>            Start a VM")
	fmt.Println("  exec <vm-id> <cmd> [...]Execute command in VM")
	fmt.Println("  status <vm-id>           Get VM status")
	fmt.Println("  list                     List all VMs")
	fmt.Println("  stop <vm-id> [--force]   Stop a VM")
	fmt.Println("  delete <vm-id>           Delete a VM")
	fmt.Println("  shell <vm-id>            Start interactive shell")
	fmt.Println("  demo                     Run interactive demo")
	fmt.Println("  help                     Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Quick start")
	fmt.Println("  vm-cli demo")
	fmt.Println()
	fmt.Println("  # Manual workflow")
	fmt.Println("  vm-cli init docker")
	fmt.Println("  vm-cli create my-vm")
	fmt.Println("  vm-cli start my-vm")
	fmt.Println("  vm-cli exec my-vm echo \"Hello World\"")
	fmt.Println("  vm-cli exec my-vm ls -la /")
	fmt.Println("  vm-cli exec my-vm pwd")
	fmt.Println("  vm-cli status my-vm")
	fmt.Println("  vm-cli list")
	fmt.Println("  vm-cli stop my-vm")
	fmt.Println("  vm-cli delete my-vm")
	fmt.Println()
	fmt.Println("  # With Firecracker")
	fmt.Println("  vm-cli init firecracker")
	fmt.Println("  vm-cli create fc-vm")
	fmt.Println("  vm-cli start fc-vm")
	fmt.Println("  vm-cli exec fc-vm whoami")
}
