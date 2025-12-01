package network

import (
	"testing"

	"github.com/aetherium/aetherium/pkg/config"
)

// TestProxyConfigDefaults tests default proxy configuration
func TestProxyConfigDefaults(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled:  false,
		Provider: "squid",
		Port:     3128,
	}

	if proxyConfig.Provider != "squid" {
		t.Errorf("Expected provider 'squid', got '%s'", proxyConfig.Provider)
	}

	if proxyConfig.Port != 3128 {
		t.Errorf("Expected port 3128, got %d", proxyConfig.Port)
	}
}

// TestVMWhitelistData tests VM whitelist data structure
func TestVMWhitelistData(t *testing.T) {
	vmData := VMWhitelistData{
		Name: "test-vm",
		IP:   "172.16.0.2",
		Domains: []string{
			"github.com",
			"registry.npmjs.org",
		},
	}

	if vmData.Name != "test-vm" {
		t.Errorf("Expected name 'test-vm', got '%s'", vmData.Name)
	}

	if len(vmData.Domains) != 2 {
		t.Errorf("Expected 2 domains, got %d", len(vmData.Domains))
	}

	if vmData.Domains[0] != "github.com" {
		t.Errorf("Expected first domain 'github.com', got '%s'", vmData.Domains[0])
	}
}

// TestProxyStats tests proxy statistics structure
func TestProxyStats(t *testing.T) {
	stats := ProxyStats{
		TotalRequests:   100,
		BlockedRequests: 10,
		CacheHitRate:    0.75,
		BytesServed:     1024000,
	}

	if stats.TotalRequests != 100 {
		t.Errorf("Expected total requests 100, got %d", stats.TotalRequests)
	}

	if stats.BlockedRequests != 10 {
		t.Errorf("Expected blocked requests 10, got %d", stats.BlockedRequests)
	}

	if stats.CacheHitRate != 0.75 {
		t.Errorf("Expected cache hit rate 0.75, got %.2f", stats.CacheHitRate)
	}

	// Calculate block rate
	blockRate := float64(stats.BlockedRequests) / float64(stats.TotalRequests)
	if blockRate != 0.10 {
		t.Errorf("Expected block rate 0.10, got %.2f", blockRate)
	}
}

// TestBlockedRequest tests blocked request structure
func TestBlockedRequest(t *testing.T) {
	req := BlockedRequest{
		ClientIP: "172.16.0.2",
		Method:   "GET",
		URL:      "http://example.com/path",
		Domain:   "example.com",
		Reason:   "Domain not in whitelist",
	}

	if req.ClientIP != "172.16.0.2" {
		t.Errorf("Expected client IP '172.16.0.2', got '%s'", req.ClientIP)
	}

	if req.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", req.Method)
	}

	if req.Domain != "example.com" {
		t.Errorf("Expected domain 'example.com', got '%s'", req.Domain)
	}
}

// TestExtractDomain tests domain extraction from URL
func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://github.com/user/repo", "github.com"},
		{"https://registry.npmjs.org/package", "registry.npmjs.org"},
		{"http://example.com:8080/path", "example.com"},
		{"https://sub.domain.com/path?query=1", "sub.domain.com"},
		{"github.com", "github.com"},
	}

	for _, tt := range tests {
		domain := extractDomain(tt.url)
		if domain != tt.expected {
			t.Errorf("extractDomain(%s): expected '%s', got '%s'", tt.url, tt.expected, domain)
		}
	}
}

// TestExtractIP tests IP extraction from CIDR
func TestExtractIP(t *testing.T) {
	tests := []struct {
		cidr     string
		expected string
	}{
		{"172.16.0.1/24", "172.16.0.1"},
		{"10.0.0.1/16", "10.0.0.1"},
		{"192.168.1.1/32", "192.168.1.1"},
		{"172.16.0.1", "172.16.0.1"}, // No CIDR
	}

	for _, tt := range tests {
		ip := extractIP(tt.cidr)
		if ip != tt.expected {
			t.Errorf("extractIP(%s): expected '%s', got '%s'", tt.cidr, tt.expected, ip)
		}
	}
}

// TestProxyManagerWithoutSquid tests proxy manager creation when disabled
func TestProxyManagerWithoutSquid(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled: false,
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	if pm == nil {
		t.Fatal("Expected proxy manager, got nil")
	}

	if pm.running {
		t.Error("Expected proxy to not be running when disabled")
	}

	if pm.IsRunning() {
		t.Error("IsRunning() should return false when proxy is disabled")
	}
}

// TestProxyManagerStats tests getting stats from disabled proxy
func TestProxyManagerStats(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled: false,
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	stats, err := pm.GetStats()
	if err != nil {
		t.Errorf("GetStats() should not fail for disabled proxy: %v", err)
	}

	if stats == nil {
		t.Error("Expected empty stats, got nil")
	}
}

// TestProxyManagerBlockedRequests tests getting blocked requests from disabled proxy
func TestProxyManagerBlockedRequests(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled: false,
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	blocked, err := pm.GetBlockedRequests(10)
	if err != nil {
		t.Errorf("GetBlockedRequests() should not fail for disabled proxy: %v", err)
	}

	if blocked == nil {
		t.Error("Expected empty slice, got nil")
	}

	if len(blocked) != 0 {
		t.Errorf("Expected 0 blocked requests, got %d", len(blocked))
	}
}

// TestProxyManagerHealth tests health check for disabled proxy
func TestProxyManagerHealth(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled: false,
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	err = pm.Health()
	if err != nil {
		t.Errorf("Health() should not fail for disabled proxy: %v", err)
	}
}

// TestProxyManagerUpdateWhitelist tests updating whitelist when proxy is disabled
func TestProxyManagerUpdateWhitelist(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled:        false,
		DefaultDomains: []string{"github.com"},
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	newDomains := []string{"github.com", "registry.npmjs.org", "pypi.org"}
	err = pm.UpdateWhitelist(newDomains)
	if err != nil {
		t.Errorf("UpdateWhitelist() should not fail for disabled proxy: %v", err)
	}

	// Verify domains were updated in config
	if len(pm.config.DefaultDomains) != len(newDomains) {
		t.Errorf("Expected %d domains, got %d", len(newDomains), len(pm.config.DefaultDomains))
	}
}

// TestProxyManagerUpdateVMWhitelist tests updating VM whitelist when proxy is disabled
func TestProxyManagerUpdateVMWhitelist(t *testing.T) {
	proxyConfig := config.ProxyConfig{
		Enabled: false,
	}

	pm, err := NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		t.Fatalf("Failed to create proxy manager: %v", err)
	}

	vmID := "test-vm-123"
	vmName := "test-vm"
	vmIP := "172.16.0.2"
	domains := []string{"github.com", "registry.npmjs.org"}

	err = pm.UpdateVMWhitelist(vmID, vmName, vmIP, domains)
	if err != nil {
		t.Errorf("UpdateVMWhitelist() should not fail for disabled proxy: %v", err)
	}
}

// BenchmarkExtractDomain benchmarks domain extraction
func BenchmarkExtractDomain(b *testing.B) {
	url := "https://registry.npmjs.org/package/version"
	for i := 0; i < b.N; i++ {
		extractDomain(url)
	}
}

// BenchmarkExtractIP benchmarks IP extraction
func BenchmarkExtractIP(b *testing.B) {
	cidr := "172.16.0.1/24"
	for i := 0; i < b.N; i++ {
		extractIP(cidr)
	}
}
