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
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   Firecracker Command Execution Demo  ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()

	// Check prerequisites
	fmt.Println("1. Checking prerequisites...")
	if err := checkPrerequisites(); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ %v\n", err)
		fmt.Println("\nPlease run:")
		fmt.Println("  sudo ./scripts/install-firecracker.sh")
		fmt.Println("  sudo ./scripts/setup-fc-agent.sh")
		os.Exit(1)
	}
	fmt.Println("   ✓ All prerequisites met")
	fmt.Println()

	// Create orchestrator
	fmt.Println("2. Creating Firecracker orchestrator...")
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := firecracker.NewFirecrackerOrchestrator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to create orchestrator: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("   ✓ Orchestrator created")
	fmt.Println()

	// Create VM
	fmt.Println("3. Creating microVM...")
	vmConfig := &types.VMConfig{
		ID:         "exec-demo-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-exec-demo-vm.sock",
		VCPUCount:  1,
		MemoryMB:   256,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to create VM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ VM created: %s\n", vm.ID)
	fmt.Println()

	// Cleanup on exit
	defer func() {
		fmt.Println("\n7. Cleaning up...")
		if err := orch.StopVM(ctx, vm.ID, true); err != nil {
			fmt.Printf("   ⚠ Stop warning: %v\n", err)
		}
		if err := orch.DeleteVM(ctx, vm.ID); err != nil {
			fmt.Printf("   ⚠ Delete warning: %v\n", err)
		} else {
			fmt.Println("   ✓ VM cleaned up")
		}
	}()

	// Start VM
	fmt.Println("4. Starting microVM...")
	if err := orch.StartVM(ctx, vm.ID); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to start VM: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("  - Ensure you're in 'kvm' group: groups | grep kvm")
		fmt.Println("  - If not, log out and back in, or run: newgrp kvm")
		fmt.Println("  - Check /dev/kvm: ls -l /dev/kvm")
		os.Exit(1)
	}
	fmt.Println("   ✓ VM started successfully!")
	fmt.Println()

	// Wait for VM to boot and agent to start
	fmt.Println("5. Waiting for VM to boot and agent to start...")
	fmt.Print("   ")
	for i := 0; i < 10; i++ {
		fmt.Print("▓")
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println(" ✓")
	fmt.Println()

	// Execute commands
	fmt.Println("6. Executing commands in VM...")
	fmt.Println()

	commands := []struct {
		name string
		cmd  *vmm.Command
	}{
		{
			name: "Echo test",
			cmd: &vmm.Command{
				Cmd:  "echo",
				Args: []string{"Hello from Firecracker VM!"},
			},
		},
		{
			name: "Show kernel version",
			cmd: &vmm.Command{
				Cmd: "uname",
				Args: []string{"-a"},
			},
		},
		{
			name: "List processes",
			cmd: &vmm.Command{
				Cmd: "ps",
				Args: []string{"aux"},
			},
		},
		{
			name: "Show memory",
			cmd: &vmm.Command{
				Cmd: "free",
				Args: []string{"-h"},
			},
		},
		{
			name: "Current directory",
			cmd: &vmm.Command{
				Cmd: "pwd",
			},
		},
		{
			name: "Environment variables",
			cmd: &vmm.Command{
				Cmd: "env",
			},
		},
	}

	for i, test := range commands {
		fmt.Printf("   [%d/%d] %s\n", i+1, len(commands), test.name)
		fmt.Printf("   Command: %s %v\n", test.cmd.Cmd, test.cmd.Args)

		result, err := orch.ExecuteCommand(ctx, vm.ID, test.cmd)
		if err != nil {
			fmt.Printf("   ✗ Error: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("   Exit Code: %d\n", result.ExitCode)
		if result.Stdout != "" {
			fmt.Println("   ┌─ Stdout ─────────────────────────────")
			fmt.Printf("   │ %s", result.Stdout)
			if result.Stdout[len(result.Stdout)-1] != '\n' {
				fmt.Println()
			}
			fmt.Println("   └──────────────────────────────────────")
		}
		if result.Stderr != "" {
			fmt.Println("   ┌─ Stderr ─────────────────────────────")
			fmt.Printf("   │ %s", result.Stderr)
			if result.Stderr[len(result.Stderr)-1] != '\n' {
				fmt.Println()
			}
			fmt.Println("   └──────────────────────────────────────")
		}
		fmt.Println()

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   ✓ Demo Complete!                     ║")
	fmt.Println("╚════════════════════════════════════════╝")
}

func checkPrerequisites() error {
	// Check Firecracker binary
	if _, err := os.Stat("/usr/local/bin/firecracker"); os.IsNotExist(err) {
		if _, err := os.Stat("/usr/bin/firecracker"); os.IsNotExist(err) {
			return fmt.Errorf("firecracker binary not found")
		}
	}

	// Check kernel
	if _, err := os.Stat("/var/firecracker/vmlinux"); os.IsNotExist(err) {
		return fmt.Errorf("kernel not found at /var/firecracker/vmlinux")
	}

	// Check rootfs
	if _, err := os.Stat("/var/firecracker/rootfs.ext4"); os.IsNotExist(err) {
		return fmt.Errorf("rootfs not found at /var/firecracker/rootfs.ext4")
	}

	// Check /dev/kvm
	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		return fmt.Errorf("/dev/kvm not found - KVM not available")
	}

	// Try to open /dev/kvm
	f, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("/dev/kvm not accessible - check permissions (add user to kvm group)")
	}
	f.Close()

	// Check if agent exists
	if _, err := os.Stat("./bin/fc-agent"); os.IsNotExist(err) {
		return fmt.Errorf("fc-agent binary not found - run: make build")
	}

	return nil
}
