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
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   Firecracker VM Lifecycle Demo       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	ctx := context.Background()

	// Check prerequisites
	fmt.Println("1. Checking prerequisites...")
	if err := checkPrerequisites(); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ %v\n", err)
		fmt.Println("\nPlease run: ./scripts/install-firecracker.sh")
		os.Exit(1)
	}
	fmt.Println("   ✓ Firecracker binary found")
	fmt.Println("   ✓ Kernel image found")
	fmt.Println("   ✓ Rootfs image found")
	fmt.Println("   ✓ /dev/kvm accessible")
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
	fmt.Println("3. Creating Firecracker microVM...")
	vmConfig := &types.VMConfig{
		ID:         "demo-fc-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-demo-fc-vm.sock",
		VCPUCount:  1,
		MemoryMB:   256,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to create VM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("   ✓ VM created: %s\n", vm.ID)
	fmt.Printf("   Status: %s\n", vm.Status)
	fmt.Printf("   vCPUs: %d\n", vm.Config.VCPUCount)
	fmt.Printf("   Memory: %d MB\n", vm.Config.MemoryMB)
	fmt.Println()

	// Cleanup function
	defer func() {
		fmt.Println("\n6. Cleaning up...")
		if err := orch.DeleteVM(ctx, vm.ID); err != nil {
			fmt.Printf("   ⚠ Cleanup warning: %v\n", err)
		} else {
			fmt.Println("   ✓ VM deleted")
		}
	}()

	// Start VM
	fmt.Println("4. Starting microVM...")
	fmt.Println("   This will:")
	fmt.Println("   - Spawn Firecracker process")
	fmt.Println("   - Wait for API socket")
	fmt.Println("   - Configure VM via HTTP API")
	fmt.Println("   - Start the VM instance")
	fmt.Println()

	if err := orch.StartVM(ctx, vm.ID); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to start VM: %v\n", err)
		fmt.Println("\nCommon issues:")
		fmt.Println("- Ensure you're in the 'kvm' group (run 'groups' to check)")
		fmt.Println("- If just added, log out and back in")
		fmt.Println("- Check /dev/kvm permissions: ls -l /dev/kvm")
		return
	}
	fmt.Println("   ✓ VM started successfully!")
	fmt.Println()

	// Check status
	fmt.Println("5. Checking VM status...")
	time.Sleep(1 * time.Second) // Give VM a moment
	status, err := orch.GetVMStatus(ctx, vm.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to get status: %v\n", err)
		return
	}

	fmt.Println("   ╔════════════════════════════════════╗")
	fmt.Printf("   ║ VM: %-30s ║\n", status.ID)
	fmt.Println("   ╠════════════════════════════════════╣")
	fmt.Printf("   ║ Status:  %-25s ║\n", status.Status)
	fmt.Printf("   ║ vCPUs:   %-25d ║\n", status.Config.VCPUCount)
	fmt.Printf("   ║ Memory:  %-21d MB ║\n", status.Config.MemoryMB)
	if status.StartedAt != nil {
		fmt.Printf("   ║ Started: %-25s ║\n", status.StartedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("   ╚════════════════════════════════════╝")
	fmt.Println()

	// List VMs
	fmt.Println("6. Listing all VMs...")
	vms, err := orch.ListVMs(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to list VMs: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Found %d VM(s)\n", len(vms))
	for _, v := range vms {
		fmt.Printf("     - %s (%s)\n", v.ID, v.Status)
	}
	fmt.Println()

	// Note about command execution
	fmt.Println("7. About command execution...")
	fmt.Println("   ℹ Firecracker VMs are running!")
	fmt.Println("   ℹ For command execution, you have options:")
	fmt.Println()
	fmt.Println("   Option 1: Use Docker orchestrator (ready now)")
	fmt.Println("     ./bin/vm-cli init docker")
	fmt.Println("     ./bin/vm-cli exec my-vm echo 'Hello!'")
	fmt.Println()
	fmt.Println("   Option 2: Add SSH to Firecracker VM (advanced)")
	fmt.Println("     - Customize rootfs with SSH server")
	fmt.Println("     - Configure network tap device")
	fmt.Println("     - Connect via SSH")
	fmt.Println()
	fmt.Println("   Option 3: Implement vsock agent (advanced)")
	fmt.Println("     - Create agent binary for VM")
	fmt.Println("     - Use vsock for communication")
	fmt.Println()

	// Stop VM
	fmt.Println("8. Stopping VM...")
	if err := orch.StopVM(ctx, vm.ID, false); err != nil {
		fmt.Fprintf(os.Stderr, "   ✗ Failed to stop VM: %v\n", err)
		return
	}
	fmt.Println("   ✓ VM stopped")
	fmt.Println()

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   ✓ Firecracker Demo Complete!        ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("What worked:")
	fmt.Println("  ✓ VM creation")
	fmt.Println("  ✓ VM start (Firecracker process spawned)")
	fmt.Println("  ✓ Status checking")
	fmt.Println("  ✓ VM listing")
	fmt.Println("  ✓ VM stop")
	fmt.Println("  ✓ Cleanup")
	fmt.Println()
	fmt.Println("The microVM is fully isolated with KVM!")
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

	return nil
}
