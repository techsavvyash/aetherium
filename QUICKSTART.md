# Aetherium Quick Start

## First Time Setup (2 commands)

```bash
sudo ./scripts/setup-network.sh       # Setup network bridge & NAT
sudo ./scripts/setup-rootfs-once.sh   # Fix rootfs (apt, DNS, auto-config)
```

## Daily Usage

```bash
# Terminal 1: Start worker
sudo ./scripts/start-worker.sh

# Terminal 2: Use VMs
./bin/aether-cli -type vm:create -name my-vm
```

## Complete Example: Clone Repo & Run Claude Code

```bash
# 1. Create VM
./bin/aether-cli -type vm:create -name veil-vm

# 2. Get VM ID
VM_ID=$(docker exec aetherium-postgres psql -U aetherium -d aetherium -t -c \
  "SELECT id FROM vms WHERE name='veil-vm' ORDER BY created_at DESC LIMIT 1" | tr -d ' ')

# 3. Clone veil repository
./bin/aether-cli -type vm:execute -vm-id "$VM_ID" \
  -cmd git -args "clone,https://github.com/try-veil/veil,/root/veil"

# 4. Run claude-code
./bin/aether-cli -type vm:execute -vm-id "$VM_ID" \
  -cmd sh -args "-c,cd /root/veil && claude-code"
```

## What Gets Fixed Automatically?

After running `setup-rootfs-once.sh`, every VM automatically has:
- ✅ Working apt-get (all dpkg directories created)
- ✅ DNS resolution (8.8.8.8, 8.8.4.4, 1.1.1.1)
- ✅ Internet connectivity (via NAT)
- ✅ Tool installation (git, nodejs, bun, claude-code)

No manual intervention needed!

## Troubleshooting

**Error: "List directory /var/lib/apt/lists/partial is missing"**
→ Run: `sudo ./scripts/setup-rootfs-once.sh`

**Error: "failed to create TAP device"**
→ Worker needs sudo: `sudo ./scripts/start-worker.sh`

**Error: "bridge aetherium0 does not exist"**
→ Run: `sudo ./scripts/setup-network.sh`
