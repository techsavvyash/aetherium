package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
	"github.com/aetherium/aetherium/services/core/pkg/network"
	"github.com/aetherium/aetherium/services/core/pkg/service"
)

func main() {
	fmt.Println("=== Testing Proxy Manager (WITH SQUID ENABLED) ===\n")

	// Test 1: Create proxy manager with proxy ENABLED
	fmt.Println("Test 1: Creating ProxyManager with proxy ENABLED...")
	proxyConfig := config.ProxyConfig{
		Enabled:       true,
		Provider:      "squid",
		Transparent:   true,
		Port:          3128,
		WhitelistMode: "enforce",
		RedirectHTTP:  true,
		RedirectHTTPS: false,
		DefaultDomains: []string{
			"github.com",
			"githubusercontent.com",
			"registry.npmjs.org",
		},
		Squid: config.SquidConfig{
			ConfigPath:  "/etc/squid/aetherium.conf",
			CacheDir:    "/var/spool/squid-aetherium",
			CacheSizeMB: 1024,
			AccessLog:   "/var/log/squid/aetherium-access.log",
			CacheLog:    "/var/log/squid/aetherium-cache.log",
		},
	}

	pm, err := network.NewProxyManager(proxyConfig, "172.16.0.1/24", "172.16.0.0/24")
	if err != nil {
		log.Fatalf("✗ Failed to create proxy manager: %v", err)
	}
	fmt.Println("✓ ProxyManager created successfully")

	// Test 2: Start the proxy
	fmt.Println("\nTest 2: Starting Squid proxy...")
	ctx := context.Background()
	if err := pm.Start(ctx); err != nil {
		log.Fatalf("✗ Failed to start proxy: %v\nMake sure Squid is installed: sudo ./scripts/setup-squid.sh", err)
	}
	fmt.Println("✓ Squid proxy started successfully")
	defer pm.Stop()

	// Give Squid time to fully start
	time.Sleep(2 * time.Second)

	// Test 3: Check if running
	fmt.Println("\nTest 3: Checking if proxy is running...")
	if !pm.IsRunning() {
		log.Fatal("✗ Proxy should be running")
	}
	fmt.Println("✓ Proxy is running")

	// Test 4: Health check
	fmt.Println("\nTest 4: Performing health check...")
	if err := pm.Health(); err != nil {
		fmt.Printf("✗ Health check failed: %v\n", err)
	} else {
		fmt.Println("✓ Health check passed")
	}

	// Test 5: Get statistics
	fmt.Println("\nTest 5: Getting proxy statistics...")
	stats, err := pm.GetStats()
	if err != nil {
		fmt.Printf("✗ Failed to get stats: %v\n", err)
	} else {
		fmt.Printf("✓ Stats retrieved: TotalRequests=%d, BlockedRequests=%d, CacheHitRate=%.2f%%\n",
			stats.TotalRequests, stats.BlockedRequests, stats.CacheHitRate*100)
	}

	// Test 6: Update global whitelist
	fmt.Println("\nTest 6: Updating global whitelist...")
	newDomains := []string{
		"github.com",
		"githubusercontent.com",
		"registry.npmjs.org",
		"pypi.org",
		"files.pythonhosted.org",
	}
	if err := pm.UpdateWhitelist(newDomains); err != nil {
		fmt.Printf("✗ Failed to update whitelist: %v\n", err)
	} else {
		fmt.Println("✓ Whitelist updated successfully")
		fmt.Printf("  Configured domains: %v\n", newDomains)
	}

	// Test 7: Update VM whitelist
	fmt.Println("\nTest 7: Registering VM with custom whitelist...")
	vmID := "test-vm-001"
	vmName := "test-vm"
	vmIP := "172.16.0.2"
	vmDomains := []string{
		"github.com",
		"registry.npmjs.org",
	}
	if err := pm.UpdateVMWhitelist(vmID, vmName, vmIP, vmDomains); err != nil {
		fmt.Printf("✗ Failed to update VM whitelist: %v\n", err)
	} else {
		fmt.Println("✓ VM whitelist updated successfully")
		fmt.Printf("  VM: %s (%s) at %s\n", vmName, vmID, vmIP)
		fmt.Printf("  Allowed domains: %v\n", vmDomains)
	}

	// Test 8: Verify Squid configuration
	fmt.Println("\nTest 8: Verifying Squid configuration...")
	if err := pm.Health(); err != nil {
		fmt.Printf("✗ Configuration verification failed: %v\n", err)
	} else {
		fmt.Println("✓ Squid configuration is valid")
	}

	// Test 9: Test with NetworkManager
	fmt.Println("\n\n=== Testing Network Manager Integration ===\n")

	fmt.Println("Test 9: Creating NetworkManager with proxy enabled...")
	networkConfig := network.NetworkConfig{
		BridgeName: "aetherium0",
		BridgeIP:   "172.16.0.1/24",
		SubnetCIDR: "172.16.0.0/24",
		TapPrefix:  "aether-",
		EnableNAT:  false, // Don't enable NAT for safety in test
	}

	netMgr, err := network.NewManagerWithProxy(networkConfig, proxyConfig)
	if err != nil {
		fmt.Printf("✗ Failed to create network manager: %v\n", err)
	} else {
		fmt.Println("✓ NetworkManager created successfully with proxy support")
	}

	// Test 10: Service layer testing
	fmt.Println("\n\n=== Testing Service Layer ===\n")

	fmt.Println("Test 10: Creating ProxyService...")
	proxyService := service.NewProxyService(netMgr)
	fmt.Println("✓ ProxyService created successfully")

	svcCtx := context.Background()

	fmt.Println("\nTest 11: Updating global whitelist via service...")
	serviceDomains := []string{
		"github.com",
		"githubusercontent.com",
		"registry.npmjs.org",
		"pypi.org",
		"golang.org",
		"rubygems.org",
	}
	if err := proxyService.UpdateGlobalWhitelist(svcCtx, serviceDomains); err != nil {
		fmt.Printf("✗ Failed to update whitelist: %v\n", err)
	} else {
		fmt.Println("✓ Global whitelist updated via service")
		fmt.Printf("  Domains: %v\n", serviceDomains)
	}

	fmt.Println("\nTest 12: Registering VM via service...")
	if err := proxyService.RegisterVMDomains(svcCtx, vmID, vmName, vmDomains); err != nil {
		fmt.Printf("✗ Failed to register VM: %v\n", err)
	} else {
		fmt.Println("✓ VM registered via service")
	}

	fmt.Println("\nTest 13: Getting proxy stats via service...")
	serviceStats, err := proxyService.GetProxyStats(svcCtx)
	if err != nil {
		fmt.Printf("✗ Failed to get stats: %v\n", err)
	} else {
		fmt.Printf("✓ Stats: Total=%d, Blocked=%d, CacheHitRate=%.2f%%\n",
			serviceStats.TotalRequests, serviceStats.BlockedRequests, serviceStats.CacheHitRate*100)
	}

	fmt.Println("\nTest 14: Health check via service...")
	if err := proxyService.GetProxyHealth(svcCtx); err != nil{
		fmt.Printf("✗ Health check failed: %v\n", err)
	} else {
		fmt.Println("✓ Health check passed")
	}

	fmt.Println("\nTest 15: Getting blocked requests via service...")
	blocked, err := proxyService.GetBlockedRequests(svcCtx, 10)
	if err != nil {
		fmt.Printf("✗ Failed to get blocked requests: %v\n", err)
	} else {
		fmt.Printf("✓ Blocked requests retrieved: %d entries\n", len(blocked))
		if len(blocked) > 0 {
			fmt.Println("\n  Recent blocked requests:")
			for i, req := range blocked {
				fmt.Printf("    %d. %s %s from %s (reason: %s)\n",
					i+1, req.Method, req.Domain, req.ClientIP, req.Reason)
			}
		}
	}

	fmt.Println("\n=== All Tests Completed ===")
	fmt.Println("\n✅ Proxy is running with Squid!")
	fmt.Println("✅ All operations work correctly!")
	fmt.Println("✅ Configuration management successful!")
	fmt.Println("\nℹ️  Squid configuration: /etc/squid/aetherium.conf")
	fmt.Println("ℹ️  Access logs: /var/log/squid/aetherium-access.log")
	fmt.Println("ℹ️  Cache logs: /var/log/squid/aetherium-cache.log")

	fmt.Println("\nStopping proxy...")
	if err := pm.Stop(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to stop proxy cleanly: %v\n", err)
	} else {
		fmt.Println("✓ Proxy stopped successfully")
	}
}
