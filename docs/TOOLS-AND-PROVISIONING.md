# VM Tool Installation & Provisioning

Aetherium provides automatic tool installation in VMs to support various development workflows, with special focus on Claude Code support.

## Overview

Every VM is provisioned with:
1. **Default Tools** (installed automatically in all VMs)
2. **Additional Tools** (specified per-request)
3. **Tool Version Management** (specify exact versions)

---

## Default Tools

These tools are installed in **EVERY** VM automatically:

| Tool | Version | Purpose |
|------|---------|---------|
| `git` | latest | Version control |
| `nodejs` | 20.x | JavaScript runtime (required for Claude Code) |
| `bun` | latest | Fast JavaScript runtime |
| `claude-code` | latest | Claude Code CLI |

### Installation Time

Default tools are installed during VM creation. Total time: **~3-5 minutes**

---

## Claude Code Integration

### Automatic Setup

Every VM is pre-configured with:
- Node.js 20.x (required for Claude Code)
- npm (comes with Node.js)
- Claude Code CLI installed globally via npm

### Usage

Once the VM is created, Claude Code is available immediately:

```bash
# Via API
POST /vms/{id}/execute
{
  "command": "claude-code",
  "args": ["--version"]
}

# Via CLI
./bin/aether-cli -type vm:execute -vm-id <uuid> -cmd claude-code -args "--version"
```

### Running Claude Code Tasks

```bash
# Execute Claude Code in a VM
POST /vms/{id}/execute
{
  "command": "claude-code",
  "args": [
    "implement",
    "feature",
    "--file",
    "src/app.js"
  ]
}
```

---

## Additional Tools

Specify additional tools when creating a VM:

### Available Tools

| Tool | Versions | Installation Time |
|------|----------|-------------------|
| `go` | 1.23.0, 1.22.0, latest | ~2 min |
| `python` | 3.11, 3.10, latest | ~3 min |
| `rust` | latest | ~5 min |
| `docker` | latest | ~3 min |

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "full-stack-vm",
    "vcpus": 4,
    "memory_mb": 4096,
    "additional_tools": ["go", "python", "rust"],
    "tool_versions": {
      "go": "1.23.0",
      "python": "3.11"
    }
  }'
```

### Via CLI

```bash
./bin/aether-cli \
  -type vm:create \
  -name full-stack-vm \
  -vcpus 4 \
  -memory 4096 \
  -tools "go,python,rust" \
  -tool-versions "go=1.23.0,python=3.11"
```

### Result

This VM will have:
- **Default**: git, nodejs@20, bun, claude-code
- **Additional**: go@1.23.0, python@3.11, rust@latest

---

## Tool Version Specifications

### Format

```json
{
  "tool_versions": {
    "nodejs": "20",      // Major version
    "go": "1.23.0",      // Specific version
    "python": "3.11",    // Minor version
    "rust": "latest"     // Latest stable
  }
}
```

### Version Resolution

- **Specific version** (e.g., `1.23.0`): Exact version installed
- **Major version** (e.g., `20`): Latest minor/patch in that major
- **Minor version** (e.g., `3.11`): Latest patch in that minor
- **`latest`**: Most recent stable version

---

## Tool Installation Process

### Workflow

1. **VM Creation**
   ```
   Create VM → Start VM → Wait for boot (5s)
   ```

2. **Tool Installation**
   ```
   Install Default Tools → Install Additional Tools → Verify Installation
   ```

3. **Ready**
   ```
   VM Status: Running
   Tools: All installed and verified
   ```

### Installation Methods

#### Node.js
```bash
# Via NodeSource repository
curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
apt-get install -y nodejs
```

#### Bun
```bash
# Official installer
curl -fsSL https://bun.sh/install | bash
export PATH="$HOME/.bun/bin:$PATH"
```

#### Claude Code
```bash
# Via npm (requires Node.js)
npm install -g claude-code
```

#### Go
```bash
# Binary download
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

#### Python
```bash
# Via apt
apt-get install -y python3.11 python3.11-pip python3.11-venv
ln -sf /usr/bin/python3.11 /usr/local/bin/python
```

---

## Rootfs Preparation

### Pre-built Rootfs

For faster VM creation, pre-install tools in the rootfs:

```bash
sudo ./scripts/prepare-rootfs-with-tools.sh
```

This creates a rootfs with:
- Ubuntu 22.04 base
- git, Node.js, Bun, Claude Code pre-installed
- fc-agent service configured

**Benefits:**
- Faster VM creation (tools already present)
- Consistent environment
- Reduced network usage

### Custom Rootfs

```bash
# Set custom paths
export ROOTFS_SIZE=4G
export ROOTFS_PATH=/var/firecracker/custom-rootfs.ext4
export UBUNTU_VERSION=22.04

sudo ./scripts/prepare-rootfs-with-tools.sh
```

---

## Configuration

### Default Tools Config

```yaml
# config/production.yaml
tools:
  default:
    - git
    - nodejs@20
    - bun@latest
    - claude-code@latest

  timeout: 20m

  available:
    - go@1.23.0
    - python@3.11
    - rust@latest
    - docker@latest
```

### Environment Variables

```bash
# Override default tools
AETHERIUM_DEFAULT_TOOLS=git,nodejs,bun,claude-code

# Set installation timeout
AETHERIUM_TOOL_TIMEOUT=30m

# Custom npm registry for Claude Code
NPM_REGISTRY=https://registry.npmjs.org
```

---

## Troubleshooting

### Claude Code Installation Fails

**Problem**: `claude-code --version` returns error

**Solutions**:
1. Verify Node.js is installed:
   ```bash
   POST /vms/{id}/execute
   {"command": "node", "args": ["--version"]}
   ```

2. Manually install Claude Code:
   ```bash
   POST /vms/{id}/execute
   {"command": "npm", "args": ["install", "-g", "claude-code"]}
   ```

3. Check npm permissions:
   ```bash
   POST /vms/{id}/execute
   {"command": "npm", "args": ["config", "get", "prefix"]}
   ```

### Tool Installation Timeout

**Problem**: VM creation takes > 20 minutes

**Solutions**:
1. Use pre-built rootfs with tools
2. Increase timeout in config:
   ```yaml
   tools:
     timeout: 30m
   ```
3. Install tools separately after VM creation:
   ```bash
   # Create VM without tools
   POST /vms
   {"name": "my-vm", "vcpus": 2, "memory_mb": 1024}

   # Install tools manually
   POST /vms/{id}/execute
   {"command": "apt-get", "args": ["install", "-y", "python3"]}
   ```

### Tool Version Mismatch

**Problem**: Wrong version installed

**Check actual version**:
```bash
POST /vms/{id}/execute
{"command": "go", "args": ["version"]}
```

**Specify exact version**:
```json
{
  "tool_versions": {
    "go": "1.23.0"  // Full version, not just "1.23"
  }
}
```

---

## Best Practices

### For Claude Code Workflows

1. **Sufficient Resources**
   ```json
   {
     "vcpus": 2,           // Minimum for Claude Code
     "memory_mb": 2048     // Recommended
   }
   ```

2. **Pre-install Dependencies**
   ```json
   {
     "additional_tools": ["git"],
     "tool_versions": {
       "nodejs": "20",  // LTS version
       "git": "latest"
     }
   }
   ```

3. **Use Pre-built Rootfs**
   - Faster VM creation
   - Consistent Claude Code setup
   - Less network overhead

### For Multi-Language Development

```json
{
  "name": "polyglot-vm",
  "vcpus": 4,
  "memory_mb": 8192,
  "additional_tools": ["go", "python", "rust"],
  "tool_versions": {
    "nodejs": "20",
    "go": "1.23.0",
    "python": "3.11",
    "rust": "latest"
  }
}
```

### For CI/CD Pipelines

```json
{
  "name": "ci-vm",
  "vcpus": 8,
  "memory_mb": 16384,
  "additional_tools": ["docker", "go"],
  "tool_versions": {
    "docker": "latest",
    "go": "1.23.0"
  }
}
```

---

## Monitoring Tool Installation

### Check Installation Status

```bash
# Query VM metadata
GET /vms/{id}

# Response includes installed tools
{
  "metadata": {
    "tools_installed": ["git", "nodejs", "bun", "claude-code", "go"],
    "installation_duration_ms": 180000
  }
}
```

### View Installation Logs

```bash
# Query logs during VM creation
POST /logs/query
{
  "vm_id": "uuid",
  "search_text": "Installing",
  "start_time": <vm_creation_time>
}
```

### Verify Tool Presence

```bash
# Test each tool
POST /vms/{id}/execute
{"command": "which", "args": ["claude-code"]}

POST /vms/{id}/execute
{"command": "node", "args": ["--version"]}
```

---

## Examples

### Create VM for Claude Code Development

```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "claude-dev",
    "vcpus": 2,
    "memory_mb": 2048
  }'

# Default tools (git, nodejs, bun, claude-code) installed automatically
```

### Create VM for Go + Claude Code

```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "go-dev",
    "vcpus": 4,
    "memory_mb": 4096,
    "additional_tools": ["go"],
    "tool_versions": {
      "go": "1.23.0"
    }
  }'

# Installs: git, nodejs, bun, claude-code, go
```

### Full Stack Development VM

```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "fullstack-vm",
    "vcpus": 8,
    "memory_mb": 16384,
    "additional_tools": ["go", "python", "rust", "docker"],
    "tool_versions": {
      "nodejs": "20",
      "go": "1.23.0",
      "python": "3.11",
      "rust": "latest",
      "docker": "latest"
    }
  }'

# Complete development environment with all tools
```
