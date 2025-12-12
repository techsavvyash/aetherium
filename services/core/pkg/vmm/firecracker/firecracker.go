package firecracker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/aetherium/aetherium/pkg/network"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/mdlayher/vsock"
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
	vm        *types.VM
	machine   *firecracker.Machine
	ipAddress string // VM's IP address for TCP fallback
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

// NewFirecrackerOrchestratorWithNetwork creates a new Firecracker orchestrator with a pre-configured network manager
// This is useful for testing or when you want to use a network manager with custom configuration (e.g., with proxy enabled)
func NewFirecrackerOrchestratorWithNetwork(configMap map[string]interface{}, netMgr *network.Manager) (*FirecrackerOrchestrator, error) {
	config := &Config{
		KernelPath:      configMap["kernel_path"].(string),
		RootFSTemplate:  configMap["rootfs_template"].(string),
		SocketDir:       configMap["socket_dir"].(string),
		DefaultVCPU:     configMap["default_vcpu"].(int),
		DefaultMemoryMB: configMap["default_memory_mb"].(int),
	}

	return &FirecrackerOrchestrator{
		config:         config,
		vms:            make(map[string]*vmHandle),
		networkManager: netMgr,
	}, nil
}

// createVMRootfs creates a per-VM copy of the rootfs template
// This ensures VM isolation - each VM gets its own rootfs copy to prevent corruption
func (f *FirecrackerOrchestrator) createVMRootfs(ctx context.Context, vmID string) (string, error) {
	templatePath := "/var/firecracker/rootfs-template.ext4"
	vmRootfsPath := fmt.Sprintf("/var/firecracker/rootfs-vm-%s.ext4", vmID)

	// Check if template exists
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return "", fmt.Errorf("rootfs template not found at %s - ensure init container ran successfully", templatePath)
	}

	// Copy template to VM-specific rootfs
	// Using cp --reflink=auto enables copy-on-write on supported filesystems (XFS, Btrfs)
	// This makes the copy instantaneous and space-efficient
	cmd := exec.CommandContext(ctx, "cp", "--reflink=auto", templatePath, vmRootfsPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create VM rootfs from template: %w, output: %s", err, string(output))
	}

	// Make VM rootfs writable (template is read-only)
	if err := os.Chmod(vmRootfsPath, 0644); err != nil {
		// Cleanup partial file on error
		os.Remove(vmRootfsPath)
		return "", fmt.Errorf("failed to set permissions on VM rootfs: %w", err)
	}

	log.Printf("Created per-VM rootfs: %s (copy-on-write from template)", vmRootfsPath)
	return vmRootfsPath, nil
}

// CreateVM creates a new Firecracker VM
func (f *FirecrackerOrchestrator) CreateVM(ctx context.Context, config *types.VMConfig) (*types.VM, error) {
	// Validate that kernel exists
	if _, err := os.Stat(config.KernelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("kernel not found: %s", config.KernelPath)
	}

	// Create per-VM rootfs from template (for isolation)
	// If config.RootFSPath is empty or points to old shared rootfs, create new per-VM copy
	if config.RootFSPath == "" || config.RootFSPath == "/var/firecracker/rootfs.ext4" {
		vmRootfsPath, err := f.createVMRootfs(ctx, config.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create per-VM rootfs: %w", err)
		}
		config.RootFSPath = vmRootfsPath
	} else {
		// Validate custom rootfs path exists (for backwards compatibility)
		if _, err := os.Stat(config.RootFSPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("rootfs not found: %s", config.RootFSPath)
		}
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

	// Extract IP address (remove CIDR suffix like /24)
	vmIP := tapDevice.IPAddress
	if idx := len(vmIP) - 3; idx > 0 && vmIP[idx] == '/' {
		vmIP = vmIP[:idx]
	}

	f.vms[config.ID] = &vmHandle{
		vm:        vm,
		machine:   machine,
		ipAddress: vmIP,
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

	// Clean up per-VM rootfs (self-healing)
	// Only delete if it's a per-VM rootfs (matches pattern rootfs-vm-{id}.ext4)
	vmRootfsPath := fmt.Sprintf("/var/firecracker/rootfs-vm-%s.ext4", vmID)
	if _, err := os.Stat(vmRootfsPath); err == nil {
		if err := os.Remove(vmRootfsPath); err != nil {
			log.Printf("Warning: Failed to delete VM rootfs %s: %v", vmRootfsPath, err)
			// Don't fail the entire deletion if rootfs cleanup fails
			// The init container will clean it up on next pod restart
		} else {
			log.Printf("Deleted per-VM rootfs: %s", vmRootfsPath)
		}
	}

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

	// Check if rootfs template exists (per-VM isolation system)
	templatePath := "/var/firecracker/rootfs-template.ext4"
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("rootfs template not found: %s (init container may not have run)", templatePath)
	}

	// Check for excessive orphaned rootfs files
	orphanedCount := 0
	vmRootfsFiles, _ := filepath.Glob("/var/firecracker/rootfs-vm-*.ext4")

	activeVMs := len(f.vms)

	// Count orphaned files (VM rootfs exists but VM not in memory)
	for _, file := range vmRootfsFiles {
		vmID := filepath.Base(file)
		vmID = vmID[len("rootfs-vm-") : len(vmID)-len(".ext4")]

		_, exists := f.vms[vmID]

		if !exists {
			orphanedCount++
		}
	}

	// Warn if orphaned files exceed threshold (may indicate init container cleanup issue)
	if orphanedCount > 10 {
		return fmt.Errorf("excessive orphaned rootfs files detected: %d orphaned, %d active VMs (init container should clean these)", orphanedCount, activeVMs)
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

// ProvideSecretsOnBoot establishes a vsock connection and sends secrets to the VM on boot
// This method listens for a reverse connection from the VM's fc-agent during boot
func (f *FirecrackerOrchestrator) ProvideSecretsOnBoot(ctx context.Context, vmID string, secrets map[string]string) error {
	if len(secrets) == 0 {
		log.Printf("No secrets to provide for VM %s", vmID)
		return nil
	}

	log.Printf("Preparing to provide %d secrets to VM %s", len(secrets), vmID)

	// Get VM handle to verify it exists
	_, exists := f.vms[vmID]
	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	// Create vsock listener on host for VM to connect to
	// The VM will dial host CID 2 (always), port 9998
	// Use the Firecracker SDK's vsock device to accept connections
	// We need to listen on the host side for the VM to dial
	// The vsock.Listen function creates a listener that the guest can connect to
	listener, err := vsock.Listen(9998, &vsock.Config{})
	if err != nil {
		return fmt.Errorf("failed to create vsock listener: %w", err)
	}
	defer listener.Close()

	log.Printf("Vsock listener created on port 9998, waiting for VM %s to connect...", vmID)

	// Accept connection with timeout
	acceptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Accept in goroutine to support timeout
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	var conn net.Conn
	select {
	case conn = <-connChan:
		defer conn.Close()
	case err := <-errChan:
		return fmt.Errorf("failed to accept secret connection: %w", err)
	case <-acceptCtx.Done():
		return fmt.Errorf("timeout waiting for VM to connect for secrets")
	}

	log.Printf("VM %s connected to fetch secrets", vmID)

	// Read GET_SECRETS request from VM
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read secret request: %w", err)
	}

	// Parse request
	type Request struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload,omitempty"`
	}

	var request Request
	if err := json.Unmarshal([]byte(line), &request); err != nil {
		return fmt.Errorf("failed to parse secret request: %w", err)
	}

	if request.Type != "get_secrets" {
		return fmt.Errorf("unexpected request type: %s (expected get_secrets)", request.Type)
	}

	// Prepare response with secrets
	type Response struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload,omitempty"`
		Error   string          `json:"error,omitempty"`
	}

	secretsPayload, err := json.Marshal(secrets)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	response := Response{
		Type:    "success",
		Payload: secretsPayload,
	}

	// Send response
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if _, err := conn.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to send secrets: %w", err)
	}

	log.Printf("✓ Successfully provided %d secrets to VM %s (in-memory only)", len(secrets), vmID)
	return nil
}

// Ensure FirecrackerOrchestrator implements vmm.VMOrchestrator
var _ vmm.VMOrchestrator = (*FirecrackerOrchestrator)(nil)
