package firecracker

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetherium/aetherium/pkg/network"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

// FirecrackerOrchestrator implements vmm.VMOrchestrator using the official Firecracker SDK
type FirecrackerOrchestrator struct {
	config         *Config
	vms            map[string]*vmHandle
	networkManager *network.Manager
}

// Config represents Firecracker-specific configuration
type Config struct {
	KernelPath      string
	RootFSTemplate  string
	SocketDir       string
	DefaultVCPU     int
	DefaultMemoryMB int
}

type vmHandle struct {
	vm      *types.VM
	machine *firecracker.Machine
}

// NewFirecrackerOrchestrator creates a new Firecracker VMM orchestrator using the official SDK
func NewFirecrackerOrchestrator(configMap map[string]interface{}) (*FirecrackerOrchestrator, error) {
	config := &Config{
		KernelPath:      configMap["kernel_path"].(string),
		RootFSTemplate:  configMap["rootfs_template"].(string),
		SocketDir:       configMap["socket_dir"].(string),
		DefaultVCPU:     configMap["default_vcpu"].(int),
		DefaultMemoryMB: configMap["default_memory_mb"].(int),
	}

	// Create network manager
	netMgr, err := network.NewManager(network.NetworkConfig{
		BridgeName:    "aetherium0",
		BridgeIP:      "172.16.0.1/24",
		SubnetCIDR:    "172.16.0.0/24",
		TapPrefix:     "aether-",
		EnableNAT:     true,
		HostInterface: "", // Auto-detect
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create network manager: %w", err)
	}

	// Setup bridge
	if err := netMgr.SetupBridge(); err != nil {
		return nil, fmt.Errorf("failed to setup network bridge: %w", err)
	}

	return &FirecrackerOrchestrator{
		config:         config,
		vms:            make(map[string]*vmHandle),
		networkManager: netMgr,
	}, nil
}

// CreateVM creates a new Firecracker VM
func (f *FirecrackerOrchestrator) CreateVM(ctx context.Context, config *types.VMConfig) (*types.VM, error) {
	// Validate that kernel and rootfs exist
	if _, err := os.Stat(config.KernelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kernel not found: %s", config.KernelPath)
	}
	if _, err := os.Stat(config.RootFSPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("rootfs not found: %s", config.RootFSPath)
	}

	// Create VM struct
	vm := &types.VM{
		ID:     config.ID,
		Status: types.VMStatusCreated,
		Config: *config,
	}

	// Build Firecracker SDK configuration
	vcpuCount := int64(config.VCPUCount)
	memSizeMib := int64(config.MemoryMB)

	// Create TAP device for network
	tapDevice, err := f.networkManager.CreateTAPDevice(config.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to create TAP device: %w", err)
	}

	// Create log file for Firecracker logs (not VM console output)
	logPath := config.SocketPath + ".log"

	// Build kernel args with network configuration
	kernelArgs := fmt.Sprintf("console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda rw ip=%s::172.16.0.1:255.255.255.0::eth0:off:8.8.8.8",
		tapDevice.IPAddress[:len(tapDevice.IPAddress)-3]) // Remove /24 suffix

	fcConfig := firecracker.Config{
		SocketPath:      config.SocketPath,
		KernelImagePath: config.KernelPath,
		KernelArgs:      kernelArgs,
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String(config.RootFSPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(vcpuCount),
			MemSizeMib: firecracker.Int64(memSizeMib),
		},
		// Add network interface
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  tapDevice.MACAddr,
					HostDevName: tapDevice.Name,
				},
			},
		},
		// Add vsock device for agent communication
		VsockDevices: []firecracker.VsockDevice{
			{
				Path: config.SocketPath + ".vsock",
				CID:  uint32(3), // Guest CID (host is always 2)
			},
		},
		// Enable logging
		LogPath:  logPath,
		LogLevel: "Debug",
	}

	// Create the machine (doesn't start it yet)
	// Use context.Background() so machine lifecycle is not tied to task context
	machine, err := firecracker.NewMachine(context.Background(), fcConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create firecracker machine: %w", err)
	}

	f.vms[config.ID] = &vmHandle{
		vm:      vm,
		machine: machine,
	}

	return vm, nil
}

// StartVM starts a Firecracker VM
func (f *FirecrackerOrchestrator) StartVM(ctx context.Context, vmID string) error {
	handle, exists := f.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if handle.vm.Status != types.VMStatusCreated {
		return fmt.Errorf("VM %s is not in created state (current: %s)", vmID, handle.vm.Status)
	}

	handle.vm.Status = types.VMStatusStarting

	// Start the VM using the SDK
	// Use context.Background() so VM process outlives the creation task
	// The VM should continue running after the task completes
	if err := handle.machine.Start(context.Background()); err != nil {
		handle.vm.Status = types.VMStatusFailed
		return fmt.Errorf("failed to start VM: %w", err)
	}

	handle.vm.Status = types.VMStatusRunning
	now := time.Now()
	handle.vm.StartedAt = &now

	return nil
}

// StopVM stops a Firecracker VM
func (f *FirecrackerOrchestrator) StopVM(ctx context.Context, vmID string, force bool) error {
	handle, exists := f.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if handle.vm.Status != types.VMStatusRunning {
		return fmt.Errorf("VM %s is not running (status: %s)", vmID, handle.vm.Status)
	}

	handle.vm.Status = types.VMStatusStopping

	var err error
	if force {
		// Force stop using StopVMM
		err = handle.machine.StopVMM()
	} else {
		// Graceful shutdown
		err = handle.machine.Shutdown(ctx)
	}

	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	handle.vm.Status = types.VMStatusStopped
	now := time.Now()
	handle.vm.StoppedAt = &now

	return nil
}

// GetVMStatus returns the current status of a VM
func (f *FirecrackerOrchestrator) GetVMStatus(ctx context.Context, vmID string) (*types.VM, error) {
	handle, exists := f.vms[vmID]
	if !exists {
		return nil, fmt.Errorf("VM %s not found", vmID)
	}

	return handle.vm, nil
}

// DeleteVM destroys a VM and cleans up resources
func (f *FirecrackerOrchestrator) DeleteVM(ctx context.Context, vmID string) error {
	handle, exists := f.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Stop if running
	if handle.vm.Status == types.VMStatusRunning {
		if err := f.StopVM(ctx, vmID, true); err != nil {
			return fmt.Errorf("failed to stop VM during delete: %w", err)
		}
	}

	// Clean up TAP device
	if err := f.networkManager.DeleteTAPDevice(vmID); err != nil {
		// Log but don't fail - TAP device might not exist
		fmt.Printf("Warning: failed to delete TAP device for VM %s: %v\n", vmID, err)
	}

	// Clean up sockets
	os.Remove(handle.vm.Config.SocketPath)
	os.Remove(handle.vm.Config.SocketPath + ".vsock")

	// Remove from map
	delete(f.vms, vmID)

	return nil
}

// ListVMs returns all VMs
func (f *FirecrackerOrchestrator) ListVMs(ctx context.Context) ([]*types.VM, error) {
	vms := make([]*types.VM, 0, len(f.vms))
	for _, handle := range f.vms {
		vms = append(vms, handle.vm)
	}
	return vms, nil
}

// StreamLogs streams logs from a VM
func (f *FirecrackerOrchestrator) StreamLogs(ctx context.Context, vmID string) (<-chan string, error) {
	// TODO: Implement log streaming via serial console
	logChan := make(chan string)
	close(logChan)
	return logChan, nil
}

// ExecuteCommand is implemented in exec.go

// Health returns the health status of the orchestrator
func (f *FirecrackerOrchestrator) Health(ctx context.Context) error {
	// Check if firecracker binary exists
	fcPath := findFirecrackerBinary()
	if fcPath == "" {
		return fmt.Errorf("firecracker binary not found in PATH")
	}

	// Check if kernel exists
	if _, err := os.Stat(f.config.KernelPath); os.IsNotExist(err) {
		return fmt.Errorf("kernel not found: %s", f.config.KernelPath)
	}

	// Check if rootfs template exists
	if _, err := os.Stat(f.config.RootFSTemplate); os.IsNotExist(err) {
		return fmt.Errorf("rootfs template not found: %s", f.config.RootFSTemplate)
	}

	// Check if /dev/kvm is accessible
	if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
		return fmt.Errorf("/dev/kvm not found - KVM not available")
	}

	kvm, err := os.OpenFile("/dev/kvm", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("/dev/kvm not accessible: %w (add user to kvm group)", err)
	}
	kvm.Close()

	return nil
}

func findFirecrackerBinary() string {
	// Check standard locations
	locations := []string{
		"/usr/local/bin/firecracker",
		"/usr/bin/firecracker",
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// Ensure FirecrackerOrchestrator implements vmm.VMOrchestrator
var _ vmm.VMOrchestrator = (*FirecrackerOrchestrator)(nil)
