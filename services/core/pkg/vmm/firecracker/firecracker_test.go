package firecracker

import (
	"context"
	"testing"

	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

func TestFirecrackerOrchestrator_CreateVM(t *testing.T) {
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := NewFirecrackerOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()

	vmConfig := &types.VMConfig{
		ID:         "test-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-test.sock",
		VCPUCount:  2,
		MemoryMB:   512,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	if vm.ID != "test-vm" {
		t.Errorf("Expected VM ID 'test-vm', got '%s'", vm.ID)
	}

	if vm.Status != types.VMStatusCreated {
		t.Errorf("Expected status 'CREATED', got '%s'", vm.Status)
	}

	// Cleanup
	defer orch.DeleteVM(ctx, vm.ID)
}

func TestFirecrackerOrchestrator_ListVMs(t *testing.T) {
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := NewFirecrackerOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()

	// Create 3 VMs
	for i := 1; i <= 3; i++ {
		vmConfig := &types.VMConfig{
			ID:         "test-vm-" + string(rune('0'+i)),
			KernelPath: "/var/firecracker/vmlinux",
			RootFSPath: "/var/firecracker/rootfs.ext4",
			SocketPath: "/tmp/firecracker-test-" + string(rune('0'+i)) + ".sock",
			VCPUCount:  2,
			MemoryMB:   512,
		}

		_, err := orch.CreateVM(ctx, vmConfig)
		if err != nil {
			t.Fatalf("Failed to create VM %d: %v", i, err)
		}
	}

	// List VMs
	vms, err := orch.ListVMs(ctx)
	if err != nil {
		t.Fatalf("Failed to list VMs: %v", err)
	}

	if len(vms) != 3 {
		t.Errorf("Expected 3 VMs, got %d", len(vms))
	}

	// Cleanup
	for _, vm := range vms {
		orch.DeleteVM(ctx, vm.ID)
	}
}

func TestFirecrackerOrchestrator_GetVMStatus(t *testing.T) {
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := NewFirecrackerOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()

	vmConfig := &types.VMConfig{
		ID:         "status-test-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-status-test.sock",
		VCPUCount:  2,
		MemoryMB:   512,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	// Get status
	statusVM, err := orch.GetVMStatus(ctx, vm.ID)
	if err != nil {
		t.Fatalf("Failed to get VM status: %v", err)
	}

	if statusVM.Status != types.VMStatusCreated {
		t.Errorf("Expected status 'CREATED', got '%s'", statusVM.Status)
	}

	// Cleanup
	defer orch.DeleteVM(ctx, vm.ID)
}

func TestFirecrackerOrchestrator_DeleteVM(t *testing.T) {
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      2,
		"default_memory_mb": 512,
	}

	orch, err := NewFirecrackerOrchestrator(config)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	ctx := context.Background()

	vmConfig := &types.VMConfig{
		ID:         "delete-test-vm",
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: "/tmp/firecracker-delete-test.sock",
		VCPUCount:  2,
		MemoryMB:   512,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	// Delete VM
	err = orch.DeleteVM(ctx, vm.ID)
	if err != nil {
		t.Fatalf("Failed to delete VM: %v", err)
	}

	// Try to get status - should fail
	_, err = orch.GetVMStatus(ctx, vm.ID)
	if err == nil {
		t.Error("Expected error when getting status of deleted VM, got nil")
	}
}
