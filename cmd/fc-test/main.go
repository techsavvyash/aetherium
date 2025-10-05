package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

func main() {
	fmt.Println("=== Firecracker VM Test ===\n")

	ctx := context.Background()

	// Clean up any existing sockets first
	socketPath := "/tmp/firecracker-test-vm.sock"
	os.Remove(socketPath)
	os.Remove(socketPath + ".vsock")

	// Create orchestrator
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 256,
	}

	orch, err := firecracker.NewFirecrackerOrchestrator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create orchestrator: %v\n", err)
		os.Exit(1)
	}

	// Create VM
	vmConfig := &types.VMConfig{
		ID:         "test-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-test-vm.sock",
		VCPUCount:  1,
		MemoryMB:   256,
	}

	fmt.Println("1. Creating VM...")
	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ VM created: %s\n\n", vm.ID)

	// Cleanup
	defer func() {
		fmt.Println("\nCleaning up...")
		orch.StopVM(ctx, vm.ID, true)
		orch.DeleteVM(ctx, vm.ID)
	}()

	// Start VM
	fmt.Println("2. Starting VM...")
	if err := orch.StartVM(ctx, vm.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   ✓ VM started\n")

	// Wait for boot and agent startup
	fmt.Println("3. Waiting for VM to boot and agent to start (20s)...")
	fmt.Printf("   Logs will be available at: %s.log\n", socketPath)
	time.Sleep(20 * time.Second)
	fmt.Println("   ✓ Boot complete\n")

	// Try simple command
	fmt.Println("4. Testing command execution...")
	cmd := &vmm.Command{
		Cmd:  "echo",
		Args: []string{"Hello from Firecracker!"},
	}

	result, err := orch.ExecuteCommand(ctx, vm.ID, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Command failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("   Exit Code: %d\n", result.ExitCode)
	if result.Stdout != "" {
		fmt.Printf("   Stdout: %s", result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Printf("   Stderr: %s", result.Stderr)
	}

	if result.ExitCode == 0 {
		fmt.Println("\n✓ SUCCESS! Command execution works!")
	} else {
		fmt.Println("\n✗ Command execution failed")
		fmt.Printf("\nTroubleshooting:\n")
		fmt.Printf("- Check VM logs: cat %s.log\n", socketPath)
		fmt.Printf("- Check vsock: ./scripts/diagnose-vsock.sh\n")
		fmt.Printf("- Verify agent in rootfs: sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt && ls -l /mnt/usr/local/bin/fc-agent && sudo umount /mnt\n")
		os.Exit(1)
	}
}
