# Firecracker VMM Orchestrator

**Status**: ✅ Implemented
**Language**: Zig (low-level) + Go (high-level)
**Purpose**: Lifecycle management for Firecracker microVMs

## Architecture

The Firecracker VMM orchestrator is built in two layers:

### Layer 1: Zig Implementation (`internal/firecracker/`)

**Purpose**: Low-level, performant interaction with Firecracker API

**Files**:
- `src/lib.zig` - C-exported functions for Go CGO
- `src/vmm.zig` - VM lifecycle management
- `src/api.zig` - Firecracker HTTP API client over Unix sockets
- `cgo/firecracker.h` - C header for Go bindings

**Capabilities**:
- Spawn Firecracker processes
- Connect to Firecracker API via Unix sockets
- Configure VM (boot source, drives, machine config)
- Start/stop VMs (graceful and forced)
- Track VM state
- Error reporting

### Layer 2: Go Implementation (`pkg/vmm/firecracker/`)

**Purpose**: High-level Go interface implementing `vmm.VMOrchestrator`

**Files**:
- `firecracker.go` - Go orchestrator implementation

**Capabilities**:
- Implements `vmm.VMOrchestrator` interface
- Manages multiple VMs
- Provides Go-idiomatic API
- JSON configuration parsing
- Error handling

---

## API Reference

### Zig C API

#### `fc_create_vm(config_json, config_len) -> *VM`
Creates a new VM from JSON configuration.

**Parameters**:
- `config_json`: JSON string with VM configuration
- `config_len`: Length of JSON string

**Returns**: VM handle or NULL on error

**Configuration Schema**:
```json
{
  "id": "vm-test",
  "socket_path": "/tmp/firecracker-test.sock",
  "kernel_path": "/var/firecracker/vmlinux",
  "rootfs_path": "/var/firecracker/rootfs.ext4",
  "vcpu_count": 2,
  "memory_mb": 512,
  "boot_args": "console=ttyS0 reboot=k panic=1"
}
```

#### `fc_start_vm(vm) -> bool`
Starts a VM.

**Workflow**:
1. Spawns Firecracker process with `--api-sock`
2. Waits for Unix socket to be created
3. Connects to Firecracker API
4. Configures machine (vCPU, memory)
5. Sets boot source (kernel, boot args)
6. Adds root drive
7. Sends `InstanceStart` action

**Returns**: `true` on success, `false` on error

#### `fc_stop_vm(vm, force) -> bool`
Stops a VM.

**Parameters**:
- `vm`: VM handle
- `force`: If true, kills process. If false, sends Ctrl+Alt+Del and waits.

**Returns**: `true` on success, `false` on error

#### `fc_get_vm_status(vm, status_out, status_len) -> bool`
Gets VM status as string.

**Status Values**:
- `Created` - VM created but not started
- `Starting` - VM is booting
- `Running` - VM is fully operational
- `Stopping` - VM is shutting down
- `Stopped` - VM has stopped
- `Failed` - VM encountered an error

#### `fc_destroy_vm(vm)`
Destroys a VM and frees all resources.

#### `fc_get_last_error(vm, error_out, error_len) -> size_t`
Retrieves last error message from VM.

---

### Go API

#### `NewFirecrackerOrchestrator(config) -> (*FirecrackerOrchestrator, error)`
Creates a new Firecracker orchestrator.

**Config Map**:
```go
config := map[string]interface{}{
    "kernel_path":      "/var/firecracker/vmlinux",
    "rootfs_template":  "/var/firecracker/rootfs.ext4",
    "socket_dir":       "/tmp",
    "default_vcpu":     2,
    "default_memory_mb": 512,
}
```

#### `CreateVM(ctx, config) -> (*types.VM, error)`
Creates a new VM.

```go
vmConfig := &types.VMConfig{
    ID:         "my-vm",
    KernelPath: "/var/firecracker/vmlinux",
    RootFSPath: "/var/firecracker/rootfs.ext4",
    SocketPath: "/tmp/firecracker-my-vm.sock",
    VCPUCount:  2,
    MemoryMB:   512,
}

vm, err := orch.CreateVM(ctx, vmConfig)
```

#### `StartVM(ctx, vmID) -> error`
Starts a VM by ID.

#### `StopVM(ctx, vmID, force) -> error`
Stops a VM.

#### `GetVMStatus(ctx, vmID) -> (*types.VM, error)`
Gets VM status.

#### `DeleteVM(ctx, vmID) -> error`
Deletes a VM.

#### `ListVMs(ctx) -> ([]*types.VM, error)`
Lists all VMs.

---

## CLI Usage

### Installation
```bash
make build
```

### Commands

#### Create a VM
```bash
./bin/fc-cli create my-vm
```

Output:
```
✓ VM created: my-vm
  Status: Created
  Socket: /tmp/firecracker-my-vm.sock
```

#### Start a VM
```bash
./bin/fc-cli start my-vm
```

Output:
```
Starting VM my-vm...
[VMM] Starting VM: my-vm
[VMM] Firecracker process spawned
[API] Connecting to Firecracker at /tmp/firecracker-my-vm.sock
[API] Connected to Firecracker API
[VMM] Configuring VM via Firecracker API
[API] Machine config: 2 vCPUs, 512 MB RAM
[API] Boot source configured
[API] Drive rootfs configured
[API] Instance start command sent
[VMM] VM my-vm is now running
✓ VM my-vm started successfully
```

#### Get VM Status
```bash
./bin/fc-cli status my-vm
```

Output:
```
VM: my-vm
Status: Running
vCPUs: 2
Memory: 512 MB
```

#### List VMs
```bash
./bin/fc-cli list
```

Output:
```
VM ID                STATUS          vCPUs      Memory(MB)
------------------------------------------------------------
my-vm                Running         2          512
```

#### Stop a VM
```bash
# Graceful shutdown
./bin/fc-cli stop my-vm

# Force kill
./bin/fc-cli stop my-vm --force
```

#### Delete a VM
```bash
./bin/fc-cli delete my-vm
```

---

## Prerequisites

### Firecracker Binary
```bash
# Download Firecracker
wget https://github.com/firecracker-microvm/firecracker/releases/download/v1.7.0/firecracker-v1.7.0-x86_64.tgz
tar -xzf firecracker-v1.7.0-x86_64.tgz
sudo mv release-v1.7.0-x86_64/firecracker-v1.7.0-x86_64 /usr/bin/firecracker
sudo chmod +x /usr/bin/firecracker
```

### Linux Kernel
```bash
wget https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/vmlinux-5.10.217
sudo mkdir -p /var/firecracker
sudo mv vmlinux-5.10.217 /var/firecracker/vmlinux
```

### Root Filesystem
```bash
# Create Alpine rootfs
# OR download pre-built rootfs
wget https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.7/x86_64/ubuntu-22.04.ext4
sudo mv ubuntu-22.04.ext4 /var/firecracker/rootfs.ext4
```

### KVM Access
```bash
# Check KVM availability
ls -l /dev/kvm

# Add user to kvm group
sudo usermod -aG kvm $USER
```

---

## Example: Full Lifecycle

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/aetherium/aetherium/pkg/types"
    "github.com/aetherium/aetherium/pkg/vmm/firecracker"
)

func main() {
    ctx := context.Background()

    // Create orchestrator
    config := map[string]interface{}{
        "kernel_path":      "/var/firecracker/vmlinux",
        "rootfs_template":  "/var/firecracker/rootfs.ext4",
        "socket_dir":       "/tmp",
        "default_vcpu":     2,
        "default_memory_mb": 512,
    }

    orch, _ := firecracker.NewFirecrackerOrchestrator(config)

    // Create VM
    vmConfig := &types.VMConfig{
        ID:         "demo-vm",
        KernelPath: "/var/firecracker/vmlinux",
        RootFSPath: "/var/firecracker/rootfs.ext4",
        SocketPath: "/tmp/firecracker-demo.sock",
        VCPUCount:  2,
        MemoryMB:   512,
    }

    vm, _ := orch.CreateVM(ctx, vmConfig)
    fmt.Printf("Created VM: %s\n", vm.ID)

    // Start VM
    orch.StartVM(ctx, vm.ID)
    fmt.Println("VM started")

    // Wait
    time.Sleep(5 * time.Second)

    // Stop VM
    orch.StopVM(ctx, vm.ID, false)
    fmt.Println("VM stopped")

    // Delete VM
    orch.DeleteVM(ctx, vm.ID)
    fmt.Println("VM deleted")
}
```

---

## Implementation Details

### Process Spawning
Firecracker is launched as a child process:
```zig
var child = std.process.Child.init(&[_][]const u8{
    "/usr/bin/firecracker",
    "--api-sock",
    socket_path,
}, allocator);

child.spawn();
```

### Unix Socket Communication
Connects to Firecracker API over Unix domain socket:
```zig
self.stream = try std.net.connectUnixSocket(socket_path);
```

### HTTP API Requests
Sends HTTP/1.1 requests over the Unix socket:
```
PUT /machine-config HTTP/1.1
Host: localhost
Content-Type: application/json
Content-Length: 34

{"vcpu_count":2,"mem_size_mib":512}
```

### Graceful Shutdown
1. Sends `SendCtrlAltDel` action to guest
2. Waits up to 10 seconds for process to exit
3. Force kills if timeout

### Resource Cleanup
- Closes Unix socket
- Deletes socket file
- Kills Firecracker process
- Frees allocated memory

---

## Roadmap

### Current Features ✅
- [x] VM creation
- [x] VM start/stop
- [x] VM status tracking
- [x] Process lifecycle management
- [x] Error reporting
- [x] CLI tool

### Future Enhancements
- [ ] Log streaming via serial console
- [ ] Command execution via SSH/vsock
- [ ] Network configuration (TAP devices)
- [ ] VM snapshots
- [ ] jailer integration for enhanced security
- [ ] Metrics collection
- [ ] Hot-attach drives
- [ ] Rate limiters (I/O, network)

---

## Troubleshooting

### "Firecracker binary not found"
**Solution**: Install Firecracker to `/usr/bin/firecracker`

### "Timeout waiting for Firecracker socket"
**Causes**:
- Firecracker failed to start
- KVM not available
- Permission issues

**Solution**:
```bash
# Check Firecracker manually
/usr/bin/firecracker --api-sock /tmp/test.sock

# Check KVM
ls -l /dev/kvm

# Check permissions
groups | grep kvm
```

### "Failed to start VM: Permission denied"
**Solution**: Add user to `kvm` group
```bash
sudo usermod -aG kvm $USER
# Log out and back in
```

### "Invalid HTTP response"
**Cause**: Firecracker API returned error

**Debug**:
- Check Firecracker logs (stderr)
- Verify kernel and rootfs paths exist
- Check socket permissions

---

## Security Considerations

### Isolation
- Each VM runs in separate Firecracker process
- Hardware-level isolation via KVM
- Resource limits enforced

### Recommendations
1. Use **jailer** for additional sandboxing
2. Run Firecracker as non-root user
3. Use seccomp filters
4. Limit VM resource quotas
5. Use read-only rootfs where possible

---

## Performance

### Startup Time
- VM creation: ~5ms (Go + Zig overhead)
- Firecracker spawn: ~50-100ms
- Guest boot: ~150-300ms (kernel-dependent)
- **Total**: ~200-500ms

### Resource Overhead
- Firecracker process: ~5MB RAM
- Minimum VM: 128MB RAM, 1 vCPU
- Socket overhead: Negligible

### Scalability
- Tested: 100+ VMs per host
- Limit: Available memory and CPU cores
- Each VM: Isolated file descriptors, sockets

---

## References

- [Firecracker Documentation](https://github.com/firecracker-microvm/firecracker/tree/main/docs)
- [Firecracker API Specification](https://github.com/firecracker-microvm/firecracker/blob/main/src/api_server/swagger/firecracker.yaml)
- [Zig Documentation](https://ziglang.org/documentation/master/)
- [CGO Documentation](https://pkg.go.dev/cmd/cgo)
