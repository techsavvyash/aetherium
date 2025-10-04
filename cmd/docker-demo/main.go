package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aetherium/aetherium/pkg/types"
	"github.com/aetherium/aetherium/pkg/vmm"
	"github.com/aetherium/aetherium/pkg/vmm/docker"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("Docker VM Command Execution Demo")
	fmt.Println("========================================\n")

	ctx := context.Background()

	// Create Docker orchestrator
	fmt.Println("1. Creating Docker orchestrator...")
	config := map[string]interface{}{
		"network": "bridge",
		"image":   "ubuntu:22.04",
	}

	orch, err := docker.NewDockerOrchestrator(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create orchestrator: %v\n", err)
		os.Exit(1)
	}

	// Check Docker health
	if err := orch.Health(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Docker is not available: %v\n", err)
		fmt.Println("\nPlease ensure Docker is installed and running:")
		fmt.Println("  sudo systemctl start docker")
		os.Exit(1)
	}

	fmt.Println("✅ Docker orchestrator created\n")

	// Create VM
	fmt.Println("2. Creating VM (Docker container)...")
	vmConfig := &types.VMConfig{
		ID:       "demo-container",
		VCPUCount: 2,
		MemoryMB: 512,
	}

	vm, err := orch.CreateVM(ctx, vmConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create VM: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ VM created: %s\n\n", vm.ID)

	// Cleanup on exit
	defer func() {
		fmt.Println("\n7. Cleaning up...")
		if err := orch.DeleteVM(ctx, vm.ID); err != nil {
			fmt.Printf("❌ Failed to cleanup: %v\n", err)
		} else {
			fmt.Println("✅ VM deleted")
		}
	}()

	// Start VM
	fmt.Println("3. Starting VM...")
	if err := orch.StartVM(ctx, vm.ID); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to start VM: %v\n", err)
		return
	}
	fmt.Println("✅ VM started\n")

	// Wait a moment for container to be ready
	time.Sleep(1 * time.Second)

	// Execute commands
	fmt.Println("4. Executing commands in VM...\n")

	commands := []struct {
		name string
		cmd  *vmm.Command
	}{
		{
			name: "Echo Hello",
			cmd: &vmm.Command{
				Cmd:  "echo",
				Args: []string{"Hello from Docker VM!"},
			},
		},
		{
			name: "List root directory",
			cmd: &vmm.Command{
				Cmd:  "ls",
				Args: []string{"-la", "/"},
			},
		},
		{
			name: "Show environment",
			cmd: &vmm.Command{
				Cmd:  "env",
				Args: []string{},
			},
		},
		{
			name: "Current directory",
			cmd: &vmm.Command{
				Cmd:  "pwd",
				Args: []string{},
			},
		},
		{
			name: "Whoami",
			cmd: &vmm.Command{
				Cmd:  "whoami",
				Args: []string{},
			},
		},
	}

	for i, cmdInfo := range commands {
		fmt.Printf("   [%d/%d] %s\n", i+1, len(commands), cmdInfo.name)
		fmt.Printf("   Command: %s %v\n", cmdInfo.cmd.Cmd, cmdInfo.cmd.Args)

		result, err := orch.ExecuteCommand(ctx, vm.ID, cmdInfo.cmd)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n\n", err)
			continue
		}

		fmt.Printf("   Exit Code: %d\n", result.ExitCode)
		if result.Stdout != "" {
			fmt.Printf("   Stdout:\n%s\n", indent(result.Stdout, "      "))
		}
		if result.Stderr != "" {
			fmt.Printf("   Stderr:\n%s\n", indent(result.Stderr, "      "))
		}
		fmt.Println()
	}

	// Get VM status
	fmt.Println("5. Checking VM status...")
	status, err := orch.GetVMStatus(ctx, vm.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to get status: %v\n", err)
		return
	}
	fmt.Printf("✅ VM Status: %s\n\n", status.Status)

	// Stop VM
	fmt.Println("6. Stopping VM...")
	if err := orch.StopVM(ctx, vm.ID, false); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to stop VM: %v\n", err)
		return
	}
	fmt.Println("✅ VM stopped\n")

	fmt.Println("========================================")
	fmt.Println("✅ Demo Complete!")
	fmt.Println("========================================")
	fmt.Println("\nKey Achievements:")
	fmt.Println("  ✓ Created isolated VM (container)")
	fmt.Println("  ✓ Executed multiple commands")
	fmt.Println("  ✓ Captured stdout/stderr")
	fmt.Println("  ✓ Managed VM lifecycle")
	fmt.Println("\nThis same interface works with Firecracker!")
}

func indent(s string, prefix string) string {
	lines := []string{}
	for _, line := range splitLines(s) {
		if line != "" {
			lines = append(lines, prefix+line)
		}
	}
	return join(lines, "\n")
}

func splitLines(s string) []string {
	result := []string{}
	current := ""
	for _, char := range s {
		if char == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func join(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
