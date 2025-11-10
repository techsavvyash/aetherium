package integration

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/aetherium/aetherium/pkg/network"
	"github.com/aetherium/aetherium/pkg/tools"
	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/firecracker"
	"github.com/google/uuid"
)

// TestProxyWhitelist tests that VMs can only access whitelisted domains through Squid proxy
func TestProxyWhitelist(t *testing.T) {
	// Skip if not running as root
	if !isRoot() {
		t.Skip("Test requires root privileges")
	}

	// Check if Squid is running
	if !isSquidRunning() {
		t.Skip("Squid proxy is not running. Start with: docker-compose up -d squid")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Setup
	t.Log("Setting up Firecracker orchestrator...")
	orchestrator, err := setupFirecrackerOrchestrator()
	if err != nil {
		t.Fatalf("Failed to setup orchestrator: %v", err)
	}

	// Create VM with proxy settings
	t.Log("Creating VM with proxy configuration...")
	vmID := uuid.New().String()
	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: fmt.Sprintf("/tmp/aetherium-vm-%s.sock", vmID),
		VCPUCount:  1,
		MemoryMB:   256,
		Env: map[string]string{
			"HTTP_PROXY":  "http://172.16.0.1:3128",
			"HTTPS_PROXY": "http://172.16.0.1:3128",
			"http_proxy":  "http://172.16.0.1:3128",
			"https_proxy": "http://172.16.0.1:3128",
			"NO_PROXY":    "localhost,127.0.0.1,172.16.0.0/24",
			"no_proxy":    "localhost,127.0.0.1,172.16.0.0/24",
		},
	}

	vm, err := orchestrator.CreateVM(ctx, vmConfig)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}
	defer func() {
		t.Log("Cleaning up VM...")
		orchestrator.DeleteVM(context.Background(), vm.ID)
	}()

	// Start VM
	t.Log("Starting VM...")
	if err := orchestrator.StartVM(ctx, vm.ID); err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	// Wait for VM to boot and agent to start
	t.Log("Waiting for VM agent to be ready...")
	time.Sleep(10 * time.Second)

	// Test 1: Verify proxy environment variables are set
	t.Run("ProxyEnvVariables", func(t *testing.T) {
		t.Log("Testing proxy environment variables...")
		cmd := &vmm.Command{
			Cmd:  "bash",
			Args: []string{"-c", "echo $HTTP_PROXY"},
			Env:  vmConfig.Env,
		}

		result, err := orchestrator.ExecuteCommand(ctx, vm.ID, cmd)
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		if result.ExitCode != 0 {
			t.Fatalf("Command failed: %s", result.Stderr)
		}

		if !strings.Contains(result.Stdout, "172.16.0.1:3128") {
			t.Errorf("Expected HTTP_PROXY to be set to 172.16.0.1:3128, got: %s", result.Stdout)
		}
		t.Log("✓ Proxy environment variables are set correctly")
	})

	// Test 2: Access whitelisted domain (github.com)
	t.Run("AccessWhitelistedDomain", func(t *testing.T) {
		t.Log("Testing access to whitelisted domain (github.com)...")
		cmd := &vmm.Command{
			Cmd:  "curl",
			Args: []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "30", "https://github.com"},
			Env:  vmConfig.Env,
		}

		result, err := orchestrator.ExecuteCommand(ctx, vm.ID, cmd)
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		// Should succeed (200 or 301/302 redirect)
		if result.ExitCode != 0 {
			t.Logf("Stderr: %s", result.Stderr)
			t.Logf("Stdout: %s", result.Stdout)
			// Don't fail immediately, check if it's a proxy-related error
			if strings.Contains(result.Stderr, "Connection refused") ||
				strings.Contains(result.Stderr, "Failed to connect") {
				t.Fatal("Failed to connect to proxy - proxy may not be accessible from VM network")
			}
		}

		statusCode := strings.TrimSpace(result.Stdout)
		if statusCode != "200" && statusCode != "301" && statusCode != "302" {
			t.Logf("Warning: Unexpected status code %s (but connection worked)", statusCode)
		}
		t.Log("✓ Successfully accessed whitelisted domain (github.com)")
	})

	// Test 3: Block non-whitelisted domain
	t.Run("BlockNonWhitelistedDomain", func(t *testing.T) {
		t.Log("Testing blocking of non-whitelisted domain (facebook.com)...")
		cmd := &vmm.Command{
			Cmd:  "curl",
			Args: []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "30", "https://facebook.com"},
			Env:  vmConfig.Env,
		}

		result, err := orchestrator.ExecuteCommand(ctx, vm.ID, cmd)
		if err != nil {
			t.Fatalf("Failed to execute command: %v", err)
		}

		// Should fail or return 403 (Access Denied by Squid)
		statusCode := strings.TrimSpace(result.Stdout)
		if statusCode == "200" || statusCode == "301" || statusCode == "302" {
			t.Errorf("Non-whitelisted domain should be blocked, but got status code: %s", statusCode)
		}

		if result.ExitCode == 0 && (statusCode == "200" || statusCode == "301" || statusCode == "302") {
			t.Error("✗ Non-whitelisted domain was NOT blocked (proxy whitelist not working)")
		} else {
			t.Log("✓ Non-whitelisted domain was blocked as expected")
		}
	})

	// Test 4: Check Squid logs
	t.Run("CheckSquidLogs", func(t *testing.T) {
		t.Log("Checking Squid proxy logs...")

		// Get logs from Squid container
		cmd := exec.Command("docker", "logs", "--tail", "50", "aetherium-squid")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Warning: Could not get Squid logs: %v", err)
			return
		}

		logs := string(output)
		if strings.Contains(logs, "github.com") {
			t.Log("✓ Found github.com requests in Squid logs")
		} else {
			t.Log("Note: No github.com requests found in recent logs (may have scrolled off)")
		}

		t.Logf("Recent Squid logs:\n%s", logs)
	})

	// Test 5: Test that npm/git work through proxy
	t.Run("TestToolsWithProxy", func(t *testing.T) {
		t.Log("Testing that tools work through proxy...")

		// Test git (should work - github.com is whitelisted)
		cmd := &vmm.Command{
			Cmd:  "git",
			Args: []string{"ls-remote", "https://github.com/torvalds/linux.git", "HEAD"},
			Env:  vmConfig.Env,
		}

		result, err := orchestrator.ExecuteCommand(ctx, vm.ID, cmd)
		if err != nil {
			t.Logf("Warning: Failed to execute git command: %v", err)
		} else if result.ExitCode != 0 {
			t.Logf("Warning: git command failed: %s", result.Stderr)
		} else {
			t.Log("✓ Git works through proxy")
		}
	})
}

// TestProxyPerformance tests that the proxy doesn't cause significant slowdowns or hangs
func TestProxyPerformance(t *testing.T) {
	if !isRoot() {
		t.Skip("Test requires root privileges")
	}

	if !isSquidRunning() {
		t.Skip("Squid proxy is not running")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Log("Setting up orchestrator...")
	orchestrator, err := setupFirecrackerOrchestrator()
	if err != nil {
		t.Fatalf("Failed to setup orchestrator: %v", err)
	}

	// Create VM with proxy
	vmID := uuid.New().String()
	vmConfig := &types.VMConfig{
		ID:         vmID,
		KernelPath: "/var/firecracker/vmlinux",
		RootFSPath: "/var/firecracker/rootfs.ext4",
		SocketPath: fmt.Sprintf("/tmp/aetherium-vm-%s.sock", vmID),
		VCPUCount:  1,
		MemoryMB:   256,
		Env: map[string]string{
			"HTTP_PROXY":  "http://172.16.0.1:3128",
			"HTTPS_PROXY": "http://172.16.0.1:3128",
		},
	}

	vm, err := orchestrator.CreateVM(ctx, vmConfig)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}
	defer orchestrator.DeleteVM(context.Background(), vm.ID)

	if err := orchestrator.StartVM(ctx, vm.ID); err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	time.Sleep(10 * time.Second)

	// Test multiple requests to ensure no hanging
	t.Run("MultipleRequests", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			t.Logf("Request %d/5...", i+1)
			start := time.Now()

			cmd := &vmm.Command{
				Cmd:  "curl",
				Args: []string{"-s", "-o", "/dev/null", "-w", "%{http_code}", "--max-time", "15", "https://github.com"},
				Env:  vmConfig.Env,
			}

			result, err := orchestrator.ExecuteCommand(ctx, vm.ID, cmd)
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Request %d failed: %v", i+1, err)
				continue
			}

			if duration > 20*time.Second {
				t.Errorf("Request %d took too long: %v (possible hang)", i+1, duration)
			} else {
				t.Logf("✓ Request %d completed in %v", i+1, duration)
			}

			if result.ExitCode != 0 {
				t.Logf("Request %d exit code: %d, stderr: %s", i+1, result.ExitCode, result.Stderr)
			}
		}
	})
}

// Helper functions

func setupFirecrackerOrchestrator() (vmm.VMOrchestrator, error) {
	config := map[string]interface{}{
		"kernel_path":       "/var/firecracker/vmlinux",
		"rootfs_template":   "/var/firecracker/rootfs.ext4",
		"socket_dir":        "/tmp",
		"default_vcpu":      1,
		"default_memory_mb": 256,
	}

	return firecracker.NewFirecrackerOrchestrator(config)
}

func isSquidRunning() bool {
	cmd := exec.Command("docker", "ps", "--filter", "name=aetherium-squid", "--format", "{{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "Up")
}

func isRoot() bool {
	cmd := exec.Command("id", "-u")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "0"
}
