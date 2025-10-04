package main

import (
	"context"
	"fmt"
	"os"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   Firecracker Official SDK Demo       ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()

	// Check prerequisites
	if _, err := os.Stat("/var/firecracker/vmlinux"); os.IsNotExist(err) {
		fmt.Println("✗ Kernel not found at /var/firecracker/vmlinux")
		fmt.Println("  Run: sudo ./scripts/install-firecracker.sh")
		os.Exit(1)
	}

	if _, err := os.Stat("/var/firecracker/rootfs.ext4"); os.IsNotExist(err) {
		fmt.Println("✗ Rootfs not found at /var/firecracker/rootfs.ext4")
		os.Exit(1)
	}

	fmt.Println("✓ Prerequisites met")
	fmt.Println()

	ctx := context.Background()

	// Configure Firecracker VM using official SDK
	vcpuCount := int64(1)
	memSizeMib := int64(256)

	cfg := firecracker.Config{
		SocketPath:      "/tmp/firecracker-sdk-demo.sock",
		KernelImagePath: "/var/firecracker/vmlinux",
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off",
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String("/var/firecracker/rootfs.ext4"),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(vcpuCount),
			MemSizeMib: firecracker.Int64(memSizeMib),
		},
		// Optional: Add vsock device for communication
		// VsockDevices: []firecracker.VsockDevice{
		// 	{
		// 		Path: "/tmp/firecracker-sdk-demo-vsock.sock",
		// 		CID:  3,
		// 	},
		// },
	}

	fmt.Println("Creating VM with official SDK...")
	fmt.Printf("  vCPUs: %d\n", vcpuCount)
	fmt.Printf("  Memory: %d MB\n", memSizeMib)
	fmt.Println()

	// Create the machine
	machine, err := firecracker.NewMachine(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to create machine: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting VM...")

	// Start the VM
	if err := machine.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "✗ Failed to start VM: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ VM started successfully!")
	fmt.Println()

	fmt.Println("VM is now running. Press Ctrl+C to stop.")
	fmt.Println()
	fmt.Println("In another terminal, you can see:")
	fmt.Println("  ps aux | grep firecracker")
	fmt.Println("  lsof /dev/kvm")
	fmt.Println("  ls -la /tmp/firecracker-sdk-demo.sock")
	fmt.Println()

	// Wait for interrupt
	machine.Wait(ctx)

	fmt.Println("\nStopping VM...")
	if err := machine.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: shutdown error: %v\n", err)
	}

	fmt.Println("✓ VM stopped")
}
