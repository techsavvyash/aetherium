package network

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
)

// SquidManager manages Squid proxy process and configuration
type SquidManager struct {
	config        config.ProxyConfig
	bridgeIP      string
	subnetCIDR    string
	process       *exec.Cmd
	pid           int
	globalDomains []string
	vmWhitelists  map[string]VMWhitelistData
	mu            sync.RWMutex
	configPath    string
}

// NewSquidManager creates a new Squid manager
func NewSquidManager(proxyConfig config.ProxyConfig, bridgeIP, subnetCIDR string) (*SquidManager, error) {
	// Check if Squid is installed
	if err := checkSquidInstalled(); err != nil {
		return nil, err
	}

	// Check if SSL certificates exist (for HTTPS interception)
	if proxyConfig.RedirectHTTPS {
		if err := checkCertsExist(); err != nil {
			return nil, err
		}
	}

	sm := &SquidManager{
		config:        proxyConfig,
		bridgeIP:      extractIP(bridgeIP),
		subnetCIDR:    subnetCIDR,
		globalDomains: proxyConfig.DefaultDomains,
		vmWhitelists:  make(map[string]VMWhitelistData),
		configPath:    proxyConfig.Squid.ConfigPath,
	}

	// Generate initial configuration
	if err := sm.GenerateConfig(); err != nil {
		return nil, fmt.Errorf("failed to generate Squid config: %w", err)
	}

	return sm, nil
}

// Start starts the Squid proxy process
func (sm *SquidManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.pid != 0 {
		return fmt.Errorf("Squid is already running (PID %d)", sm.pid)
	}

	// Initialize cache directory if needed
	if err := sm.initializeCacheDir(); err != nil {
		return fmt.Errorf("failed to initialize cache directory: %w", err)
	}

	// Start Squid process
	cmd := exec.CommandContext(ctx, "squid", "-f", sm.configPath, "-N")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Squid: %w", err)
	}

	sm.process = cmd
	sm.pid = cmd.Process.Pid

	// Wait a moment for Squid to initialize
	time.Sleep(2 * time.Second)

	// Verify Squid process is still running by checking if our cmd.Process is valid
	// If Squid crashed immediately, the process would be invalid
	if sm.process == nil || sm.process.Process == nil {
		sm.pid = 0
		sm.process = nil
		return fmt.Errorf("Squid process terminated immediately after start")
	}

	// On Unix, we can send signal 0 to check if process exists without affecting it
	// This is more reliable than FindProcess which can return non-nil even for dead processes
	err := sm.process.Process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission to signal it
		// Cleanup without calling Stop() to avoid deadlock
		if sm.process != nil {
			sm.process.Process.Kill()
			sm.process.Wait()
		}
		sm.pid = 0
		sm.process = nil
		return fmt.Errorf("Squid process not responding after start: %w", err)
	}

	return nil
}

// Stop stops the Squid proxy process
func (sm *SquidManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.pid == 0 {
		return nil // Already stopped
	}

	// Try graceful shutdown first
	if err := exec.Command("squid", "-f", sm.configPath, "-k", "shutdown").Run(); err != nil {
		// If graceful shutdown fails, kill the process
		if sm.process != nil {
			sm.process.Process.Kill()
		}
	}

	// Wait for process to exit with timeout
	if sm.process != nil {
		done := make(chan error, 1)
		go func() {
			done <- sm.process.Wait()
		}()

		select {
		case <-done:
			// Process exited successfully
		case <-time.After(5 * time.Second):
			// Timeout - force kill
			if sm.process != nil && sm.process.Process != nil {
				sm.process.Process.Kill()
				sm.process.Wait() // Wait for kill to complete
			}
		}
	}

	sm.pid = 0
	sm.process = nil

	return nil
}

// Reload reloads the Squid configuration without stopping
func (sm *SquidManager) Reload() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.pid == 0 {
		return fmt.Errorf("Squid is not running")
	}

	// Send reconfigure signal to Squid
	if err := exec.Command("squid", "-f", sm.configPath, "-k", "reconfigure").Run(); err != nil {
		return fmt.Errorf("failed to reload Squid: %w", err)
	}

	return nil
}

// GenerateConfig generates Squid configuration from template
func (sm *SquidManager) GenerateConfig() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Load template
	tmpl, err := template.ParseFiles("templates/squid.conf.tmpl")
	if err != nil {
		return fmt.Errorf("failed to load template: %w", err)
	}

	// Prepare template data
	data := map[string]interface{}{
		"BridgeIP":      sm.bridgeIP,
		"Port":          sm.config.Port,
		"HTTPSPort":     sm.config.Port + 1,
		"SubnetCIDR":    sm.subnetCIDR,
		"GlobalDomains": sm.globalDomains,
		"VMWhitelists":  sm.vmWhitelists,
		"CacheDir":      sm.config.Squid.CacheDir,
		"CacheSizeMB":   sm.config.Squid.CacheSizeMB,
		"AccessLog":     sm.config.Squid.AccessLog,
		"CacheLog":      sm.config.Squid.CacheLog,
	}

	// Create config file
	file, err := os.Create(sm.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// UpdateGlobalWhitelist updates the global whitelist
func (sm *SquidManager) UpdateGlobalWhitelist(domains []string) error {
	sm.mu.Lock()
	sm.globalDomains = domains
	sm.mu.Unlock()

	// Regenerate configuration
	if err := sm.GenerateConfig(); err != nil {
		return fmt.Errorf("failed to regenerate config: %w", err)
	}

	return nil
}

// UpdateVMWhitelist updates whitelist for a specific VM
func (sm *SquidManager) UpdateVMWhitelist(vmID string, data VMWhitelistData) error {
	sm.mu.Lock()
	sm.vmWhitelists[vmID] = data
	sm.mu.Unlock()

	// Regenerate configuration
	if err := sm.GenerateConfig(); err != nil {
		return fmt.Errorf("failed to regenerate config: %w", err)
	}

	return nil
}

// RemoveVMWhitelist removes whitelist for a specific VM
func (sm *SquidManager) RemoveVMWhitelist(vmID string) error {
	sm.mu.Lock()
	delete(sm.vmWhitelists, vmID)
	sm.mu.Unlock()

	// Regenerate configuration
	if err := sm.GenerateConfig(); err != nil {
		return fmt.Errorf("failed to regenerate config: %w", err)
	}

	return nil
}

// GetStats returns current proxy statistics
func (sm *SquidManager) GetStats() (*ProxyStats, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.pid == 0 {
		return &ProxyStats{}, nil
	}

	// Get Squid stats using squidclient
	output, err := exec.Command("squidclient", "-h", sm.bridgeIP, "-p", strconv.Itoa(sm.config.Port), "mgr:info").CombinedOutput()
	if err != nil {
		// If squidclient is not available, return basic stats
		return &ProxyStats{
			Uptime: sm.getUptime(),
		}, nil
	}

	// Parse stats from output
	stats := &ProxyStats{
		Uptime: sm.getUptime(),
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Number of HTTP requests received:") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				if val, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
					stats.TotalRequests = val
				}
			}
		} else if strings.Contains(line, "Request Hit Ratios:") {
			// Parse cache hit rate
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasSuffix(part, "%") {
					if val, err := strconv.ParseFloat(strings.TrimSuffix(part, "%"), 64); err == nil {
						stats.CacheHitRate = val / 100.0
					}
				}
			}
		}
	}

	// Count blocked requests from access log
	blocked, err := sm.countBlockedRequests()
	if err == nil {
		stats.BlockedRequests = blocked
	}

	return stats, nil
}

// Health checks if Squid is running and healthy
func (sm *SquidManager) Health() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.pid == 0 {
		return fmt.Errorf("Squid is not running")
	}

	// Check if process is still alive using signal 0 (doesn't actually send a signal)
	process, err := os.FindProcess(sm.pid)
	if err != nil {
		return fmt.Errorf("Squid process not found: %w", err)
	}

	// Send signal 0 to check if process exists without affecting it
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("Squid process is not responding: %w", err)
	}

	return nil
}

// GetPID returns the Squid process ID
func (sm *SquidManager) GetPID() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.pid
}

// GetBlockedRequests returns a list of recently blocked requests
func (sm *SquidManager) GetBlockedRequests(limit int) ([]BlockedRequest, error) {
	sm.mu.RLock()
	accessLog := sm.config.Squid.AccessLog
	sm.mu.RUnlock()

	// Read access log
	file, err := os.Open(accessLog)
	if err != nil {
		if os.IsNotExist(err) {
			return []BlockedRequest{}, nil
		}
		return nil, fmt.Errorf("failed to open access log: %w", err)
	}
	defer file.Close()

	var blocked []BlockedRequest
	scanner := bufio.NewScanner(file)

	// Read last N lines (simple implementation)
	lines := make([]string, 0, limit*10)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Process lines in reverse order to get most recent first
	for i := len(lines) - 1; i >= 0 && len(blocked) < limit; i-- {
		if req, err := parseSquidLogLine(lines[i]); err == nil && req != nil {
			blocked = append(blocked, *req)
		}
	}

	return blocked, nil
}

// initializeCacheDir initializes Squid cache directory
func (sm *SquidManager) initializeCacheDir() error {
	cacheDir := sm.config.Squid.CacheDir

	// Check if cache directory is already initialized
	if _, err := os.Stat(cacheDir + "/00"); err == nil {
		return nil // Already initialized
	}

	// Initialize cache directory
	cmd := exec.Command("squid", "-f", sm.configPath, "-z")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize cache: %w (output: %s)", err, string(output))
	}

	return nil
}

// getUptime calculates proxy uptime
func (sm *SquidManager) getUptime() time.Duration {
	if sm.pid == 0 || sm.process == nil {
		return 0
	}

	// Get process start time (simplified - would need /proc parsing for accurate time)
	// For now, return a placeholder
	return time.Since(time.Now().Add(-1 * time.Hour)) // Placeholder
}

// countBlockedRequests counts blocked requests in access log
func (sm *SquidManager) countBlockedRequests() (int64, error) {
	accessLog := sm.config.Squid.AccessLog

	file, err := os.Open(accessLog)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer file.Close()

	var count int64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		// Check for TCP_DENIED or 403 status
		if strings.Contains(line, "TCP_DENIED") || strings.Contains(line, "/403 ") {
			count++
		}
	}

	return count, scanner.Err()
}

// extractIP extracts IP address from CIDR notation
func extractIP(cidr string) string {
	parts := strings.Split(cidr, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return cidr
}
