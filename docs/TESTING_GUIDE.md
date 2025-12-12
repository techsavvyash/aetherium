# Aetherium Testing Guide

Comprehensive guide for testing Aetherium from setup through end-to-end validation.

## Table of Contents
- [Pre-Execution Checklist](#pre-execution-checklist)
- [Setup Verification](#setup-verification)
- [Running Tests](#running-tests)
- [Expected Outputs](#expected-outputs)
- [Verification Queries](#verification-queries)
- [Troubleshooting](#troubleshooting)

---

## Pre-Execution Checklist

Before running any tests, verify your environment:

### System Requirements
- [ ] Operating System: Linux
- [ ] Go version: 1.21+
- [ ] CPU with KVM support (for Firecracker)
- [ ] At least 4GB available memory
- [ ] At least 10GB disk space (for rootfs)
- [ ] Docker installed and running
- [ ] Network access

### Project Setup
- [ ] Repository cloned: `/home/techsavvyash/sweatAndBlood/remote-agents/aetherium`
- [ ] Go dependencies available: `go mod download`
- [ ] Binaries can be built: `make build`
- [ ] No permission issues on /var, /tmp, or /home

### Infrastructure Requirements
- [ ] PostgreSQL available: `docker ps | grep aetherium-postgres`
- [ ] Redis available: `docker ps | grep aetherium-redis`
- [ ] PostgreSQL connectivity: `docker exec aetherium-postgres pg_isready`
- [ ] Redis connectivity: `docker exec aetherium-redis redis-cli ping`
- [ ] Database migrations applied: `psql -U aetherium -d aetherium -c "\dt"`

### Firecracker Requirements
- [ ] Firecracker binary available: `which firecracker` or `/var/firecracker/firecracker`
- [ ] Kernel image: `/var/firecracker/vmlinux` (5.10+, vsock support)
- [ ] Rootfs template: `/var/firecracker/rootfs.ext4` (2GB)
- [ ] KVM available: `ls -l /dev/kvm`
- [ ] vhost-vsock available: `ls -l /dev/vhost-vsock`

### Network Requirements
- [ ] Bridge exists: `ip addr show aetherium0`
- [ ] Bridge has correct IP: `172.16.0.1/24`
- [ ] NAT rules configured: `sudo iptables -t nat -L | grep 172.16`
- [ ] IP forwarding enabled: `sysctl net.ipv4.ip_forward` (should be 1)

### File System
- [ ] All scripts executable: `ls -l scripts/*.sh`
- [ ] Binaries built: `ls -l bin/api-gateway bin/worker bin/aether-cli`
- [ ] Source code readable: `find pkg -name "*.go" | wc -l` (should be 50+)

### Cleanup
- [ ] Old VMs removed from database (optional)
- [ ] Old tasks removed from Redis queue (optional)
- [ ] Previous test logs archived (optional)

---

## Setup Verification

Run these commands to verify your system is ready:

### 1. Verify Docker Services

```bash
# Check services running
docker ps | grep -E "(postgres|redis)"

# Expected output:
# CONTAINER ID  IMAGE                    ... NAMES
# xxxxxxxx      postgres:15-alpine       ... aetherium-postgres
# xxxxxxxx      redis:7-alpine           ... aetherium-redis
```

### 2. Verify Database

```bash
# Check PostgreSQL connectivity
docker exec aetherium-postgres pg_isready

# Expected output: accepting connections

# Check tables exist
docker exec aetherium-postgres psql -U aetherium -d aetherium -c "\dt"

# Expected output: ~8 tables (vms, tasks, executions, etc.)
```

### 3. Verify Redis

```bash
# Check Redis connectivity
docker exec aetherium-redis redis-cli ping

# Expected output: PONG

# Check Redis is empty (or verify existing data)
docker exec aetherium-redis redis-cli DBSIZE

# Expected output: (integer) 0 or (integer) N
```

### 4. Verify Firecracker

```bash
# Check kernel
ls -lh /var/firecracker/vmlinux

# Check rootfs
ls -lh /var/firecracker/rootfs.ext4

# Check KVM
ls -l /dev/kvm

# Check vsock
ls -l /dev/vhost-vsock
```

### 5. Verify Network

```bash
# Check bridge
ip addr show aetherium0

# Expected output includes: inet 172.16.0.1/24

# Check NAT rules
sudo iptables -t nat -L | grep 172.16

# Check IP forwarding
sysctl net.ipv4.ip_forward

# Expected output: net.ipv4.ip_forward = 1
```

### 6. Verify Binaries

```bash
# Check binaries built
ls -lh bin/api-gateway bin/worker bin/aether-cli bin/fc-agent

# Try to get versions/help
./bin/api-gateway --help 2>&1 | head -5
./bin/worker --help 2>&1 | head -5
./bin/aether-cli --help 2>&1 | head -5
```

### Full Verification Script

```bash
#!/bin/bash
set -e

echo "=== Aetherium System Verification ==="
echo

echo "✓ Docker services:"
docker ps | grep -E "(postgres|redis)" || echo "❌ Missing services"

echo
echo "✓ PostgreSQL:"
docker exec aetherium-postgres pg_isready || echo "❌ PostgreSQL not responding"

echo
echo "✓ Redis:"
docker exec aetherium-redis redis-cli ping || echo "❌ Redis not responding"

echo
echo "✓ Firecracker:"
ls -lh /var/firecracker/vmlinux || echo "❌ Kernel missing"
ls -lh /var/firecracker/rootfs.ext4 || echo "❌ Rootfs missing"

echo
echo "✓ Network:"
ip addr show aetherium0 | grep "inet 172" || echo "❌ Bridge not configured"

echo
echo "✓ Binaries:"
ls -lh bin/api-gateway bin/worker bin/aether-cli || echo "❌ Binaries not built"

echo
echo "=== All systems ready! ==="
```

---

## Running Tests

### Unit Tests

```bash
# Run all unit tests
make test

# Run specific package tests
go test ./pkg/vmm/...
go test ./pkg/queue/...
go test ./pkg/storage/...

# Run with verbose output
go test -v ./pkg/...

# Run with coverage
make test-coverage
```

### Integration Tests

```bash
# Navigate to integration tests
cd tests/integration

# Run integration tests (requires sudo)
sudo go test -v -timeout 30m

# Or use the script
cd /path/to/aetherium
sudo ./scripts/run-integration-test.sh
```

### Manual End-to-End Test

**Terminal 1: Start API Gateway**
```bash
cd /path/to/aetherium
./bin/api-gateway
```

Expected output:
```
2025-12-12 10:15:32 INFO Starting API Gateway on :8080
2025-12-12 10:15:32 INFO Initialized task service
2025-12-12 10:15:32 INFO API Gateway ready
```

**Terminal 2: Start Worker**
```bash
cd /path/to/aetherium
sudo -E env \
  POSTGRES_HOST=localhost \
  POSTGRES_PORT=5432 \
  POSTGRES_USER=aetherium \
  POSTGRES_PASSWORD=aetherium \
  POSTGRES_DB=aetherium \
  REDIS_ADDR=localhost:6379 \
  KERNEL_PATH=/var/firecracker/vmlinux \
  ROOTFS_TEMPLATE=/var/firecracker/rootfs.ext4 \
  ./bin/worker
```

Expected output:
```
2025-12-12 10:15:35 INFO Starting worker daemon
2025-12-12 10:15:35 INFO Registered handlers: vm:create, vm:execute, vm:delete
2025-12-12 10:15:35 INFO Worker ready, listening for tasks
```

**Terminal 3: Run Test Script**
```bash
cd /path/to/aetherium

# Create test VM
./bin/aether-cli -type vm:create -name test-vm-e2e -vcpus 2 -memory 1024

# Wait for VM to be created (~30 seconds)
sleep 40

# List VMs to get ID
VM_ID=$(./bin/aether-cli -type vm:list | grep test-vm-e2e | awk '{print $1}')

# Execute command
./bin/aether-cli -type vm:execute -vm-id "$VM_ID" -cmd bun -args "--version"

# Wait for execution
sleep 5

# Delete VM
./bin/aether-cli -type vm:delete -vm-id "$VM_ID"

# Verify results in database
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT name, status FROM vms WHERE name = 'test-vm-e2e' ORDER BY created_at DESC LIMIT 1;"
```

---

## Expected Outputs

### API Gateway Startup

```
INFO Starting API Gateway
INFO Registering routes
INFO POST /api/v1/vms
INFO GET /api/v1/vms
INFO GET /api/v1/vms/:id
INFO DELETE /api/v1/vms/:id
INFO POST /api/v1/vms/:id/execute
INFO GET /api/v1/vms/:id/executions
INFO GET /health
INFO API Gateway listening on :8080
```

### Worker Startup

```
INFO Starting worker daemon
INFO Connecting to PostgreSQL: localhost:5432/aetherium
INFO Connecting to Redis: localhost:6379
INFO Registering task handlers
INFO Handler registered: vm:create
INFO Handler registered: vm:execute
INFO Handler registered: vm:delete
INFO Worker ready, processing tasks
```

### VM Creation

**CLI Output:**
```
Creating VM: test-vm-e2e
Task ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Status: pending
```

**Worker Logs:**
```
INFO Task received: vm:create
INFO Creating VM: test-vm-e2e
INFO Creating TAP device: aether-xxxxxxxx
INFO Attaching TAP to bridge: aetherium0
INFO Starting Firecracker process
INFO Waiting for VM boot...
INFO VM booted successfully
INFO Installing tools...
INFO Installed: git, nodejs, bun, claude-code
INFO VM ready: test-vm-e2e (IP: 172.16.0.2)
INFO Task completed successfully
```

### Command Execution

**CLI Output:**
```
Executing command in VM
Task ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Status: pending
```

**Worker Logs:**
```
INFO Task received: vm:execute
INFO Executing command in VM: bun --version
INFO Connecting to VM via vsock...
INFO Command output: bun 1.x.x
INFO Command exited: 0
INFO Task completed successfully
```

### VM Deletion

**CLI Output:**
```
Deleting VM: test-vm-e2e
Task ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Status: pending
```

**Worker Logs:**
```
INFO Task received: vm:delete
INFO Stopping VM: test-vm-e2e
INFO VM stopped successfully
INFO Deleting TAP device: aether-xxxxxxxx
INFO Deleting VM from database
INFO Task completed successfully
```

---

## Verification Queries

### Check VMs Created

```bash
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT id, name, status, vcpu_count, memory_mb, created_at FROM vms ORDER BY created_at DESC LIMIT 5;"
```

Expected output:
```
                   id                   |      name       | status |  vcpu_count | memory_mb |         created_at
----------------------------------------+-----------------+--------+-------------+-----------+----------------------------
 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx   | test-vm-e2e     | running |           2 |      1024 | 2025-12-12 10:15:40.123+00
```

### Check Executions

```bash
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT vm_id, command, exit_code, started_at FROM executions ORDER BY started_at DESC LIMIT 5;"
```

Expected output:
```
                vm_id                 | command | exit_code |      started_at
---------------------------------------+---------+-----------+------------------------
 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx | bun     |         0 | 2025-12-12 10:15:45.123
```

### Check Tasks

```bash
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT id, type, status, created_at FROM tasks ORDER BY created_at DESC LIMIT 5;"
```

Expected output:
```
                   id                   |  type   | status |         created_at
----------------------------------------+---------+---------+------------------------
 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx   | vm:delete   | completed | 2025-12-12 10:15:50.123
 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx   | vm:execute  | completed | 2025-12-12 10:15:45.123
 xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx   | vm:create   | completed | 2025-12-12 10:15:40.123
```

### Full VM Details

```bash
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT * FROM vms WHERE name = 'test-vm-e2e' \gx"
```

This shows all columns in expanded format (one per line).

---

## Troubleshooting

### API Gateway Won't Start

**Error:** `Address already in use`

**Solution:**
```bash
# Kill existing process
lsof -i :8080 | grep LISTEN | awk '{print $2}' | xargs kill -9

# Or use different port
API_PORT=8081 ./bin/api-gateway
```

### Worker Won't Start

**Error:** `Operation not permitted`

**Solution:** Worker needs network privileges:
```bash
# Use sudo
sudo ./bin/worker

# Or grant capability
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

### PostgreSQL Connection Failed

**Error:** `FATAL: database "aetherium" does not exist`

**Solution:**
```bash
# Check database exists
docker exec aetherium-postgres psql -U postgres -l | grep aetherium

# If missing, run migrations
cd migrations
migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" -path . up
```

### Redis Queue Not Processing

**Error:** Tasks stay in "pending" state

**Solution:**
```bash
# Check Redis has tasks
docker exec aetherium-redis redis-cli LLEN asynq:queues:default

# Check worker is running
ps aux | grep worker

# Check worker logs
tail -50 worker.log
```

### VM Creation Times Out

**Error:** VM stays in "creating" state after 2+ minutes

**Solution:**
```bash
# Check if Firecracker process exists
ps aux | grep firecracker

# Check if TAP device created
ip link show | grep aether-

# Check worker logs for errors
tail -100 worker.log | grep -i error

# Manually delete stuck VM from database
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "DELETE FROM vms WHERE name = 'stuck-vm';"
```

### vsock Connection Failed

**Error:** `Cannot connect to VM agent via vsock`

**Solution:**
```bash
# Check kernel has vsock
file /var/firecracker/vmlinux | grep -i "vsock"

# Check module loaded in VM (from inside VM)
lsmod | grep virtio_vsock

# SSH into VM and check fc-agent
ssh -i /path/to/key root@172.16.0.2
ps aux | grep fc-agent
```

---

## Test Success Criteria

A successful test run should show:

✅ **API Gateway:**
- Starts without errors
- Responds to HTTP requests
- Logs show listening on :8080

✅ **Worker:**
- Starts without errors (with sudo)
- Registers all task handlers
- Processes tasks from queue

✅ **VM Creation:**
- Task transitions from "pending" → "running"
- VM status is "running"
- Tools are installed (bun, nodejs, git)
- Network is configured (IP: 172.16.0.x)

✅ **Command Execution:**
- Command runs in VM
- Output is captured
- Exit code is recorded
- Task completes successfully

✅ **Database:**
- VM record exists in `vms` table
- Execution record exists in `executions` table
- All fields are populated correctly

✅ **Final State:**
- VM deleted successfully
- Cleanup complete
- No orphaned processes

---

## Performance Benchmarks

| Operation | Time | Notes |
|-----------|------|-------|
| VM Creation | 30-45s | Including tool installation |
| VM Boot | 5-10s | Time from start to ready |
| Tool Installation | 20-35s | git, nodejs, bun, claude-code |
| Command Execution | <1s | Local vsock communication |
| Task Queue Latency | <500ms | From enqueue to start |
| VM Deletion | 2-5s | Cleanup and resource release |

---

## Next Steps

1. ✅ Run pre-execution checklist
2. ✅ Run setup verification
3. ✅ Run manual E2E test
4. ✅ Verify database state
5. 📖 Read integration test guide
6. 🚀 Run integration tests

---

**Last Updated:** December 12, 2025  
**Status:** Ready for testing
