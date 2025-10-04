# VM CLI - Interactive MicroVM Manager

A step-by-step CLI tool to manage VMs and execute commands.

## Quick Start

### Option 1: Automated Demo (Recommended)

Run the full interactive demo that walks you through each step:

```bash
./bin/vm-cli demo
```

This will:
1. Initialize Docker orchestrator
2. Create a VM
3. Start the VM
4. Execute multiple commands
5. Show VM status
6. Stop and cleanup

**Press Enter after each step to continue.**

---

### Option 2: Manual Step-by-Step

#### Step 1: Initialize Orchestrator

```bash
# Using Docker (easiest, works immediately)
./bin/vm-cli init docker

# OR using Firecracker (requires setup)
./bin/vm-cli init firecracker
```

**Output:**
```
Initializing docker orchestrator...
✓ docker orchestrator initialized successfully
```

#### Step 2: Create a VM

```bash
./bin/vm-cli create my-vm
```

**Output:**
```
Creating VM: my-vm
✓ VM created: my-vm
  Status: CREATED
  Container: my-vm
```

#### Step 3: Start the VM

```bash
./bin/vm-cli start my-vm
```

**Output:**
```
Starting VM: my-vm
✓ VM my-vm started
```

#### Step 4: Execute Commands

```bash
# Echo command
./bin/vm-cli exec my-vm echo "Hello from VM!"

# List files
./bin/vm-cli exec my-vm ls -la /

# Show current directory
./bin/vm-cli exec my-vm pwd

# Show username
./bin/vm-cli exec my-vm whoami

# Show environment
./bin/vm-cli exec my-vm env

# Run shell script
./bin/vm-cli exec my-vm bash -c "cd /tmp && echo test > file.txt && cat file.txt"
```

**Example Output:**
```
Executing in my-vm: echo Hello from VM!

─── Output ───
Hello from VM!

─── Exit Code: 0 ───
```

#### Step 5: Check VM Status

```bash
./bin/vm-cli status my-vm
```

**Output:**
```
╔════════════════════════════════════════╗
║ VM Status: my-vm                       ║
╠════════════════════════════════════════╣
║ Status:     RUNNING                    ║
║ vCPUs:      2                          ║
║ Memory:     512 MB                     ║
║ Created:    2025-10-04 11:30:15        ║
║ Started:    2025-10-04 11:30:16        ║
╚════════════════════════════════════════╝
```

#### Step 6: List All VMs

```bash
./bin/vm-cli list
```

**Output:**
```
╔════════════════════════════════════════════════════════════════╗
║                         VM List                                ║
╠════════════════════════════════════════════════════════════════╣
║ VM ID                STATUS          vCPUs      Memory(MB) ║
╠════════════════════════════════════════════════════════════════╣
║ my-vm                RUNNING         2          512        ║
╚════════════════════════════════════════════════════════════════╝
Total: 1 VM(s)
```

#### Step 7: Stop the VM

```bash
# Graceful stop
./bin/vm-cli stop my-vm

# Force stop
./bin/vm-cli stop my-vm --force
```

**Output:**
```
Stopping VM: my-vm
✓ VM my-vm stopped
```

#### Step 8: Delete the VM

```bash
./bin/vm-cli delete my-vm
```

**Output:**
```
Deleting VM: my-vm
✓ VM my-vm deleted
```

---

## Complete Command Reference

### Initialization

```bash
vm-cli init <docker|firecracker>
```

Initializes the orchestrator. Must be run first.

**Examples:**
```bash
vm-cli init docker         # Use Docker (easiest)
vm-cli init firecracker    # Use Firecracker (requires setup)
```

---

### VM Management

#### Create VM

```bash
vm-cli create [vm-id]
```

Creates a new VM. If vm-id is omitted, uses "my-vm".

**Examples:**
```bash
vm-cli create              # Creates "my-vm"
vm-cli create test-vm      # Creates "test-vm"
vm-cli create agent-1      # Creates "agent-1"
```

#### Start VM

```bash
vm-cli start <vm-id>
```

Starts a created VM.

**Examples:**
```bash
vm-cli start my-vm
vm-cli start test-vm
```

#### Stop VM

```bash
vm-cli stop <vm-id> [--force]
```

Stops a running VM.

**Examples:**
```bash
vm-cli stop my-vm          # Graceful shutdown
vm-cli stop my-vm --force  # Force kill
```

#### Delete VM

```bash
vm-cli delete <vm-id>
```

Deletes a VM and cleans up resources.

**Examples:**
```bash
vm-cli delete my-vm
vm-cli delete test-vm
```

---

### Command Execution

#### Execute Command

```bash
vm-cli exec <vm-id> <command> [args...]
```

Executes a command in the VM and shows output.

**Examples:**
```bash
# Simple commands
vm-cli exec my-vm echo "Hello"
vm-cli exec my-vm pwd
vm-cli exec my-vm whoami
vm-cli exec my-vm hostname

# Commands with arguments
vm-cli exec my-vm ls -la /
vm-cli exec my-vm ls -la /home
vm-cli exec my-vm cat /etc/os-release

# Environment commands
vm-cli exec my-vm env
vm-cli exec my-vm printenv PATH

# Shell scripts
vm-cli exec my-vm bash -c "echo 'Multi-line script'; pwd; whoami"
vm-cli exec my-vm bash -c "cd /tmp && ls -la"
vm-cli exec my-vm sh -c "for i in 1 2 3; do echo \$i; done"

# File operations
vm-cli exec my-vm bash -c "echo 'test data' > /tmp/file.txt && cat /tmp/file.txt"
vm-cli exec my-vm bash -c "mkdir -p /tmp/mydir && ls -la /tmp"

# System info
vm-cli exec my-vm uname -a
vm-cli exec my-vm cat /proc/cpuinfo
vm-cli exec my-vm free -h
vm-cli exec my-vm df -h
```

---

### Information Commands

#### Status

```bash
vm-cli status <vm-id>
```

Shows detailed VM information.

**Example:**
```bash
vm-cli status my-vm
```

#### List

```bash
vm-cli list
```

Lists all VMs.

**Example:**
```bash
vm-cli list
```

---

### Utility Commands

#### Demo

```bash
vm-cli demo
```

Runs an interactive demo that walks through all features.

#### Help

```bash
vm-cli help
```

Shows usage information.

---

## Example Workflows

### Workflow 1: Quick Test

```bash
# Initialize
./bin/vm-cli init docker

# Create and start
./bin/vm-cli create test-vm
./bin/vm-cli start test-vm

# Run command
./bin/vm-cli exec test-vm echo "It works!"

# Cleanup
./bin/vm-cli delete test-vm
```

### Workflow 2: File Operations

```bash
./bin/vm-cli init docker
./bin/vm-cli create filetest
./bin/vm-cli start filetest

# Create file
./bin/vm-cli exec filetest bash -c "echo 'Hello World' > /tmp/hello.txt"

# Read file
./bin/vm-cli exec filetest cat /tmp/hello.txt

# List directory
./bin/vm-cli exec filetest ls -la /tmp

# Cleanup
./bin/vm-cli delete filetest
```

### Workflow 3: System Information

```bash
./bin/vm-cli init docker
./bin/vm-cli create sysinfo
./bin/vm-cli start sysinfo

# Get OS info
./bin/vm-cli exec sysinfo cat /etc/os-release

# Get kernel version
./bin/vm-cli exec sysinfo uname -r

# Get CPU info
./bin/vm-cli exec sysinfo bash -c "cat /proc/cpuinfo | grep 'model name' | head -1"

# Get memory info
./bin/vm-cli exec sysinfo free -h

# Cleanup
./bin/vm-cli delete sysinfo
```

### Workflow 4: Multiple VMs

```bash
./bin/vm-cli init docker

# Create multiple VMs
./bin/vm-cli create vm1
./bin/vm-cli create vm2
./bin/vm-cli create vm3

# Start all
./bin/vm-cli start vm1
./bin/vm-cli start vm2
./bin/vm-cli start vm3

# List them
./bin/vm-cli list

# Execute in each
./bin/vm-cli exec vm1 echo "I am VM 1"
./bin/vm-cli exec vm2 echo "I am VM 2"
./bin/vm-cli exec vm3 echo "I am VM 3"

# Cleanup all
./bin/vm-cli delete vm1
./bin/vm-cli delete vm2
./bin/vm-cli delete vm3
```

---

## Troubleshooting

### "Orchestrator not initialized"

**Problem:** You forgot to run `init`.

**Solution:**
```bash
./bin/vm-cli init docker
```

### "Docker is not available"

**Problem:** Docker is not running.

**Solution:**
```bash
sudo systemctl start docker
sudo usermod -aG docker $USER
# Log out and back in
```

### "VM not found"

**Problem:** VM doesn't exist or was deleted.

**Solution:**
```bash
# List VMs to see what exists
./bin/vm-cli list

# Create the VM
./bin/vm-cli create my-vm
```

### "Container name already in use"

**Problem:** Previous VM with same name wasn't cleaned up.

**Solution:**
```bash
docker rm -f <vm-name>
# OR
./bin/vm-cli delete <vm-name>
```

---

## Important: CLI Design Pattern

**The vm-cli is designed for demo and testing purposes.** The recommended way to use it is via the **demo mode**:

```bash
./bin/vm-cli demo
```

Individual commands (`init`, `create`, `start`, etc.) run as separate processes and don't persist state between invocations. For production use cases, consider:

1. **Use the docker-demo binary** for programmatic workflows:
   ```bash
   ./bin/docker-demo
   ```

2. **Integrate the Go packages directly** in your application:
   ```go
   import "github.com/aetherium/aetherium/pkg/vmm/docker"
   ```

3. **Build a stateful daemon** that persists the orchestrator (future enhancement)

---

## Tips & Tricks

### 1. Use Demo Mode for Testing

The demo mode runs all steps in one process:
```bash
./bin/vm-cli demo
```

### 2. Use Descriptive VM Names

```bash
./bin/vm-cli create agent-task-123
./bin/vm-cli create test-environment
./bin/vm-cli create pr-builder-456
```

### 3. Use Bash for Complex Commands

Instead of:
```bash
./bin/vm-cli exec my-vm echo hello
./bin/vm-cli exec my-vm cd /tmp  # Won't work as expected
./bin/vm-cli exec my-vm ls
```

Do:
```bash
./bin/vm-cli exec my-vm bash -c "echo hello && cd /tmp && ls"
```

### 4. Capture Output in Scripts

```bash
#!/bin/bash
OUTPUT=$(./bin/vm-cli exec my-vm whoami)
echo "VM is running as: $OUTPUT"
```

---

## What's Next?

Now that you can execute commands in VMs, you can:

1. **Run autonomous agents**
   - Create VM
   - Execute agent code
   - Capture results

2. **Build CI/CD pipelines**
   - Spin up clean VMs
   - Run tests
   - Tear down

3. **Parallel task execution**
   - Create multiple VMs
   - Run tasks concurrently
   - Aggregate results

4. **Switch to Firecracker**
   - Better isolation
   - Faster startup
   - Same CLI!

---

**Built:** 2025-10-04
**Status:** ✅ Fully Functional
