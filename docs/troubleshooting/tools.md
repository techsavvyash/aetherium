# Tool Installation Issues

## Tool Installation Timeout

**Error:**
```
tool installation timeout after 20m
```

**Cause:** Installing tools via package manager takes too long

**Solution 1 (Recommended):**
```bash
sudo ./scripts/prepare-rootfs-with-tools.sh
```

Pre-installs all default tools (git, nodejs, bun, claude-code) in rootfs.

**Solution 2:**
Increase timeout in config or command:
```bash
# Check worker config
cat config/config.yaml | grep tool_timeout

# Or set environment variable
export TOOL_INSTALL_TIMEOUT=30m
./bin/worker
```

## Tool Not Found After Installation

**Error:**
```
command not found: git
# or
command not found: node
```

**Check:**
```bash
# Inside VM, verify tool is installed
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd which -args "git"

# Check PATH
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd echo -args "\$PATH"
```

**Solution:**
1. Create new VM with pre-built rootfs:
```bash
sudo ./scripts/prepare-rootfs-with-tools.sh
./bin/aether-cli -type vm:create -name new-vm
```

2. Or manually install tool:
```bash
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd apt-get -args "install,-y,git"
```

## Node.js/npm Not Found

**Check:**
```bash
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd node -args "--version"
```

**Solution:**
```bash
# Install Node.js
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd apt-get -args "install,-y,nodejs,npm"

# Or use pre-built rootfs
sudo ./scripts/prepare-rootfs-with-tools.sh
```

## Bun Not Found

**Check:**
```bash
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd bash -args "-c,~/.bun/bin/bun --version"
```

**Solution:**
```bash
# Install Bun (requires Node.js first)
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd bash -args "-c,curl -fsSL https://bun.sh/install | bash"

# Or use pre-built rootfs
sudo ./scripts/prepare-rootfs-with-tools.sh
```

## Claude Code Not Found

**Check:**
```bash
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd claude-code -args "--version"
```

**Solution:**
```bash
# Install via npm
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd npm -args "install,-g,claude-code"

# Or use pre-built rootfs (includes it)
sudo ./scripts/prepare-rootfs-with-tools.sh
```

## Package Manager Errors

**Error:**
```
Unable to locate package git
# or
E: Could not get lock /var/lib/apt/lists/lock
```

**Solution:**
```bash
# Update package lists
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd apt-get -args "update"

# Try install again
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd apt-get -args "install,-y,git"
```

## Additional Tools Installation

To install tools beyond the defaults, specify when creating VM:

```bash
./bin/aether-cli -type vm:create -name my-vm \
  -tools "go,python,rust" \
  -tool-versions "go:1.23.0,python:3.11"
```

Or via REST API:
```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-vm",
    "additional_tools": ["go", "python", "rust"],
    "tool_versions": {"go": "1.23.0", "python": "3.11"}
  }'
```

## Checking Tool Status

```bash
# Check what tools are available
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd mise -args "list"

# Check specific tool
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd which -args "python"

# Check version
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd python -args "--version"
```
