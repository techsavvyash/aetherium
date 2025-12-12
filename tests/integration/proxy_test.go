package integration

import (
	"context"
	"testing"
	"time"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
	"github.com/aetherium/aetherium/services/core/pkg/network"
	"github.com/aetherium/aetherium/services/core/pkg/service"
	"github.com/aetherium/aetherium/services/core/pkg/storage/postgres"
	"github.com/aetherium/aetherium/services/core/pkg/queue/asynq"
	"github.com/aetherium/aetherium/services/core/pkg/vmm/firecracker"
	"github.com/aetherium/aetherium/services/core/pkg/worker"
)

// TestProxyWhitelistingBasic tests basic proxy setup and whitelisting
func TestProxyWhitelistingBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proxy integration test in short mode")
	}

	_ = context.Background() // context not used in this basic test

	// Setup network with proxy enabled
	proxyConfig := config.ProxyConfig{
		Enabled:       true,
		Provider:      "squid",
		Transparent:   true,
		Port:          3128,
		WhitelistMode: "enforce",
		DefaultDomains: []string{
			"github.com",
			"githubusercontent.com",
			"registry.npmjs.org",
		},
		RedirectHTTP:  true,
		RedirectHTTPS: false, // Skip HTTPS for basic test
		Squid: config.SquidConfig{
			ConfigPath:  "/etc/squid/aetherium-test.conf",
			CacheDir:    "/var/spool/squid-aetherium-test",
			CacheSizeMB: 512,
			AccessLog:   "/var/log/squid/aetherium-test-access.log",
			CacheLog:    "/var/log/squid/aetherium-test-cache.log",
		},
	}

	networkConfig := network.NetworkConfig{
		BridgeName:    "aetherium0",
		BridgeIP:      "172.16.0.1/24",
		SubnetCIDR:    "172.16.0.0/24",
		TapPrefix:     "aether-",
		EnableNAT:     true,
		HostInterface: "",
	}

	netMgr, err := network.NewManagerWithProxy(networkConfig, proxyConfig)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}

	// Setup bridge (idempotent)
	if err := netMgr.SetupBridge(); err != nil {
		t.Fatalf("Failed to setup bridge: %v", err)
	}
	defer netMgr.Shutdown()

	t.Run("ProxyHealthCheck", func(t *testing.T) {
		// Check proxy health
		if err := netMgr.GetProxyHealth(); err != nil {
			t.Errorf("Proxy health check failed: %v", err)
		} else {
			t.Log("✓ Proxy is healthy")
		}
	})

	t.Run("GetProxyStats", func(t *testing.T) {
		// Get proxy statistics
		stats, err := netMgr.GetProxyStats()
		if err != nil {
			t.Errorf("Failed to get proxy stats: %v", err)
		} else {
			t.Logf("✓ Proxy stats: %d total requests, %d blocked, %.2f%% cache hit rate",
				stats.TotalRequests, stats.BlockedRequests, stats.CacheHitRate*100)
		}
	})

	t.Run("UpdateGlobalWhitelist", func(t *testing.T) {
		// Update global whitelist
		newDomains := []string{
			"github.com",
			"githubusercontent.com",
			"registry.npmjs.org",
			"pypi.org",
			"files.pythonhosted.org",
		}

		if err := netMgr.UpdateGlobalWhitelist(newDomains); err != nil {
			t.Errorf("Failed to update global whitelist: %v", err)
		} else {
			t.Logf("✓ Global whitelist updated with %d domains", len(newDomains))
		}

		// Allow proxy to reload
		time.Sleep(2 * time.Second)
	})
}

// TestProxyWithVM tests proxy functionality with an actual VM
func TestProxyWithVM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping VM proxy test in short mode")
	}

	ctx := context.Background()

	// Setup infrastructure
	store, err := postgres.NewStore(postgres.Config{
		Host:     "localhost",
		Port:     5432,
		User:     "aetherium",
		Password: "aetherium",
		Database: "aetherium",
		SSLMode:  "disable",
	})
	if err != nil {
		t.Fatalf("Failed to initialize store: %v", err)
	}
	defer store.Close()

	queue, err := asynq.NewQueue(asynq.Config{
		RedisAddr: "localhost:6379",
		Queues: map[string]int{
			"default": 10,
		},
	})
	if err != nil {
		t.Fatalf("Failed to initialize queue: %v", err)
	}

	// Setup network with proxy
	proxyConfig := config.ProxyConfig{
		Enabled:       true,
		Provider:      "squid",
		Transparent:   true,
		Port:          3128,
		WhitelistMode: "enforce",
		DefaultDomains: []string{
			"github.com",
			"githubusercontent.com",
		},
		RedirectHTTP:  true,
		RedirectHTTPS: false,
		Squid: config.SquidConfig{
			ConfigPath:  "/etc/squid/aetherium-vm-test.conf",
			CacheDir:    "/var/spool/squid-aetherium-vm-test",
			CacheSizeMB: 512,
			AccessLog:   "/var/log/squid/aetherium-vm-test-access.log",
			CacheLog:    "/var/log/squid/aetherium-vm-test-cache.log",
		},
	}

	networkConfig := network.NetworkConfig{
		BridgeName:    "aetherium0",
		BridgeIP:      "172.16.0.1/24",
		SubnetCIDR:    "172.16.0.0/24",
		TapPrefix:     "aether-",
		EnableNAT:     true,
		HostInterface: "",
	}

	netMgr, err := network.NewManagerWithProxy(networkConfig, proxyConfig)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}
	defer netMgr.Shutdown()

	if err := netMgr.SetupBridge(); err != nil {
		t.Fatalf("Failed to setup bridge: %v", err)
	}

	// Create orchestrator with network manager
	orchestrator, err := firecracker.NewFirecrackerOrchestratorWithNetwork(map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 512,
	}, netMgr)
	if err != nil {
		t.Fatalf("Failed to initialize orchestrator: %v", err)
	}

	// Start worker
	w := worker.New(store, orchestrator)
	if err := w.RegisterHandlers(queue); err != nil {
		t.Fatalf("Failed to register handlers: %v", err)
	}

	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := queue.Start(workerCtx); err != nil {
		t.Fatalf("Failed to start queue: %v", err)
	}
	defer func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer stopCancel()
		queue.Stop(stopCtx)
	}()

	taskService := service.NewTaskService(queue, store)

	t.Run("CreateVMWithProxy", func(t *testing.T) {
		// Create VM
		taskID, err := taskService.CreateVMTask(ctx, "proxy-test-vm", 1, 512)
		if err != nil {
			t.Fatalf("Failed to create VM task: %v", err)
		}

		t.Logf("Created VM task: %s", taskID)

		// Wait for VM creation
		time.Sleep(30 * time.Second)

		// Get VM
		vm, err := taskService.GetVMByName(ctx, "proxy-test-vm")
		if err != nil {
			t.Fatalf("Failed to get VM: %v", err)
		}

		t.Logf("VM created: %s (Status: %s)", vm.ID, vm.Status)

		// Register VM-specific whitelist
		vmDomains := []string{
			"github.com",
			"githubusercontent.com",
			"registry.npmjs.org",
		}

		if err := netMgr.RegisterVMWithProxy(vm.ID.String(), "proxy-test-vm", vmDomains); err != nil {
			t.Errorf("Failed to register VM with proxy: %v", err)
		} else {
			t.Logf("✓ VM registered with proxy, whitelist: %v", vmDomains)
		}

		// Allow proxy to reload
		time.Sleep(2 * time.Second)

		// Test whitelisted domain (should succeed)
		t.Run("TestWhitelistedDomain", func(t *testing.T) {
			taskID, err := taskService.ExecuteCommandTask(
				ctx,
				vm.ID.String(),
				"curl",
				[]string{"-I", "--max-time", "10", "http://github.com"},
			)
			if err != nil {
				t.Errorf("Failed to execute curl to whitelisted domain: %v", err)
			} else {
				t.Logf("✓ Curl to whitelisted domain (github.com) task: %s", taskID)
			}

			time.Sleep(5 * time.Second)

			// Check execution result
			executions, err := taskService.GetExecutions(ctx, vm.ID)
			if err != nil {
				t.Errorf("Failed to get executions: %v", err)
			} else if len(executions) > 0 {
				lastExec := executions[len(executions)-1]
				if lastExec.ExitCode != nil && *lastExec.ExitCode == 0 {
					t.Log("✓ Whitelisted domain request succeeded")
				} else {
					t.Logf("Whitelisted domain request exit code: %d", *lastExec.ExitCode)
				}
			}
		})

		// Test non-whitelisted domain (should fail/block)
		t.Run("TestBlockedDomain", func(t *testing.T) {
			taskID, err := taskService.ExecuteCommandTask(
				ctx,
				vm.ID.String(),
				"curl",
				[]string{"-I", "--max-time", "10", "http://example.com"},
			)
			if err != nil {
				t.Logf("Curl to non-whitelisted domain returned error (expected): %v", err)
			} else {
				t.Logf("Curl to blocked domain (example.com) task: %s", taskID)
			}

			time.Sleep(5 * time.Second)

			// Check blocked requests
			blocked, err := netMgr.GetBlockedRequests(10)
			if err != nil {
				t.Errorf("Failed to get blocked requests: %v", err)
			} else {
				t.Logf("✓ Blocked requests count: %d", len(blocked))
				for _, req := range blocked {
					t.Logf("  - Blocked: %s %s from %s (reason: %s)",
						req.Method, req.URL, req.ClientIP, req.Reason)
				}
			}
		})

		// Check final proxy stats
		t.Run("FinalProxyStats", func(t *testing.T) {
			stats, err := netMgr.GetProxyStats()
			if err != nil {
				t.Errorf("Failed to get final proxy stats: %v", err)
			} else {
				t.Logf("✓ Final proxy stats:")
				t.Logf("  - Total requests: %d", stats.TotalRequests)
				t.Logf("  - Blocked requests: %d", stats.BlockedRequests)
				t.Logf("  - Cache hit rate: %.2f%%", stats.CacheHitRate*100)
				t.Logf("  - Uptime: %s", stats.Uptime)
			}
		})

		// Unregister VM from proxy
		if err := netMgr.UnregisterVMFromProxy(vm.ID.String()); err != nil {
			t.Errorf("Failed to unregister VM from proxy: %v", err)
		} else {
			t.Log("✓ VM unregistered from proxy")
		}

		// Cleanup VM
		_, err = taskService.DeleteVMTask(ctx, vm.ID.String())
		if err != nil {
			t.Errorf("Failed to delete VM: %v", err)
		}
	})
}

// TestProxyServiceLayer tests the service layer abstraction
func TestProxyServiceLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping proxy service test in short mode")
	}

	ctx := context.Background()

	// Setup network manager
	proxyConfig := config.ProxyConfig{
		Enabled:       true,
		Provider:      "squid",
		Transparent:   true,
		Port:          3128,
		WhitelistMode: "enforce",
		DefaultDomains: []string{
			"github.com",
		},
		RedirectHTTP:  true,
		RedirectHTTPS: false,
		Squid: config.SquidConfig{
			ConfigPath:  "/etc/squid/aetherium-service-test.conf",
			CacheDir:    "/var/spool/squid-aetherium-service-test",
			CacheSizeMB: 512,
			AccessLog:   "/var/log/squid/aetherium-service-test-access.log",
			CacheLog:    "/var/log/squid/aetherium-service-test-cache.log",
		},
	}

	networkConfig := network.NetworkConfig{
		BridgeName: "aetherium0",
		BridgeIP:   "172.16.0.1/24",
		SubnetCIDR: "172.16.0.0/24",
		TapPrefix:  "aether-",
		EnableNAT:  true,
	}

	netMgr, err := network.NewManagerWithProxy(networkConfig, proxyConfig)
	if err != nil {
		t.Fatalf("Failed to create network manager: %v", err)
	}
	defer netMgr.Shutdown()

	if err := netMgr.SetupBridge(); err != nil {
		t.Fatalf("Failed to setup bridge: %v", err)
	}

	// Create proxy service
	proxyService := service.NewProxyService(netMgr)

	t.Run("UpdateGlobalWhitelistViaService", func(t *testing.T) {
		domains := []string{
			"github.com",
			"githubusercontent.com",
			"registry.npmjs.org",
			"pypi.org",
		}

		if err := proxyService.UpdateGlobalWhitelist(ctx, domains); err != nil {
			t.Errorf("Failed to update global whitelist via service: %v", err)
		} else {
			t.Logf("✓ Global whitelist updated via service: %v", domains)
		}
	})

	t.Run("RegisterVMDomainsViaService", func(t *testing.T) {
		vmID := "test-vm-123"
		vmName := "test-vm"
		domains := []string{
			"github.com",
			"registry.npmjs.org",
		}

		if err := proxyService.RegisterVMDomains(ctx, vmID, vmName, domains); err != nil {
			t.Errorf("Failed to register VM domains via service: %v", err)
		} else {
			t.Logf("✓ VM domains registered via service: %v", domains)
		}

		// Unregister
		if err := proxyService.UnregisterVM(ctx, vmID); err != nil {
			t.Errorf("Failed to unregister VM via service: %v", err)
		} else {
			t.Log("✓ VM unregistered via service")
		}
	})

	t.Run("GetProxyStatsViaService", func(t *testing.T) {
		stats, err := proxyService.GetProxyStats(ctx)
		if err != nil {
			t.Errorf("Failed to get proxy stats via service: %v", err)
		} else {
			t.Logf("✓ Proxy stats via service: %d total, %d blocked",
				stats.TotalRequests, stats.BlockedRequests)
		}
	})

	t.Run("GetProxyHealthViaService", func(t *testing.T) {
		if err := proxyService.GetProxyHealth(ctx); err != nil {
			t.Errorf("Proxy health check via service failed: %v", err)
		} else {
			t.Log("✓ Proxy is healthy (checked via service)")
		}
	})

	t.Run("GetBlockedRequestsViaService", func(t *testing.T) {
		blocked, err := proxyService.GetBlockedRequests(ctx, 10)
		if err != nil {
			t.Errorf("Failed to get blocked requests via service: %v", err)
		} else {
			t.Logf("✓ Blocked requests via service: %d entries", len(blocked))
			for _, req := range blocked {
				t.Logf("  - %s %s from %s", req.Method, req.URL, req.ClientIP)
			}
		}
	})
}
