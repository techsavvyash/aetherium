package network

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sync"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
)

// NetworkConfig holds network configuration
type NetworkConfig struct {
	BridgeName    string
	BridgeIP      string
	SubnetCIDR    string
	TapPrefix     string
	EnableNAT     bool
	HostInterface string
}

// Manager manages network resources for VMs
type Manager struct {
	config       NetworkConfig
	tapDevices   map[string]*TAPDevice
	ipAllocator  *IPAllocator
	proxyManager *ProxyManager
	mu           sync.Mutex
	bridgeSetup  bool
}

// TAPDevice represents a TAP network device
type TAPDevice struct {
	Name      string
	IPAddress string
	Gateway   string
	MACAddr   string
}

// IPAllocator manages IP address allocation
type IPAllocator struct {
	subnet      *net.IPNet
	allocated   map[string]bool
	nextOffset  int
	mu          sync.Mutex
}

// NewManager creates a new network manager
func NewManager(config NetworkConfig) (*Manager, error) {
	_, subnet, err := net.ParseCIDR(config.SubnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR: %w", err)
	}

	return &Manager{
		config:      config,
		tapDevices:  make(map[string]*TAPDevice),
		ipAllocator: &IPAllocator{
			subnet:     subnet,
			allocated:  make(map[string]bool),
			nextOffset: 2, // Start from .2 (.1 is gateway)
		},
	}, nil
}

// NewManagerWithProxy creates a new network manager with proxy support
func NewManagerWithProxy(netConfig NetworkConfig, proxyConfig config.ProxyConfig) (*Manager, error) {
	_, subnet, err := net.ParseCIDR(netConfig.SubnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid subnet CIDR: %w", err)
	}

	manager := &Manager{
		config:      netConfig,
		tapDevices:  make(map[string]*TAPDevice),
		ipAllocator: &IPAllocator{
			subnet:     subnet,
			allocated:  make(map[string]bool),
			nextOffset: 2, // Start from .2 (.1 is gateway)
		},
	}

	// Initialize proxy manager if enabled
	if proxyConfig.Enabled {
		proxyMgr, err := NewProxyManager(proxyConfig, netConfig.BridgeIP, netConfig.SubnetCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy manager: %w", err)
		}
		manager.proxyManager = proxyMgr
	}

	return manager, nil
}

// SetupBridge creates and configures the bridge interface
func (m *Manager) SetupBridge() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.bridgeSetup {
		return nil
	}

	// Check if bridge exists
	iface, err := net.InterfaceByName(m.config.BridgeName)
	if err == nil {
		// Bridge already exists, check if it's already up
		if iface.Flags&net.FlagUp != 0 {
			// Bridge is already up and configured
			fmt.Printf("✓ Using existing bridge %s\n", m.config.BridgeName)
			m.bridgeSetup = true
			return nil
		}
		// Bridge exists but is down, try to bring it up (might need sudo)
		if err := exec.Command("ip", "link", "set", m.config.BridgeName, "up").Run(); err != nil {
			return fmt.Errorf("bridge exists but cannot bring it up (needs sudo): %w\n\nRun: sudo ./scripts/setup-network.sh", err)
		}
		m.bridgeSetup = true
		return nil
	}

	// Create bridge
	if err := exec.Command("ip", "link", "add", m.config.BridgeName, "type", "bridge").Run(); err != nil {
		return fmt.Errorf("failed to create bridge: %w", err)
	}

	// Set bridge IP
	if err := exec.Command("ip", "addr", "add", m.config.BridgeIP, "dev", m.config.BridgeName).Run(); err != nil {
		// Ignore if address already exists
		if err := exec.Command("ip", "link", "set", m.config.BridgeName, "up").Run(); err != nil {
			return fmt.Errorf("failed to bring up bridge: %w", err)
		}
	} else {
		// Bring bridge up
		if err := exec.Command("ip", "link", "set", m.config.BridgeName, "up").Run(); err != nil {
			return fmt.Errorf("failed to bring up bridge: %w", err)
		}
	}

	// Setup NAT if enabled
	if m.config.EnableNAT {
		if err := m.setupNAT(); err != nil {
			return fmt.Errorf("failed to setup NAT: %w", err)
		}
	}

	// Start proxy if enabled
	if m.proxyManager != nil {
		ctx := context.Background()
		if err := m.proxyManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}
	}

	m.bridgeSetup = true
	return nil
}

// setupNAT configures NAT for VMs
func (m *Manager) setupNAT() error {
	// Enable IP forwarding
	if err := exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Add iptables NAT rule
	hostIface := m.config.HostInterface
	if hostIface == "" {
		// Try to detect default interface
		var err error
		hostIface, err = getDefaultInterface()
		if err != nil {
			return fmt.Errorf("failed to detect host interface: %w", err)
		}
	}

	log.Printf("Network: Using host interface %s for NAT", hostIface)

	// Clean up any old NAT rules with wrong interfaces (e.g., eth0)
	// This prevents issues when switching network interfaces
	commonWrongIfaces := []string{"eth0", "eth1", "wlan0", "wlan1"}
	for _, wrongIface := range commonWrongIfaces {
		if wrongIface != hostIface {
			// Try to delete rule with wrong interface (ignore errors if it doesn't exist)
			exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING",
				"-s", m.config.SubnetCIDR, "-o", wrongIface, "-j", "MASQUERADE").Run()
		}
	}

	// Check if rule already exists with correct interface
	checkCmd := exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING",
		"-s", m.config.SubnetCIDR, "-o", hostIface, "-j", "MASQUERADE")
	if checkCmd.Run() != nil {
		// Rule doesn't exist, add it
		addCmd := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING",
			"-s", m.config.SubnetCIDR, "-o", hostIface, "-j", "MASQUERADE")
		if err := addCmd.Run(); err != nil {
			return fmt.Errorf("failed to add NAT rule: %w", err)
		}
		log.Printf("Network: Added NAT MASQUERADE rule for %s -> %s", m.config.SubnetCIDR, hostIface)
	}

	// Allow forwarding from bridge (inbound)
	checkForward := exec.Command("iptables", "-C", "FORWARD",
		"-i", m.config.BridgeName, "-j", "ACCEPT")
	if checkForward.Run() != nil {
		addForward := exec.Command("iptables", "-A", "FORWARD",
			"-i", m.config.BridgeName, "-j", "ACCEPT")
		if err := addForward.Run(); err != nil {
			return fmt.Errorf("failed to add forward rule (inbound): %w", err)
		}
		log.Printf("Network: Added FORWARD rule for bridge inbound traffic")
	}

	// Allow forwarding to bridge (outbound/return traffic)
	checkForwardOut := exec.Command("iptables", "-C", "FORWARD",
		"-o", m.config.BridgeName, "-j", "ACCEPT")
	if checkForwardOut.Run() != nil {
		addForwardOut := exec.Command("iptables", "-A", "FORWARD",
			"-o", m.config.BridgeName, "-j", "ACCEPT")
		if err := addForwardOut.Run(); err != nil {
			return fmt.Errorf("failed to add forward rule (outbound): %w", err)
		}
		log.Printf("Network: Added FORWARD rule for bridge outbound traffic")
	}

	return nil
}

// CreateTAPDevice creates a TAP device for a VM
func (m *Manager) CreateTAPDevice(vmID string) (*TAPDevice, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if tap, exists := m.tapDevices[vmID]; exists {
		return tap, nil
	}

	// Allocate IP
	ip, err := m.ipAllocator.AllocateIP()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	// Generate TAP device name
	suffix := vmID
	if len(vmID) > 8 {
		suffix = vmID[:8]
	}
	tapName := fmt.Sprintf("%s%s", m.config.TapPrefix, suffix)
	if len(tapName) > 15 {
		tapName = tapName[:15]
	}

	// Create TAP device
	if err := exec.Command("ip", "tuntap", "add", tapName, "mode", "tap").Run(); err != nil {
		m.ipAllocator.ReleaseIP(ip)
		return nil, fmt.Errorf("failed to create TAP device (needs CAP_NET_ADMIN): %w\n\nThe worker needs network privileges. Either:\n  1. Run worker with sudo: sudo ./bin/worker\n  2. Or grant capability: sudo setcap cap_net_admin+ep ./bin/worker\n  3. Or use start script: sudo ./scripts/start-worker.sh", err)
	}

	// Attach to bridge
	if err := exec.Command("ip", "link", "set", tapName, "master", m.config.BridgeName).Run(); err != nil {
		exec.Command("ip", "link", "delete", tapName).Run()
		m.ipAllocator.ReleaseIP(ip)
		return nil, fmt.Errorf("failed to attach TAP to bridge: %w", err)
	}

	// Bring TAP up
	if err := exec.Command("ip", "link", "set", tapName, "up").Run(); err != nil {
		exec.Command("ip", "link", "delete", tapName).Run()
		m.ipAllocator.ReleaseIP(ip)
		return nil, fmt.Errorf("failed to bring up TAP device: %w", err)
	}

	// Generate MAC address
	macAddr := generateMAC(vmID)

	tap := &TAPDevice{
		Name:      tapName,
		IPAddress: ip,
		Gateway:   m.config.BridgeIP,
		MACAddr:   macAddr,
	}

	m.tapDevices[vmID] = tap
	return tap, nil
}

// DeleteTAPDevice removes a TAP device
func (m *Manager) DeleteTAPDevice(vmID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tap, exists := m.tapDevices[vmID]
	if !exists {
		return nil
	}

	// Delete TAP device
	if err := exec.Command("ip", "link", "delete", tap.Name).Run(); err != nil {
		// Ignore error if device doesn't exist
	}

	// Release IP
	m.ipAllocator.ReleaseIP(tap.IPAddress)

	delete(m.tapDevices, vmID)
	return nil
}

// AllocateIP allocates an IP address from the subnet
func (a *IPAllocator) AllocateIP() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Convert subnet to usable IP range
	ip := make(net.IP, len(a.subnet.IP))
	copy(ip, a.subnet.IP)

	// Try to find an available IP
	for i := 0; i < 253; i++ {
		offset := a.nextOffset + i
		if offset > 254 {
			offset = 2 + (offset - 255)
		}

		ip[3] = byte(offset)
		ipStr := ip.String()

		if !a.allocated[ipStr] {
			a.allocated[ipStr] = true
			a.nextOffset = offset + 1
			if a.nextOffset > 254 {
				a.nextOffset = 2
			}
			return ipStr + "/24", nil
		}
	}

	return "", fmt.Errorf("no available IPs in subnet")
}

// ReleaseIP releases an IP address
func (a *IPAllocator) ReleaseIP(ipWithMask string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Remove /24 suffix
	ip := ipWithMask
	if idx := len(ipWithMask) - 3; idx > 0 && ipWithMask[idx:] == "/24" {
		ip = ipWithMask[:idx]
	}

	delete(a.allocated, ip)
}

// getDefaultInterface returns the default network interface
func getDefaultInterface() (string, error) {
	// Get default route
	cmd := exec.Command("ip", "route", "show", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output: "default via X.X.X.X dev INTERFACE ..."
	fields := string(output)
	if len(fields) == 0 {
		return "", fmt.Errorf("no default route found")
	}

	// Simple parsing - look for "dev <interface>"
	lines := string(output)
	start := 0
	for i, c := range lines {
		if c == ' ' && i+4 < len(lines) && lines[i+1:i+4] == "dev" {
			start = i + 5
			for j := start; j < len(lines); j++ {
				if lines[j] == ' ' || lines[j] == '\n' {
					return lines[start:j], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not parse default interface")
}

// generateMAC generates a MAC address based on VM ID
func generateMAC(vmID string) string {
	// Use a simple hash of the VM ID to generate MAC
	// Format: 52:54:00:XX:XX:XX (QEMU/KVM range)
	hash := 0
	for _, c := range vmID {
		hash = (hash * 31) + int(c)
	}

	return fmt.Sprintf("52:54:00:%02x:%02x:%02x",
		byte((hash>>16)&0xFF),
		byte((hash>>8)&0xFF),
		byte(hash&0xFF))
}

// Proxy-related methods

// RegisterVMWithProxy registers a VM with the proxy whitelist
func (m *Manager) RegisterVMWithProxy(vmID, vmName string, domains []string) error {
	m.mu.Lock()
	tap, exists := m.tapDevices[vmID]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("VM %s not found", vmID)
	}

	if m.proxyManager == nil {
		return nil // Proxy not enabled
	}

	// Extract IP without /24 suffix
	vmIP := tap.IPAddress
	if idx := len(vmIP) - 3; idx > 0 && vmIP[idx:] == "/24" {
		vmIP = vmIP[:idx]
	}

	return m.proxyManager.UpdateVMWhitelist(vmID, vmName, vmIP, domains)
}

// UnregisterVMFromProxy removes a VM from the proxy whitelist
func (m *Manager) UnregisterVMFromProxy(vmID string) error {
	if m.proxyManager == nil {
		return nil // Proxy not enabled
	}

	if m.proxyManager.squidManager != nil {
		if err := m.proxyManager.squidManager.RemoveVMWhitelist(vmID); err != nil {
			return err
		}
		return m.proxyManager.Reload()
	}

	return nil
}

// UpdateGlobalWhitelist updates the global proxy whitelist
func (m *Manager) UpdateGlobalWhitelist(domains []string) error {
	if m.proxyManager == nil {
		return nil // Proxy not enabled
	}

	return m.proxyManager.UpdateWhitelist(domains)
}

// GetProxyStats returns proxy statistics
func (m *Manager) GetProxyStats() (*ProxyStats, error) {
	if m.proxyManager == nil {
		return &ProxyStats{}, nil
	}

	return m.proxyManager.GetStats()
}

// GetProxyHealth checks proxy health
func (m *Manager) GetProxyHealth() error {
	if m.proxyManager == nil {
		return nil // Proxy not enabled
	}

	return m.proxyManager.Health()
}

// GetBlockedRequests returns recently blocked requests
func (m *Manager) GetBlockedRequests(limit int) ([]BlockedRequest, error) {
	if m.proxyManager == nil {
		return []BlockedRequest{}, nil
	}

	return m.proxyManager.GetBlockedRequests(limit)
}

// Shutdown gracefully shuts down the network manager
func (m *Manager) Shutdown() error {
	// Stop proxy if running
	if m.proxyManager != nil {
		if err := m.proxyManager.Stop(); err != nil {
			return fmt.Errorf("failed to stop proxy: %w", err)
		}
	}

	return nil
}
