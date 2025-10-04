package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("Aetherium Firecracker VMM Demonstration")
	fmt.Println("========================================\n")

	ctx := context.Background()

	// Create orchestrator
	fmt.Println("1. Creating Firecracker Orchestrator...")
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := firecracker.NewFirecrackerOrchestrator(config)
	if err != nil {
		fmt.Printf("❌ Failed to create orchestrator: %v\n", err)
		return
	}
	fmt.Println("✅ Orchestrator created\n")

	// Create VM 1
	fmt.Println("2. Creating VM: demo-vm-1...")
	vm1Config := &types.VMConfig{
		ID:         "demo-vm-1",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-demo-1.sock",
		VCPUCount:  2,
		MemoryMB:   512,
	}

	vm1, err := orch.CreateVM(ctx, vm1Config)
	if err != nil {
		fmt.Printf("❌ Failed to create VM: %v\n", err)
		return
	}
	fmt.Printf("✅ VM created: %s (Status: %s)\n\n", vm1.ID, vm1.Status)

	// Create VM 2
	fmt.Println("3. Creating VM: demo-vm-2...")
	vm2Config := &types.VMConfig{
		ID:         "demo-vm-2",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-demo-2.sock",
		VCPUCount:  1,
		MemoryMB:   256,
	}

	vm2, err := orch.CreateVM(ctx, vm2Config)
	if err != nil {
		fmt.Printf("❌ Failed to create VM: %v\n", err)
		orch.DeleteVM(ctx, vm1.ID)
		return
	}
	fmt.Printf("✅ VM created: %s (Status: %s)\n\n", vm2.ID, vm2.Status)

	// List VMs
	fmt.Println("4. Listing all VMs...")
	vms, err := orch.ListVMs(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to list VMs: %v\n", err)
		return
	}

	fmt.Printf("Found %d VMs:\n", len(vms))
	for _, vm := range vms {
		fmt.Printf("  - %s: Status=%s, vCPUs=%d, Memory=%dMB\n",
			vm.ID, vm.Status, vm.Config.VCPUCount, vm.Config.MemoryMB)
	}
	fmt.Println()

	// Get status
	fmt.Println("5. Checking VM status...")
	status1, err := orch.GetVMStatus(ctx, vm1.ID)
	if err != nil {
		fmt.Printf("❌ Failed to get status: %v\n", err)
		return
	}
	fmt.Printf("✅ VM %s status: %s\n\n", status1.ID, status1.Status)

	// Note: Starting VMs requires Firecracker binary, kernel, and rootfs
	fmt.Println("6. Simulating VM lifecycle...")
	fmt.Println("   (Note: Actual VM start requires Firecracker binary + kernel/rootfs)")
	fmt.Println("   The following would start the VM if prerequisites exist:")
	fmt.Printf("   - orch.StartVM(ctx, \"%s\")\n", vm1.ID)
	fmt.Printf("   - orch.StopVM(ctx, \"%s\", false)\n\n", vm1.ID)

	// Wait a moment
	time.Sleep(1 * time.Second)

	// Cleanup
	fmt.Println("7. Cleaning up VMs...")
	for _, vm := range vms {
		err := orch.DeleteVM(ctx, vm.ID)
		if err != nil {
			fmt.Printf("❌ Failed to delete VM %s: %v\n", vm.ID, err)
		} else {
			fmt.Printf("✅ VM %s deleted\n", vm.ID)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Demonstration Complete!")
	fmt.Println("========================================")
	fmt.Println("\nKey Features Demonstrated:")
	fmt.Println("  ✓ Firecracker orchestrator initialization")
	fmt.Println("  ✓ VM creation with custom configuration")
	fmt.Println("  ✓ Multi-VM management")
	fmt.Println("  ✓ VM listing and status queries")
	fmt.Println("  ✓ Resource cleanup")
	fmt.Println("\nNext Steps:")
	fmt.Println("  - Install Firecracker binary to /usr/bin/firecracker")
	fmt.Println("  - Download kernel to /var/firecracker/vmlinux")
	fmt.Println("  - Download rootfs to /var/firecracker/rootfs.ext4")
	fmt.Println("  - Run: ./bin/fc-cli create my-vm && ./bin/fc-cli start my-vm")
}
