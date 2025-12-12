# Tool Installation and Provisioning

## Default Tools

All VMs automatically install:
- git
- nodejs (v20)
- bun (latest)
- claude-code (latest)

These are installed automatically during VM creation.

## Installation Method

Tools are installed via **mise** (version manager):
- Supports version pinning (e.g., `go@1.23.0`)
- Fast installation from prebuilt binaries
- Installed inside the VM via `ExecuteCommand`

## Additional Tools

Specify extra tools when creating VM:

**Via CLI:**
```bash
./bin/aether-cli -type vm:create -name my-vm \
  -tools "go,python,rust" \
  -tool-versions "go:1.23.0,python:3.11"
```

**Via REST API:**
```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-vm",
    "vcpus": 2,
    "memory_mb": 2048,
    "additional_tools": ["go", "python", "rust"],
    "tool_versions": {"go": "1.23.0", "python": "3.11"}
  }'
```

## Supported Tools

Common tools with version support:
- go
- python
- rust
- node (different version than default)
- bun (different version than default)
- docker
- kubectl
- helm
- terraform
- And more...

## Version Pinning

Specify exact versions:

```bash
./bin/aether-cli -type vm:create \
  -tools "go,python" \
  -tool-versions "go:1.23.0,python:3.11.2"
```

Format: `tool:version` in tool_versions map

## Installation Timeline

VM creation with tools:
1. VM boots (30s)
2. Default tools installed (2-4 min)
3. Additional tools installed (1-2 min each)

Total: ~5-10 minutes depending on tools

## Pre-Built Rootfs

To skip tool installation for each VM, use pre-built rootfs:

```bash
# One-time setup (5-10 minutes)
sudo ./scripts/prepare-rootfs-with-tools.sh
```

This creates `/var/firecracker/rootfs.ext4` with all tools pre-installed.

Result:
- VM creation: ~30 seconds (no tool installation)
- All default tools available immediately
- Consistent tool versions across VMs

## Installation Process

When creating a VM:

1. Create VM container
2. Start VM
3. Boot kernel (30s)
4. For each tool:
   - Execute: `mise install tool@version`
   - Wait for installation
   - Verify: `tool --version`
5. Store VM metadata in PostgreSQL

## Checking Tool Availability

```bash
# Check what's installed
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd mise -args "list"

# Check specific tool
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd which -args "go"

# Check version
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd go -args "version"
```

## Troubleshooting

**Tool not found:**
- Use pre-built rootfs: `sudo ./scripts/prepare-rootfs-with-tools.sh`
- Or manually install: `./bin/aether-cli -type vm:execute -vm-id $VM_ID -cmd apt-get -args "install,-y,tool-name"`

**Installation timeout:**
- Use pre-built rootfs (much faster)
- Increase timeout in config
- Install tools manually after VM creation

**Wrong version installed:**
- Delete VM, create new with correct version
- Or manually install: `./bin/aether-cli -type vm:execute -vm-id $VM_ID -cmd mise -args "install,tool@version"`

## Implementation Details

**Location:** `services/core/pkg/tools/installer.go`

The installer:
1. Connects to VM via vsock
2. Sends install command for each tool
3. Waits for completion with timeout (20 minutes default)
4. Verifies installation
5. Reports results

Timeout configuration:
```yaml
tools:
  timeout: 20m  # Can be customized
```

## Mise Integration

Mise provides:
- Fast installation from prebuilt binaries
- Version management
- Multiple tool support
- Environment variable setup

Inside VM, tools use mise shims for version management.
