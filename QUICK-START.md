# Quick Start Guide

Get Aetherium running in 5 minutes with Claude Code support!

## Prerequisites Check

```bash
# Check Go
go version  # Should be 1.23+

# Check Docker
docker --version

# Check if running on Linux with KVM
ls -l /dev/kvm  # Should exist

# Check Firecracker (install if missing)
which firecracker || {
    wget https://github.com/firecracker-microvm/firecracker/releases/download/v1.5.0/firecracker-v1.5.0-x86_64.tgz
    tar xvf firecracker-v1.5.0-x86_64.tgz
    sudo cp release-v1.5.0-x86_64/firecracker-v1.5.0-x86_64 /usr/local/bin/firecracker
    sudo chmod +x /usr/local/bin/firecracker
}

# Load vsock module
sudo modprobe vhost_vsock
```

---

## Step 1: Start Infrastructure (2 minutes)

```bash
# PostgreSQL
docker run -d --name aetherium-postgres \
  -e POSTGRES_USER=aetherium \
  -e POSTGRES_PASSWORD=aetherium \
  -e POSTGRES_DB=aetherium \
  -p 5432:5432 \
  postgres:15-alpine

# Redis
docker run -d --name aetherium-redis \
  -p 6379:6379 \
  redis:7-alpine

# Loki (optional - for logging)
docker run -d --name aetherium-loki \
  -p 3100:3100 \
  grafana/loki:latest

# Wait for services
sleep 5
```

---

## Step 2: Build Aetherium (1 minute)

```bash
cd /home/techsavvyash/sweatAndBlood/remote-agents/aetherium

# Build binaries
go build -o bin/worker ./cmd/worker
go build -o bin/api-gateway ./cmd/api-gateway
go build -o bin/aether-cli ./cmd/aether-cli

# Build fc-agent (for vsock communication)
go build -o bin/fc-agent ./cmd/fc-agent
```

---

## Step 3: Setup Database (30 seconds)

```bash
# Install migrate tool if not present
which migrate || go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -database "postgres://aetherium:aetherium@localhost:5432/aetherium?sslmode=disable" \
        -path ./migrations up
```

---

## Step 4: Prepare Rootfs (2-5 minutes)

**Option A: Quick (use existing rootfs if available)**
```bash
# Check if rootfs exists
ls -lh /var/firecracker/rootfs.ext4

# If exists, ensure fc-agent is deployed
sudo cp bin/fc-agent /var/firecracker/rootfs.ext4
```

**Option B: Full (create new rootfs with all tools)**
```bash
# This installs Ubuntu + git + nodejs + bun + claude-code
sudo ./scripts/prepare-rootfs-with-tools.sh

# This takes ~5-10 minutes but gives you a perfect rootfs
```

**Option C: Minimal (for testing)**
```bash
# Use existing rootfs and install tools on-demand
# Tools will be installed during VM creation (takes longer per VM)
```

---

## Step 5: Start Services (30 seconds)

**Terminal 1 - Worker:**
```bash
sudo ./bin/worker
```

**Terminal 2 - API Gateway:**
```bash
./bin/api-gateway
```

You should see:
```
Worker: ✓ Registered handlers for: vm:create, vm:execute, vm:delete
API:    ✓ API Gateway listening on :8080
```

---

## Step 6: Create Your First VM (3 minutes)

```bash
# Create VM with Claude Code (in a new terminal)
./bin/aether-cli -type vm:create -name my-first-vm

# Output:
# ✓ VM creation task submitted: <task-id>
#   Name: my-first-vm
#   vCPUs: 1
#   Memory: 256MB
#   Default Tools: git, nodejs, bun, claude-code (installed automatically)
```

**Wait ~3-5 minutes** for:
1. VM creation (30s)
2. Default tool installation (2-4 min)

Watch the worker terminal for progress logs.

---

## Step 7: Get VM ID

```bash
# Check database for VM ID
docker exec -it aetherium-postgres psql -U aetherium -c \
  "SELECT id, name, status FROM vms WHERE name='my-first-vm';"

# Copy the ID (UUID)
# Example: 123e4567-e89b-12d3-a456-426614174000
```

Or via API:
```bash
curl http://localhost:8080/api/v1/vms | jq
```

---

## Step 8: Run Claude Code!

```bash
# Set VM ID
VM_ID="<your-vm-id-here>"

# Test Claude Code
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd claude-code -args "--version"

# Test Node.js
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd node -args "--version"

# Test Bun
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd bash -args "-c,~/.bun/bin/bun --version"

# Test git
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd git -args "--version"
```

---

## Step 9: Clone a Repo (original request!)

```bash
# Clone a repository
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd git -args "clone,https://github.com/techsavvyash/vync"

# Wait 30 seconds

# Verify clone
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd ls -args "-la,vync"
```

---

## Step 10: Run Integration Test (original 3 VM test)

```bash
# This will:
# 1. Create 3 VMs
# 2. Clone: vync, veil, web repos
# 3. Verify all clones
# 4. Cleanup

sudo ./scripts/run-integration-test.sh
```

Expected output:
```
=== Step 1: Creating 3 VMs ===
✓ VM created: clone-vync
✓ VM created: clone-veil
✓ VM created: clone-web

=== Step 3: Cloning repositories ===
Submitted clone task for https://github.com/techsavvyash/vync
Submitted clone task for https://github.com/try-veil/veil
Submitted clone task for https://github.com/try-veil/web

=== Test Completed Successfully ===
```

---

## Via REST API

All the same operations via API:

### Create VM
```bash
curl -X POST http://localhost:8080/api/v1/vms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "api-vm",
    "vcpus": 2,
    "memory_mb": 2048,
    "additional_tools": ["go"],
    "tool_versions": {"go": "1.23.0"}
  }'
```

### Execute Command
```bash
VM_ID="<uuid>"

curl -X POST http://localhost:8080/api/v1/vms/$VM_ID/execute \
  -H "Content-Type: application/json" \
  -d '{
    "command": "claude-code",
    "args": ["--version"]
  }'
```

### List VMs
```bash
curl http://localhost:8080/api/v1/vms | jq
```

### Query Logs
```bash
curl -X POST http://localhost:8080/api/v1/logs/query \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "'$VM_ID'",
    "limit": 50
  }' | jq
```

---

## Verify Everything Works

```bash
# 1. Check worker is running
ps aux | grep worker

# 2. Check API is responding
curl http://localhost:8080/api/v1/health

# 3. Check database has VMs
docker exec -it aetherium-postgres psql -U aetherium -c "SELECT COUNT(*) FROM vms;"

# 4. Check Redis has tasks
docker exec -it aetherium-redis redis-cli KEYS "asynq:*"

# 5. View worker logs
journalctl -u aetherium-worker -f  # If using systemd
# or
tail -f /var/log/aetherium/worker.log

# 6. View API logs
journalctl -u aetherium-api-gateway -f  # If using systemd
# or
tail -f /var/log/aetherium/api-gateway.log
```

---

## Common Issues & Fixes

### VM Creation Fails

```bash
# Check KVM access
ls -l /dev/kvm
sudo usermod -aG kvm $USER

# Check vsock module
lsmod | grep vhost_vsock
sudo modprobe vhost_vsock

# Check rootfs
ls -lh /var/firecracker/rootfs.ext4
```

### Tool Installation Timeout

```bash
# Use pre-built rootfs
sudo ./scripts/prepare-rootfs-with-tools.sh

# Or increase timeout in config
echo 'tools:
  timeout: 30m' >> config/production.yaml
```

### Claude Code Not Found

```bash
# Check if Node.js is installed
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd node -args "--version"

# Manually install Claude Code
./bin/aether-cli -type vm:execute -vm-id $VM_ID \
  -cmd npm -args "install,-g,claude-code"
```

---

## What You Have Now

✅ **Worker** processing tasks from queue
✅ **API Gateway** on port 8080
✅ **PostgreSQL** storing VM state
✅ **Redis** managing task queue
✅ **Loki** collecting logs (optional)
✅ **VMs with Claude Code** ready to use
✅ **GitHub/Slack integrations** ready (if configured)

---

## Next Steps

1. **Add More VMs**
   ```bash
   ./bin/aether-cli -type vm:create -name vm2 -vcpus 4 -memory 4096 -tools "go,python"
   ```

2. **Configure Integrations**
   ```bash
   export GITHUB_TOKEN=ghp_xxxxx
   export SLACK_BOT_TOKEN=xoxb-xxxxx
   ./bin/api-gateway
   ```

3. **Deploy to Production**
   See `docs/DEPLOYMENT.md` for Docker/Kubernetes deployment

4. **Build Your Workflow**
   - Create VMs with specific tools
   - Execute Claude Code tasks
   - Integrate with GitHub PRs
   - Get Slack notifications

---

## Documentation

- `IMPLEMENTATION-SUMMARY.md` - Complete feature list
- `docs/API-GATEWAY.md` - API reference
- `docs/TOOLS-AND-PROVISIONING.md` - Tool installation details
- `docs/INTEGRATIONS.md` - GitHub/Slack setup
- `docs/DEPLOYMENT.md` - Production deployment

---

## You're All Set! 🎉

Your Aetherium platform is running with:
- ✅ Claude Code installed in all VMs
- ✅ Tool provisioning system
- ✅ REST API
- ✅ Logging with Loki
- ✅ Integration support

**Test the original requirement:**
```bash
# Create 3 VMs and clone the repos
sudo ./scripts/run-integration-test.sh
```

Enjoy building with Aetherium! 🚀
