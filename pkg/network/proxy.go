package network

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aetherium/aetherium/pkg/config"
)

// ProxyManager manages the proxy infrastructure for VM traffic filtering
type ProxyManager struct {
	config       config.ProxyConfig
	bridgeIP     string
	subnetCIDR   string
	squidManager *SquidManager
	mu           sync.RWMutex
	running      bool
	squidPID     int
}

// ProxyStats holds proxy statistics
type ProxyStats struct {
	TotalRequests   int64
	BlockedRequests int64
	CacheHitRate    float64
	BytesServed     int64
	Uptime          time.Duration
}

// VMWhitelistData holds per-VM whitelist configuration
type VMWhitelistData struct {
	Name    string
	IP      string
	Domains []string
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(proxyConfig config.ProxyConfig, bridgeIP, subnetCIDR string) (*ProxyManager, error) {
	if !proxyConfig.Enabled {
		return &ProxyManager{
			config:     proxyConfig,
			bridgeIP:   bridgeIP,
			subnetCIDR: subnetCIDR,
			running:    false,
		}, nil
	}

	// Create Squid manager based on provider
	var squidMgr *SquidManager
	var err error

	if proxyConfig.Provider == "squid" {
		squidMgr, err = NewSquidManager(proxyConfig, bridgeIP, subnetCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to create Squid manager: %w", err)
		}
	}

	return &ProxyManager{
		config:       proxyConfig,
		bridgeIP:     bridgeIP,
		subnetCIDR:   subnetCIDR,
		squidManager: squidMgr,
		running:      false,
	}, nil
}

// Start starts the proxy service
func (pm *ProxyManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.config.Enabled {
		return nil
	}

	if pm.running {
		return nil
	}

	// Start Squid
	if pm.squidManager != nil {
		if err := pm.squidManager.Start(ctx); err != nil {
			return fmt.Errorf("failed to start Squid: %w", err)
		}
		pm.squidPID = pm.squidManager.GetPID()
	}

	// Setup iptables rules for transparent proxy
	if pm.config.Transparent {
		if err := pm.setupTransparentProxy(); err != nil {
			// Try to stop Squid on failure
			if pm.squidManager != nil {
				pm.squidManager.Stop()
			}
			return fmt.Errorf("failed to setup transparent proxy: %w", err)
		}
	}

	pm.running = true
	fmt.Printf("✓ Proxy started (%s on %s:%d)\n", pm.config.Provider, pm.bridgeIP, pm.config.Port)
	return nil
}

// Stop stops the proxy service
func (pm *ProxyManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.config.Enabled || !pm.running {
		return nil
	}

	// Remove iptables rules
	if pm.config.Transparent {
		pm.removeTransparentProxyRules()
	}

	// Stop Squid
	if pm.squidManager != nil {
		if err := pm.squidManager.Stop(); err != nil {
			return fmt.Errorf("failed to stop Squid: %w", err)
		}
	}

	pm.running = false
	pm.squidPID = 0
	fmt.Println("✓ Proxy stopped")
	return nil
}

// Reload reloads the proxy configuration without stopping
func (pm *ProxyManager) Reload() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.config.Enabled || !pm.running {
		return nil
	}

	if pm.squidManager != nil {
		if err := pm.squidManager.Reload(); err != nil {
			return fmt.Errorf("failed to reload Squid: %w", err)
		}
	}

	fmt.Println("✓ Proxy configuration reloaded")
	return nil
}

// UpdateWhitelist updates the global whitelist
func (pm *ProxyManager) UpdateWhitelist(domains []string) error {
	pm.mu.Lock()
	pm.config.DefaultDomains = domains

	if !pm.config.Enabled || !pm.running {
		pm.mu.Unlock()
		return nil
	}

	if pm.squidManager != nil {
		if err := pm.squidManager.UpdateGlobalWhitelist(domains); err != nil {
			pm.mu.Unlock()
			return fmt.Errorf("failed to update whitelist: %w", err)
		}
	}
	pm.mu.Unlock()

	// Call Reload() after releasing the lock to avoid deadlock
	return pm.Reload()
}

// UpdateVMWhitelist updates whitelist for a specific VM
func (pm *ProxyManager) UpdateVMWhitelist(vmID, vmName, vmIP string, domains []string) error {
	pm.mu.Lock()

	if !pm.config.Enabled || !pm.running {
		pm.mu.Unlock()
		return nil
	}

	if pm.squidManager != nil {
		vmData := VMWhitelistData{
			Name:    vmName,
			IP:      vmIP,
			Domains: domains,
		}
		if err := pm.squidManager.UpdateVMWhitelist(vmID, vmData); err != nil {
			pm.mu.Unlock()
			return fmt.Errorf("failed to update VM whitelist: %w", err)
		}
	}
	pm.mu.Unlock()

	// Call Reload() after releasing the lock to avoid deadlock
	return pm.Reload()
}

// GetStats returns current proxy statistics
func (pm *ProxyManager) GetStats() (*ProxyStats, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.config.Enabled || !pm.running {
		return &ProxyStats{}, nil
	}

	if pm.squidManager != nil {
		return pm.squidManager.GetStats()
	}

	return &ProxyStats{}, nil
}

// Health checks if the proxy is running and healthy
func (pm *ProxyManager) Health() error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.config.Enabled {
		return nil
	}

	if !pm.running {
		return fmt.Errorf("proxy is not running")
	}

	if pm.squidManager != nil {
		return pm.squidManager.Health()
	}

	return nil
}

// IsRunning returns whether the proxy is currently running
func (pm *ProxyManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.running
}

// setupTransparentProxy configures iptables for transparent proxy
func (pm *ProxyManager) setupTransparentProxy() error {
	bridgeName := "aetherium0" // TODO: Make this configurable

	// HTTP redirection
	if pm.config.RedirectHTTP {
		rule := []string{
			"-t", "nat",
			"-A", "PREROUTING",
			"-i", bridgeName,
			"-p", "tcp",
			"--dport", "80",
			"-j", "REDIRECT",
			"--to-port", strconv.Itoa(pm.config.Port),
		}

		// Check if rule exists
		checkRule := append([]string{"-t", "nat", "-C", "PREROUTING"}, rule[4:]...)
		if err := exec.Command("iptables", checkRule...).Run(); err != nil {
			// Rule doesn't exist, add it
			if err := exec.Command("iptables", rule...).Run(); err != nil {
				return fmt.Errorf("failed to add HTTP redirect rule: %w", err)
			}
			fmt.Println("✓ HTTP redirect rule added")
		}
	}

	// HTTPS redirection (port 443 -> 3129 for SSL bump)
	if pm.config.RedirectHTTPS {
		httpsPort := pm.config.Port + 1 // HTTPS port is typically port+1
		rule := []string{
			"-t", "nat",
			"-A", "PREROUTING",
			"-i", bridgeName,
			"-p", "tcp",
			"--dport", "443",
			"-j", "REDIRECT",
			"--to-port", strconv.Itoa(httpsPort),
		}

		// Check if rule exists
		checkRule := append([]string{"-t", "nat", "-C", "PREROUTING"}, rule[4:]...)
		if err := exec.Command("iptables", checkRule...).Run(); err != nil {
			// Rule doesn't exist, add it
			if err := exec.Command("iptables", rule...).Run(); err != nil {
				return fmt.Errorf("failed to add HTTPS redirect rule: %w", err)
			}
			fmt.Println("✓ HTTPS redirect rule added")
		}
	}

	return nil
}

// removeTransparentProxyRules removes iptables rules for transparent proxy
func (pm *ProxyManager) removeTransparentProxyRules() {
	bridgeName := "aetherium0"

	// Remove HTTP redirect rule
	if pm.config.RedirectHTTP {
		rule := []string{
			"-t", "nat",
			"-D", "PREROUTING",
			"-i", bridgeName,
			"-p", "tcp",
			"--dport", "80",
			"-j", "REDIRECT",
			"--to-port", strconv.Itoa(pm.config.Port),
		}
		exec.Command("iptables", rule...).Run() // Ignore errors on cleanup
	}

	// Remove HTTPS redirect rule
	if pm.config.RedirectHTTPS {
		httpsPort := pm.config.Port + 1
		rule := []string{
			"-t", "nat",
			"-D", "PREROUTING",
			"-i", bridgeName,
			"-p", "tcp",
			"--dport", "443",
			"-j", "REDIRECT",
			"--to-port", strconv.Itoa(httpsPort),
		}
		exec.Command("iptables", rule...).Run() // Ignore errors on cleanup
	}

	fmt.Println("✓ Proxy iptables rules removed")
}

// checkSquidInstalled checks if Squid is installed on the system
func checkSquidInstalled() error {
	if _, err := exec.LookPath("squid"); err != nil {
		return fmt.Errorf("squid is not installed. Run: sudo ./scripts/setup-squid.sh")
	}
	return nil
}

// checkCertsExist checks if SSL certificates exist for HTTPS interception
func checkCertsExist() error {
	certPath := "/etc/squid/certs/aetherium-ca.pem"
	keyPath := "/etc/squid/certs/aetherium-ca-key.pem"

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("SSL certificate not found at %s. Run: sudo ./scripts/generate-ssl-certs.sh", certPath)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("SSL private key not found at %s. Run: sudo ./scripts/generate-ssl-certs.sh", keyPath)
	}

	return nil
}

// GetBlockedRequests returns a list of recently blocked requests (parsed from logs)
func (pm *ProxyManager) GetBlockedRequests(limit int) ([]BlockedRequest, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if !pm.config.Enabled || !pm.running {
		return []BlockedRequest{}, nil
	}

	if pm.squidManager != nil {
		return pm.squidManager.GetBlockedRequests(limit)
	}

	return []BlockedRequest{}, nil
}

// BlockedRequest represents a blocked HTTP request
type BlockedRequest struct {
	Timestamp time.Time
	ClientIP  string
	Method    string
	URL       string
	Domain    string
	Reason    string
}

// parseSquidLogLine parses a single Squid access log line
func parseSquidLogLine(line string) (*BlockedRequest, error) {
	// Squid log format: timestamp elapsed remotehost code/status bytes method URL rfc931 peerstatus/peerhost type
	fields := strings.Fields(line)
	if len(fields) < 10 {
		return nil, fmt.Errorf("invalid log line format")
	}

	// Parse timestamp (Unix time with milliseconds)
	timestampSecs := strings.Split(fields[0], ".")[0]
	timestamp, err := strconv.ParseInt(timestampSecs, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	// Parse response code (e.g., "TCP_DENIED/403")
	codeParts := strings.Split(fields[3], "/")
	if len(codeParts) < 2 {
		return nil, fmt.Errorf("invalid code format")
	}

	action := codeParts[0]
	statusCode := codeParts[1]

	// Only return blocked requests (403, TCP_DENIED, etc.)
	if statusCode != "403" && action != "TCP_DENIED" {
		return nil, nil // Not a blocked request
	}

	return &BlockedRequest{
		Timestamp: time.Unix(timestamp, 0),
		ClientIP:  fields[2],
		Method:    fields[5],
		URL:       fields[6],
		Domain:    extractDomain(fields[6]),
		Reason:    "Domain not in whitelist",
	}, nil
}

// extractDomain extracts domain from URL
func extractDomain(url string) string {
	// Remove protocol
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")

	// Get domain (before first /)
	parts := strings.Split(url, "/")
	domain := parts[0]

	// Remove port if present
	domain = strings.Split(domain, ":")[0]

	return domain
}
