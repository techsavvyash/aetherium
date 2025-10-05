# Aetherium - Remote VMM Orchestration Engine

## Project Overview

Aetherium is a remote VMM (Virtual Machine Manager) orchestration platform designed to run isolated code execution environments using either Firecracker microVMs or Docker containers.

**Current Status:** Firecracker orchestrator implemented with official SDK, vsock communication in progress.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│ API Gateway (Go)                                    │
│ - REST/gRPC endpoints                               │
│ - Authentication & rate limiting                    │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│ Task Orchestrator (Go)                              │
│ - Job queuing (Redis/Asynq)                         │
│ - Task distribution                                 │
│ - State management (PostgreSQL)                     │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│ Agent Workers (Go)                                  │
│ - Pull tasks from queue                             │
│ - Execute in isolated VMs/containers                │
│ - Report results                                    │
└─────────────────┬───────────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
┌───────▼────────┐  ┌────────▼─────────┐
│ Firecracker    │  │ Docker           │
│ Orchestrator   │  │ Orchestrator     │
└────────────────┘  └──────────────────┘
```

## Technology Stack

### Core
- **Language:** Go 1.25+
- **VMM:** Firecracker microVMs (official `firecracker-go-sdk`)
- **Container Runtime:** Docker (official Go SDK)
- **Task Queue:** Redis + Asynq
- **State Store:** PostgreSQL
- **Logging:** Loki (Grafana)

### Communication
- **Host ↔ VM:** vsock (virtio-vsock) - direct socket communication
- **Fallback:** TCP (when vsock unavailable)
- **Protocol:** JSON-RPC over vsock/TCP

## Project Structure

```
aetherium/
├── cmd/
│   ├── fc-agent/          # Agent running inside Firecracker VM
│   ├── fc-test/           # Test program for Firecracker orchestrator
│   ├── docker-demo/       # Docker orchestrator demo
│   └── [other services]   # API gateway, orchestrator, worker (TODO)
├── pkg/
│   ├── vmm/
│   │   ├── interface.go           # VMM interface
│   │   ├── firecracker/
│   │   │   ├── firecracker.go     # Firecracker orchestrator (uses official SDK)
│   │   │   ├── exec.go            # Command execution via vsock
│   │   │   └── types.go           # Firecracker-specific types
│   │   └── docker/
│   │       └── docker.go          # Docker orchestrator
│   ├── queue/             # Task queue interface + implementations
│   ├── storage/           # State storage interface + implementations
│   ├── logging/           # Logging interface + implementations
│   ├── events/            # Event bus for integrations
│   ├── integrations/      # Plugin system for GitHub, Slack, etc.
│   └── types/             # Common types
├── config/                # Configuration files
├── scripts/               # Setup and utility scripts
└── docs/                  # Documentation

```

## Key Implementation Details

### 1. Firecracker Orchestrator

**Migration from Zig/CGO to Official SDK:**
- ✅ Removed entire Zig codebase and CGO complexity
- ✅ Using official `github.com/firecracker-microvm/firecracker-go-sdk`
- ✅ Pure Go implementation with proper vsock support

**File:** `pkg/vmm/firecracker/firecracker.go`

```go
import (
    firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
    "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

// VM configuration with vsock device
fcConfig := firecracker.Config{
    SocketPath:      config.SocketPath,
    KernelImagePath: config.KernelPath,
    KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off",
    Drives: []models.Drive{
        {
            DriveID:      firecracker.String("rootfs"),
            PathOnHost:   firecracker.String(config.RootFSPath),
            IsRootDevice: firecracker.Bool(true),
            IsReadOnly:   firecracker.Bool(false),
        },
    },
    MachineCfg: models.MachineConfiguration{
        VcpuCount:  firecracker.Int64(1),
        MemSizeMib: firecracker.Int64(512),
    },
    VsockDevices: []firecracker.VsockDevice{
        {
            Path: config.SocketPath + ".vsock",
            CID:  uint32(3), // Guest CID (host is always 2)
        },
    },
}

machine, err := firecracker.NewMachine(ctx, fcConfig)
```

### 2. Vsock Communication

**Host Side:** `pkg/vmm/firecracker/exec.go`

```go
import "github.com/mdlayher/vsock"

const (
    GuestCID  = 3    // Guest VM's context ID
    AgentPort = 9999 // Port agent listens on
)

func (f *FirecrackerOrchestrator) connectViaVsock(ctx context.Context, timeout time.Duration) (net.Conn, error) {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        conn, err := vsock.Dial(GuestCID, AgentPort, nil)
        if err == nil {
            return conn, nil
        }
        time.Sleep(500 * time.Millisecond)
    }
    return nil, fmt.Errorf("vsock connection timeout")
}
```

**Guest Side:** `cmd/fc-agent/main.go`

```go
import "github.com/mdlayher/vsock"

func createListener(port uint32) (net.Listener, string, error) {
    // Try vsock first
    vsockListener, err := vsock.Listen(port, nil)
    if err == nil {
        return vsockListener, "vsock", nil
    }

    // Fall back to TCP if vsock unavailable in guest kernel
    tcpListener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        return nil, "", fmt.Errorf("both vsock and TCP failed: %w", err)
    }
    return tcpListener, "TCP", nil
}
```

### 3. Command Execution Protocol

**Request (Host → Guest):**
```json
{"cmd":"echo","args":["Hello from Firecracker!"],"env":null}
```

**Response (Guest → Host):**
```json
{"exit_code":0,"stdout":"Hello from Firecracker!\n","stderr":""}
```

### 4. Agent Deployment

Agent is deployed to VM rootfs at `/usr/local/bin/fc-agent` with systemd service:

**File:** `/etc/systemd/system/fc-agent.service` (in rootfs)
```ini
[Unit]
Description=Firecracker Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/fc-agent
Restart=always
RestartSec=3
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

## Setup and Installation

### Prerequisites

```bash
# Install Firecracker and setup rootfs
sudo ./scripts/install-firecracker.sh

# Download kernel with vsock support (v5.10.239)
sudo ./scripts/download-vsock-kernel.sh

# Complete setup and test
sudo ./scripts/setup-and-test.sh
```

### Kernel Requirements

**Host:**
- `vhost-vsock` kernel module loaded
- `/dev/vhost-vsock` device available

**Guest:**
- Kernel with `CONFIG_VIRTIO_VSOCK=y`
- Current: vmlinux-5.10.239 from Firecracker CI (v1.13)
- URL: `https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.13/x86_64/vmlinux-5.10.239`

### Configuration Files

**Development:** `config/config.dev.yaml`
```yaml
vmm:
  type: firecracker
  firecracker:
    binary_path: /usr/bin/firecracker
    kernel_path: /var/firecracker/vmlinux
    rootfs_path: /var/firecracker/rootfs.ext4
    socket_path: /tmp/firecracker.sock
```

## Known Issues & Solutions

### Issue 1: Vsock Connection Timeout

**Symptom:**
```
Cannot connect to VM agent via vsock: vsock connection timeout (guest CID: 3, port: 9999)
```

**Root Cause:**
Guest kernel lacks vsock support (`CONFIG_VIRTIO_VSOCK=y` not compiled in)

**Solutions:**
1. ✅ **Use correct kernel:** Run `sudo ./scripts/download-vsock-kernel.sh`
2. Configure network (TAP device) and use TCP fallback
3. Use Docker orchestrator (has built-in networking)

### Issue 2: Kernel Panic on Boot

**Symptom:**
VM crashes during boot with kernel panic

**Root Cause:**
Kernel v6.1.x incompatible with Ubuntu 22.04 rootfs

**Solution:**
Use kernel v5.10.239 instead (better compatibility)

### Issue 3: Go Not Found in Sudo Context

**Symptom:**
```
sudo: go: command not found
```

**Root Cause:**
Go installed via mise/version manager not in root's PATH

**Solution:**
Scripts now use `sudo -u $USER bash -l -c 'which go'` to find go in user's environment

## Testing

### Test Firecracker Orchestrator

#### Quick Test
```bash
# Build and run test
go build -o bin/fc-test ./cmd/fc-test
./bin/fc-test
```

#### Test with Full Diagnostics
```bash
# Interactive diagnostic test (recommended for debugging)
./scripts/test-and-diagnose.sh
```

#### Manual Diagnostics
```bash
# Check vsock configuration
./scripts/diagnose-vsock.sh

# View VM logs after test
cat /tmp/firecracker-test-vm.sock.log
```

**Expected Output:**
```
=== Firecracker VM Test ===

1. Creating VM...
   ✓ VM created: test-vm

2. Starting VM...
   ✓ VM started

3. Waiting for VM to boot (15s)...
   ✓ Boot complete

4. Testing command execution...
   Exit Code: 0
   Stdout: Hello from Firecracker!

✓ SUCCESS! Command execution works!
```

### Test Docker Orchestrator

```bash
./bin/docker-demo
```

## Build Commands

```bash
# Build all services
make go-build

# Build specific components
go build -o bin/fc-agent ./cmd/fc-agent
go build -o bin/fc-test ./cmd/fc-test
go build -o bin/docker-demo ./cmd/docker-demo
```

## Git Repository

**Remote:** https://github.com/techsavvyash/aetherium

**Initial Commit:** 052ca2b
- Firecracker orchestrator with official SDK
- Vsock support implementation
- Docker orchestrator
- Project structure with Go modules
- Installation and testing scripts

## Roadmap & TODO

### ✅ Completed
1. Firecracker orchestrator using official `firecracker-go-sdk`
2. Vsock support for host-VM communication
3. fc-agent with vsock + TCP fallback
4. Docker orchestrator implementation
5. Project structure and build system
6. Installation scripts

### 🚧 In Progress
- Debugging vsock connection (kernel compatibility issues)

### 📋 Pending

**Infrastructure:**
1. Redis task queue implementation with Asynq
2. PostgreSQL state store with migrations
3. Loki logging backend

**Integration Framework:**
4. Plugin registry and event bus
5. Integration SDK for custom plugins
6. GitHub integration (PR creation, webhook handling)
7. Slack integration (notifications, slash commands)

**Core Services:**
8. API Gateway service (REST/gRPC)
9. Task Orchestrator service
10. Agent Worker service

**Networking:**
11. TAP device configuration for VM networking (vsock alternative)
12. Network isolation and security policies

## Development Notes

### Why Go Instead of Zig?

Initially considered Zig for low-level VMM control, but analysis showed:
- Firecracker is a binary with HTTP API (no need for low-level access)
- Go excels at HTTP, task queuing, orchestration
- Official `firecracker-go-sdk` provides everything needed
- Simpler build process (no CGO)

**Decision:** Pure Go stack using official SDKs

### Why Vsock Over TCP?

**Vsock Advantages:**
- Direct host-VM communication (no network setup)
- Lower latency than bridged networking
- Simpler security model (no IP/port exposure)
- Standard in Firecracker/KVM environments

**TCP Fallback:**
Agent gracefully falls back to TCP when vsock unavailable, but requires network configuration.

### Design Decisions

1. **Pluggable VMM Interface:** Support both Firecracker and Docker
2. **Event-Driven Architecture:** Event bus for integrations
3. **Async Task Queue:** Redis/Asynq for distributed job processing
4. **Structured Logging:** Loki for centralized log aggregation
5. **State Persistence:** PostgreSQL for reliable state management

## Troubleshooting

### Automated Diagnostics

```bash
# Comprehensive vsock diagnostics
./scripts/diagnose-vsock.sh

# Full test with diagnostics on failure
./scripts/test-and-diagnose.sh
```

### Check Vsock Support

```bash
# Host kernel module
lsmod | grep vhost_vsock
ls -l /dev/vhost-vsock

# Guest kernel config (if available)
curl -s "https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.13/x86_64/vmlinux-5.10.239.config" | grep VSOCK
```

### View VM Logs

```bash
# After running a test, check Firecracker logs
cat /tmp/firecracker-test-vm.sock.log

# Look for agent startup
grep -i "fc-agent\|vsock" /tmp/firecracker-test-vm.sock.log

# Check for errors
grep -i "error\|fail\|panic" /tmp/firecracker-test-vm.sock.log
```

### View VM Boot Logs

Boot logs are saved to `/tmp/firecracker-test-vm.sock.log`. Look for:
```
[  OK  ] Started Firecracker Agent
Agent listening on vsock port 9999
```

### Verify Agent Deployment

```bash
sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt
ls -l /mnt/usr/local/bin/fc-agent
cat /mnt/etc/systemd/system/fc-agent.service
sudo umount /mnt
```

### Manual Firecracker Test

```bash
# Start firecracker manually
sudo firecracker --api-sock /tmp/firecracker.sock

# In another terminal, send API commands
curl --unix-socket /tmp/firecracker.sock -i \
  -X PUT 'http://localhost/boot-source' \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -d '{
    "kernel_image_path": "/var/firecracker/vmlinux",
    "boot_args": "console=ttyS0 reboot=k panic=1 pci=off"
  }'
```

## Resources

- **Firecracker Docs:** https://github.com/firecracker-microvm/firecracker/tree/main/docs
- **Firecracker Go SDK:** https://pkg.go.dev/github.com/firecracker-microvm/firecracker-go-sdk
- **Vsock Docs:** https://github.com/firecracker-microvm/firecracker/blob/main/docs/vsock.md
- **Go Vsock Library:** https://pkg.go.dev/github.com/mdlayher/vsock

## Contributing

When adding features:
1. Update this document with architectural changes
2. Add tests for new components
3. Update scripts if setup changes
4. Document any new dependencies

## License

[To be determined]

---

**Last Updated:** 2025-10-04
**Project Status:** Active Development
**Primary Focus:** Vsock communication debugging and core service implementation

## Recent Changes

### 2025-10-04 - Diagnostic Tools
- Added `scripts/diagnose-vsock.sh` for comprehensive vsock diagnostics
- Added `scripts/test-and-diagnose.sh` for interactive testing with failure analysis
- Updated Firecracker orchestrator to enable debug logging
- Improved fc-test to show log file locations and troubleshooting steps
- Increased boot wait time from 15s to 20s for agent startup

### Next Steps for Vsock Debugging
1. Run `sudo ./scripts/setup-and-test.sh` to ensure agent is deployed
2. Run `./scripts/test-and-diagnose.sh` to test with diagnostics
3. Check `/tmp/firecracker-test-vm.sock.log` for VM console output
4. Look for agent startup messages and vsock module loading
5. If vsock fails, check if guest kernel modules are loading properly
