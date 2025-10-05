# Aetherium VM Walkthrough: Provisioning a VM and Cloning a Repository

This guide walks you through provisioning a VM, cloning the veil repository inside it, and executing claude-code on the repository.

## Prerequisites

The network infrastructure is already set up:
- ✅ Bridge `aetherium0` exists (172.16.0.1/24)
- ✅ IP forwarding is enabled
- ✅ NAT is configured
- ✅ Worker binary is compiled with network support

## Current Status

**Blocked:** The worker needs elevated privileges to create TAP network devices for VMs. Without network, VMs cannot:
- Install tools (git, nodejs, bun, claude-code)
- Clone repositories from the internet
- Access external resources

## Solution: Run Worker with Sudo

Since TAP device creation requires `CAP_NET_ADMIN` capability, the worker must run with elevated privileges.

### Option 1: Run Worker with Sudo (Recommended for Development)

```bash
cd /home/techsavvyash/sweatAndBlood/remote-agents/aetherium
sudo ./scripts/start-worker.sh
```

This will start the worker with the necessary permissions to create and manage network devices.

### Option 2: Grant CAP_NET_ADMIN Capability (Production)

```bash
sudo setcap cap_net_admin+ep ./bin/worker
./bin/worker
```

This allows the worker to manage network devices without running as root.

## Complete Walkthrough

Once the worker is running with network support:

### Step 1: Start the Worker

```bash
# In one terminal
sudo ./scripts/start-worker.sh
```

### Step 2: Create a VM

```bash
# In another terminal
./bin/aether-cli -type vm:create -name veil-vm
```

The worker will:
1. Create the VM with Firecracker
2. Create a TAP device and attach it to the bridge
3. Configure the VM with an IP address (172.16.0.x/24)
4. Boot the VM
5. Attempt to install tools: git, nodejs, bun, claude-code

### Step 3: Get the VM ID

```bash
# Query PostgreSQL to get the VM ID
docker exec aetherium-postgres psql -U aetherium -d aetherium -c \
  "SELECT id, name, status FROM vms WHERE name='veil-vm'"
```

Save the VM ID for the next steps.

### Step 4: Clone the Veil Repository

```bash
VM_ID="<your-vm-id-from-step-3>"

./bin/aether-cli -type vm:execute \
  -vm-id "$VM_ID" \
  -cmd git \
  -args "clone,https://github.com/try-veil/veil,/root/veil"
```

This will:
1. Connect to the VM via vsock
2. Execute: `git clone https://github.com/try-veil/veil /root/veil`
3. Wait for the command to complete

### Step 5: Execute Claude Code

```bash
./bin/aether-cli -type vm:execute \
  -vm-id "$VM_ID" \
  -cmd "sh" \
  -args "-c,cd /root/veil && claude-code"
```

Or for an interactive session:

```bash
./bin/aether-cli -type vm:execute \
  -vm-id "$VM_ID" \
  -cmd "bash" \
  -args "-c,cd /root/veil && exec bash"
```

## Monitoring

### View Worker Logs

The worker outputs logs to stderr:
```bash
# If running in background, view logs at:
tail -f /tmp/aetherium-worker.log  # if redirected

# Or check the background process output
```

### View VM Logs

Each VM creates a log file:
```bash
# List VM logs
ls -lh /tmp/aetherium-vm-*.sock.log

# View a specific VM's log
tail -f /tmp/aetherium-vm-<VM-ID>.sock.log
```

### View Network Status

```bash
# Check bridge
ip addr show aetherium0

# Check TAP devices
ip link show | grep aether

# Check NAT rules
sudo iptables -t nat -L POSTROUTING -n -v | grep 172.16.0.0
```

## Troubleshooting

### TAP Device Creation Fails

**Error:** `failed to create TAP device: exit status 1`

**Solution:** Worker needs elevated privileges. Run with:
```bash
sudo ./scripts/start-worker.sh
```

### DNS Resolution Fails Inside VM

**Error:** `Temporary failure resolving 'github.com'`

**Causes:**
1. TAP device not created (worker lacks permissions)
2. NAT not configured
3. IP forwarding not enabled

**Check:**
```bash
# Verify bridge is up
ip addr show aetherium0

# Verify IP forwarding
sysctl net.ipv4.ip_forward

# Verify NAT rule exists
sudo iptables -t nat -L POSTROUTING -n -v
```

### Tool Installation Fails

If git/nodejs/bun/claude-code installation fails, you can:

1. Pre-install in the rootfs template
2. Manually install after VM boots
3. Use vsock to transfer pre-compiled binaries

## Alternative: Pre-created TAP Pool (Advanced)

For running the worker without sudo in production:

```bash
# 1. Create a pool of TAP devices with sudo
sudo ./scripts/create-tap-pool.sh

# 2. Modify the network manager to use pooled TAP devices
# (requires code changes to pkg/network/network.go)

# 3. Run worker without sudo
./bin/worker
```

## Notes

- VMs get IP addresses in the range 172.16.0.2 - 172.16.0.254
- The bridge gateway is at 172.16.0.1
- Each VM gets a unique TAP device (aether-<vm-id-prefix>)
- Network access requires the worker to have CAP_NET_ADMIN capability
