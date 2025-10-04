# Quickstart: Running Commands in Firecracker VMs

This guide shows you how to start a Firecracker microVM and execute commands in it.

## Prerequisites Setup

### Step 1: Install Firecracker

```bash
# Create directory
sudo mkdir -p /var/firecracker
sudo chown $USER:$USER /var/firecracker

# Download Firecracker v1.7.0
cd /tmp
wget https://github.com/firecracker-microvm/firecracker/releases/download/v1.7.0/firecracker-v1.7.0-x86_64.tgz
tar -xzf firecracker-v1.7.0-x86_64.tgz
sudo mv release-v1.7.0-x86_64/firecracker-v1.7.0-x86_64 /usr/bin/firecracker
sudo chmod +x /usr/bin/firecracker

# Verify
firecracker --version
```

### Step 2: Download Kernel

```bash
wget -O /var/firecracker/vmlinux \
    https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/vmlinux-5.10.217
```

### Step 3: Download Rootfs

```bash
# Download Ubuntu 22.04 rootfs (~200MB)
wget -O /var/firecracker/rootfs.ext4 \
    https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/ubuntu-22.04.ext4
```

### Step 4: Configure KVM Access

```bash
# Check if you're in kvm group
groups | grep kvm

# If not, add yourself
sudo usermod -aG kvm $USER

# Log out and back in for group changes to take effect
```

---

## Method 1: Using Serial Console (Simple)

The simplest way to run commands is via the serial console.

### Implementation

We'll extend the Zig VMM to capture serial console output:

```zig
// internal/firecracker/src/vmm.zig

pub fn readSerialConsole(self: *VM, buffer: []u8) !usize {
    if (self.fc_process) |*proc| {
        if (proc.stdout) |stdout| {
            return try stdout.read(buffer);
        }
    }
    return error.NoConsole;
}
```

### Go Wrapper

```go
// pkg/vmm/firecracker/console.go

func (f *FirecrackerOrchestrator) ExecuteViaConsole(
    ctx context.Context,
    vmID string,
    command string,
) (string, error) {
    // Send command + newline to stdin
    // Read from stdout until prompt
    // Parse output
}
```

---

## Method 2: Using vsock (Advanced)

Firecracker supports virtio-vsock for fast host-guest communication.

### Configuration

Add to VM config:
```json
{
  "vsock": {
    "guest_cid": 3,
    "uds_path": "/tmp/firecracker-vsock.sock"
  }
}
```

### Guest Agent

Inside the VM, run an agent that:
1. Listens on vsock
2. Receives commands
3. Executes them
4. Returns output

---

## Method 3: Using SSH (Production-Ready)

The most robust method for production.

### Setup

1. Configure network in Firecracker:
```json
{
  "network-interfaces": [{
    "iface_id": "eth0",
    "guest_mac": "AA:FC:00:00:00:01",
    "host_dev_name": "tap0"
  }]
}
```

2. Create TAP device:
```bash
sudo ip tuntap add tap0 mode tap
sudo ip addr add 172.16.0.1/24 dev tap0
sudo ip link set tap0 up
```

3. Inside VM, configure SSH and get IP

4. Execute commands via SSH:
```go
func (f *FirecrackerOrchestrator) ExecuteCommand(
    ctx context.Context,
    vmID string,
    cmd *vmm.Command,
) (*vmm.ExecResult, error) {
    // SSH to VM IP
    // Run command
    // Return output
}
```

---

## Quick Test (Without Networking)

For quick testing without network setup, we can use the serial console approach:

### Step 1: Start VM with Console

```bash
# Build our CLI
make build

# Create and start VM
./bin/fc-cli create test-vm
./bin/fc-cli start test-vm
```

### Step 2: Attach to Serial Console

The serial console is available via the Firecracker API socket. We can read/write to it.

### Step 3: Run Commands

Create a simple command executor:

```go
package main

import (
    "bufio"
    "fmt"
    "net"
    "time"
)

func main() {
    // Connect to Firecracker socket
    conn, _ := net.Dial("unix", "/tmp/firecracker-test-vm.sock")
    defer conn.Close()

    // Send command via actions API
    // Or use serial console if configured
}
```

---

## Recommended Approach for Aetherium

For the Aetherium use case (autonomous agents), I recommend:

### Short-term: Serial Console + Init Script

1. Boot VM with custom init script
2. Script executes agent code
3. Captures output to serial console
4. Host reads from console

**Pros**:
- Simple, no networking needed
- Works immediately
- Good for batch jobs

**Cons**:
- One-way communication
- Limited interactivity

### Long-term: vsock + Agent

1. Boot VM with vsock configured
2. Guest agent listens on vsock
3. Host sends commands via vsock
4. Bidirectional, fast communication

**Pros**:
- Fast (no network overhead)
- Secure (no network exposure)
- Bidirectional

**Cons**:
- Requires agent in guest
- More complex setup

---

## Implementation Plan

### Phase 1: Serial Console (This Week)

```zig
// Add to vmm.zig
pub fn getSerialOutput(self: *VM) ![]const u8 {
    // Read from Firecracker stdout
    // Return accumulated output
}
```

```go
// Add to firecracker.go
func (f *FirecrackerOrchestrator) GetVMOutput(
    ctx context.Context,
    vmID string,
) (string, error) {
    // Call Zig function
    // Return output
}
```

### Phase 2: Init Script Injection (Next Week)

1. Modify rootfs to include startup script
2. Script runs agent code
3. Outputs results to serial

### Phase 3: vsock Communication (Week 3)

1. Configure vsock in Firecracker
2. Build guest agent
3. Implement host-guest protocol

---

## Example: Complete Workflow

Once implemented, usage will look like:

```go
package main

import (
    "context"
    "fmt"
    "github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

func main() {
    ctx := context.Background()
    orch, _ := firecracker.NewFirecrackerOrchestrator(config)

    // Create VM
    vm, _ := orch.CreateVM(ctx, vmConfig)

    // Start VM
    orch.StartVM(ctx, vm.ID)

    // Wait for boot
    time.Sleep(5 * time.Second)

    // Execute commands
    result, _ := orch.ExecuteCommand(ctx, vm.ID, &vmm.Command{
        Cmd: "echo",
        Args: []string{"Hello from VM!"},
    })

    fmt.Println("Output:", result.Stdout)
    // Output: Hello from VM!

    result, _ = orch.ExecuteCommand(ctx, vm.ID, &vmm.Command{
        Cmd: "ls",
        Args: []string{"-la", "/"},
    })

    fmt.Println("Filesystem:", result.Stdout)

    // Stop VM
    orch.StopVM(ctx, vm.ID, false)
}
```

---

## Next Steps

1. **Run setup script** (requires sudo):
   ```bash
   ./scripts/setup-firecracker.sh
   ```

2. **Test VM boot**:
   ```bash
   make build
   ./bin/fc-cli create my-vm
   ./bin/fc-cli start my-vm
   ```

3. **Implement serial console reading** (I'll do this next)

4. **Test command execution**

Would you like me to implement the serial console approach now, or would you prefer to set up Firecracker first and I'll implement the full vsock solution?
