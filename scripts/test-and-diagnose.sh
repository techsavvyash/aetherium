#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$SCRIPT_DIR/.."

echo "╔═══════════════════════════════════════════════════════╗"
echo "║  Firecracker Test with Diagnostics                   ║"
echo "╚═══════════════════════════════════════════════════════╝"
echo ""

# Step 1: Run diagnostics
echo "Step 1: Running vsock diagnostics..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
"$SCRIPT_DIR/diagnose-vsock.sh"
echo ""
echo "Press Enter to continue with VM test..."
read

# Step 2: Clean up old logs and sockets
echo "Step 2: Cleaning up old test artifacts..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
rm -f /tmp/firecracker-test-vm.sock*
echo "✓ Cleaned up"
echo ""

# Step 3: Run the test
echo "Step 3: Running Firecracker test..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cd "$PROJECT_DIR"

if ./bin/fc-test; then
    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║  ✓ TEST PASSED!                                       ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    exit 0
else
    echo ""
    echo "╔═══════════════════════════════════════════════════════╗"
    echo "║  ✗ TEST FAILED - Running Diagnostics                  ║"
    echo "╚═══════════════════════════════════════════════════════╝"
    echo ""

    # Step 4: Check VM logs
    echo "Step 4: Checking VM logs..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    LOG_FILE="/tmp/firecracker-test-vm.sock.log"

    if [ -f "$LOG_FILE" ]; then
        echo "Last 50 lines of VM log:"
        echo "════════════════════════════════════════════════════"
        tail -50 "$LOG_FILE"
        echo "════════════════════════════════════════════════════"
        echo ""

        # Look for specific issues
        if grep -q "fc-agent" "$LOG_FILE"; then
            echo "✓ Agent mentioned in logs"
        else
            echo "✗ No mention of fc-agent in logs - agent may not be installed"
        fi

        if grep -qi "vsock" "$LOG_FILE"; then
            echo "✓ Vsock mentioned in logs"
            echo "  Vsock-related lines:"
            grep -i "vsock" "$LOG_FILE" | tail -5 | sed 's/^/    /'
        else
            echo "○ No vsock mentions in logs"
        fi

        if grep -qi "error\|fail\|panic" "$LOG_FILE"; then
            echo "⚠ Errors found in logs:"
            grep -i "error\|fail\|panic" "$LOG_FILE" | tail -10 | sed 's/^/    /'
        fi
    else
        echo "✗ No log file found at $LOG_FILE"
        echo "  This means Firecracker didn't create logs, which is unusual."
    fi
    echo ""

    # Step 5: Recommendations
    echo "Step 5: Recommendations..."
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "Based on the diagnostics, try these fixes:"
    echo ""
    echo "1. Verify agent is deployed in rootfs:"
    echo "   sudo mount -o loop /var/firecracker/rootfs.ext4 /mnt"
    echo "   ls -lh /mnt/usr/local/bin/fc-agent"
    echo "   cat /mnt/etc/systemd/system/fc-agent.service"
    echo "   sudo umount /mnt"
    echo ""
    echo "2. Re-run full setup:"
    echo "   sudo ./scripts/setup-and-test.sh"
    echo ""
    echo "3. Check complete logs:"
    echo "   cat /tmp/firecracker-test-vm.sock.log | less"
    echo ""

    exit 1
fi
