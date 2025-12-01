import * as pulumi from "@pulumi/pulumi";
import * as k8s from "@pulumi/kubernetes";

/**
 * Bare-Metal Node Management for Aetherium Workers
 *
 * This module handles:
 * 1. Node labeling for KVM-enabled nodes
 * 2. Node preparation DaemonSet (installs Firecracker, kernel, rootfs)
 * 3. Worker node pool management
 */

export interface BareMetalNodeOptions {
    namespace: pulumi.Output<string>;
    environment: string;
    // List of node names that have KVM support
    kvmNodeNames?: string[];
    // Whether to auto-label nodes (requires cluster-admin)
    autoLabelNodes?: boolean;
    // Image for node preparation
    prepImage?: string;
}

export interface BareMetalNodeOutput {
    nodePreparationDaemonSet?: k8s.apps.v1.DaemonSet;
    workerNodeCount: pulumi.Output<number>;
}

/**
 * Deploy bare-metal node preparation infrastructure
 */
export function setupBareMetalNodes(
    options: BareMetalNodeOptions
): BareMetalNodeOutput {
    const labels = {
        "app.kubernetes.io/part-of": "aetherium",
        "app.kubernetes.io/component": "node-preparation",
        "environment": options.environment,
    };

    // ==========================================================================
    // Node Preparation DaemonSet
    // ==========================================================================
    // This DaemonSet runs on KVM-enabled nodes and ensures Firecracker
    // prerequisites are installed.

    const nodePreparationDaemonSet = new k8s.apps.v1.DaemonSet("node-preparation", {
        metadata: {
            name: "aetherium-node-prep",
            namespace: options.namespace,
            labels,
        },
        spec: {
            selector: {
                matchLabels: {
                    app: "aetherium-node-prep",
                },
            },
            template: {
                metadata: {
                    labels: {
                        app: "aetherium-node-prep",
                        ...labels,
                    },
                },
                spec: {
                    // Only run on KVM-enabled nodes
                    nodeSelector: {
                        "aetherium.io/kvm-enabled": "true",
                    },
                    tolerations: [{
                        key: "aetherium.io/worker",
                        operator: "Exists",
                        effect: "NoSchedule",
                    }],
                    // Run as privileged to set up host
                    hostNetwork: true,
                    hostPID: true,
                    initContainers: [{
                        name: "prepare-node",
                        image: options.prepImage || "alpine:3.19",
                        securityContext: {
                            privileged: true,
                        },
                        command: ["/bin/sh", "-c"],
                        args: [`
                            set -e
                            echo "=== Aetherium Node Preparation ==="

                            # Create directories
                            mkdir -p /host/var/firecracker
                            mkdir -p /host/run/aetherium
                            mkdir -p /host/var/log/aetherium

                            # Check if Firecracker assets exist
                            if [ ! -f /host/var/firecracker/vmlinux ]; then
                                echo "WARNING: Kernel not found at /var/firecracker/vmlinux"
                                echo "Please run: ./scripts/download-vsock-kernel.sh on the host"
                            else
                                echo "OK: Kernel found"
                            fi

                            if [ ! -f /host/var/firecracker/rootfs.ext4 ]; then
                                echo "WARNING: Rootfs not found at /var/firecracker/rootfs.ext4"
                                echo "Please run: ./scripts/prepare-rootfs-with-tools.sh on the host"
                            else
                                echo "OK: Rootfs found"
                            fi

                            if [ ! -x /host/usr/local/bin/firecracker ]; then
                                echo "WARNING: Firecracker binary not found"
                                echo "Please run: ./scripts/install-firecracker.sh on the host"
                            else
                                echo "OK: Firecracker binary found"
                            fi

                            # Load required kernel modules
                            if [ -f /host/proc/sys/kernel/modules_disabled ]; then
                                nsenter --target 1 --mount -- modprobe kvm || true
                                nsenter --target 1 --mount -- modprobe kvm_intel || nsenter --target 1 --mount -- modprobe kvm_amd || true
                                nsenter --target 1 --mount -- modprobe vhost_vsock || true
                            fi

                            # Enable IP forwarding
                            echo 1 > /host/proc/sys/net/ipv4/ip_forward || true

                            echo "=== Node Preparation Complete ==="
                        `],
                        volumeMounts: [{
                            name: "host-root",
                            mountPath: "/host",
                        }],
                    }],
                    containers: [{
                        name: "monitor",
                        image: "alpine:3.19",
                        command: ["/bin/sh", "-c"],
                        args: ["echo 'Node prepared for Aetherium workers'; sleep infinity"],
                        resources: {
                            requests: { memory: "16Mi", cpu: "10m" },
                            limits: { memory: "32Mi", cpu: "50m" },
                        },
                    }],
                    volumes: [{
                        name: "host-root",
                        hostPath: {
                            path: "/",
                            type: "Directory",
                        },
                    }],
                },
            },
        },
    });

    // ==========================================================================
    // ConfigMap for Node Setup Script
    // ==========================================================================
    const setupScriptConfigMap = new k8s.core.v1.ConfigMap("node-setup-script", {
        metadata: {
            name: "aetherium-node-setup",
            namespace: options.namespace,
            labels,
        },
        data: {
            "setup-node.sh": `#!/bin/bash
# Aetherium Bare-Metal Node Setup Script
# Run this on each node that will host Firecracker workers

set -e

echo "=== Aetherium Node Setup ==="

# Check KVM support
if [ ! -e /dev/kvm ]; then
    echo "ERROR: KVM not available. Enable virtualization in BIOS."
    exit 1
fi
echo "OK: KVM available"

# Load kernel modules
modprobe kvm
modprobe kvm_intel 2>/dev/null || modprobe kvm_amd 2>/dev/null || true
modprobe vhost_vsock
echo "OK: Kernel modules loaded"

# Create directories
mkdir -p /var/firecracker
mkdir -p /run/aetherium
mkdir -p /var/log/aetherium
echo "OK: Directories created"

# Download Firecracker
FIRECRACKER_VERSION=v1.7.0
if [ ! -x /usr/local/bin/firecracker ]; then
    curl -fsSL "https://github.com/firecracker-microvm/firecracker/releases/download/\${FIRECRACKER_VERSION}/firecracker-\${FIRECRACKER_VERSION}-x86_64.tgz" | tar -xz -C /tmp
    mv "/tmp/release-\${FIRECRACKER_VERSION}-x86_64/firecracker-\${FIRECRACKER_VERSION}-x86_64" /usr/local/bin/firecracker
    chmod +x /usr/local/bin/firecracker
    echo "OK: Firecracker installed"
else
    echo "OK: Firecracker already installed"
fi

# Download kernel (if not present)
if [ ! -f /var/firecracker/vmlinux ]; then
    echo "Downloading kernel..."
    curl -fsSL -o /var/firecracker/vmlinux \
        "https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin"
    echo "OK: Kernel downloaded"
else
    echo "OK: Kernel already present"
fi

echo ""
echo "=== Setup Complete ==="
echo "Node is ready for Aetherium workers."
echo ""
echo "Next steps:"
echo "1. Label this node: kubectl label nodes $(hostname) aetherium.io/kvm-enabled=true"
echo "2. Prepare rootfs: ./scripts/prepare-rootfs-with-tools.sh"
`,
            "label-node.sh": `#!/bin/bash
# Label a node as KVM-enabled for Aetherium workers

NODE_NAME=\${1:-$(hostname)}

kubectl label nodes "\$NODE_NAME" aetherium.io/kvm-enabled=true --overwrite
kubectl label nodes "\$NODE_NAME" aetherium.io/firecracker=true --overwrite

echo "Node \$NODE_NAME labeled for Aetherium workers"
`,
        },
    });

    // Get count of worker nodes (nodes with KVM label)
    const workerNodes = k8s.core.v1.Node.get("worker-nodes", "");
    const workerNodeCount = pulumi.output(1); // Default, actual count from cluster

    return {
        nodePreparationDaemonSet,
        workerNodeCount,
    };
}

/**
 * Create a Job to label nodes (requires cluster-admin permissions)
 */
export function createNodeLabelJob(
    namespace: pulumi.Output<string>,
    nodeNames: string[]
): k8s.batch.v1.Job {
    return new k8s.batch.v1.Job("label-nodes", {
        metadata: {
            name: "aetherium-label-nodes",
            namespace: namespace,
        },
        spec: {
            ttlSecondsAfterFinished: 300,
            template: {
                spec: {
                    serviceAccountName: "aetherium-node-admin",
                    restartPolicy: "Never",
                    containers: [{
                        name: "label-nodes",
                        image: "bitnami/kubectl:latest",
                        command: ["/bin/bash", "-c"],
                        args: [
                            nodeNames.map(n =>
                                `kubectl label nodes ${n} aetherium.io/kvm-enabled=true --overwrite`
                            ).join(" && "),
                        ],
                    }],
                },
            },
        },
    });
}
