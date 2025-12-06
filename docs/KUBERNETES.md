# Aetherium Kubernetes Deployment Guide

This guide covers deploying Aetherium to Kubernetes using Helm charts and Pulumi infrastructure-as-code.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites](#prerequisites)
- [Bare-Metal Node Setup](#bare-metal-node-setup)
- [Auto-Scaling Setup](#auto-scaling-setup)
- [Quick Start](#quick-start)
- [Service Discovery with Consul](#service-discovery-with-consul)
- [Helm Chart](#helm-chart)
- [Pulumi Infrastructure](#pulumi-infrastructure)
- [Configuration](#configuration)
- [Operations](#operations)
- [Troubleshooting](#troubleshooting)

## Architecture Overview

Aetherium uses a **bare-metal Kubernetes architecture** where:

- **Workers run as K8s pods** on bare-metal nodes with direct KVM access
- **Firecracker runs on the host** - the container is a thin wrapper around the Go binary
- **Consul provides service discovery** - workers register themselves for the API Gateway
- **API Gateway is containerized** - stateless, can run anywhere

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Kubernetes Cluster                                │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────────┐ │
│  │                      aetherium namespace                            │ │
│  │                                                                      │ │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐            │ │
│  │  │ API Gateway │    │   Consul    │    │ PostgreSQL  │            │ │
│  │  │ (Deployment)│◄──►│(StatefulSet)│    │(StatefulSet)│            │ │
│  │  │  Replicas:3 │    │  Service    │    │             │            │ │
│  │  └──────┬──────┘    │  Discovery  │    └─────────────┘            │ │
│  │         │           └──────▲──────┘                                │ │
│  │         │                  │                                        │ │
│  │         │           Register/Discover                               │ │
│  │         │                  │                                        │ │
│  └─────────┼──────────────────┼────────────────────────────────────────┘ │
│            │                  │                                          │
│            │    ┌─────────────┴─────────────────────────────────┐       │
│            │    │         Bare-Metal Worker Nodes (KVM)          │       │
│            │    │                                                 │       │
│            │    │  ┌─────────────────────────────────────────┐   │       │
│            │    │  │      DaemonSet: aetherium-worker        │   │       │
│            │    │  │                                          │   │       │
│            ▼    │  │   Node 1          Node 2          Node 3│   │       │
│       ┌────────┐│  │   ┌──────┐       ┌──────┐       ┌──────┐│   │       │
│       │  Task  ││  │   │Worker│       │Worker│       │Worker││   │       │
│       │  Queue ││  │   │ Pod  │       │ Pod  │       │ Pod  ││   │       │
│       │(Redis) ││  │   │      │       │      │       │      ││   │       │
│       └────────┘│  │   │/dev/ │       │/dev/ │       │/dev/ ││   │       │
│                 │  │   │ kvm  │       │ kvm  │       │ kvm  ││   │       │
│                 │  │   └──┬───┘       └──┬───┘       └──┬───┘│   │       │
│                 │  │      │              │              │     │   │       │
│                 │  │   ┌──▼───┐       ┌──▼───┐       ┌──▼───┐│   │       │
│                 │  │   │ VMs  │       │ VMs  │       │ VMs  ││   │       │
│                 │  │   │(Fire-│       │(Fire-│       │(Fire-││   │       │
│                 │  │   │crack)│       │crack)│       │crack)││   │       │
│                 │  │   └──────┘       └──────┘       └──────┘│   │       │
│                 │  └──────────────────────────────────────────┘   │       │
│                 └─────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Workers as K8s Pods**: Managed by DaemonSet, one per KVM-enabled node
2. **Firecracker on Host**: VMs run directly on the host via `/dev/kvm` passthrough
3. **Host Network**: Workers use `hostNetwork: true` for VM networking (TAP/bridge)
4. **Consul Service Discovery**: Workers register with Consul, Gateway discovers them
5. **Privileged Containers**: Required for KVM, TAP devices, and bridge management

## Prerequisites

### Required Tools

```bash
# Kubernetes CLI
kubectl version --client

# Helm 3.x
helm version

# Pulumi (for infrastructure provisioning)
pulumi version

# Docker (for building images)
docker version
```

### Kubernetes Cluster Requirements

- Kubernetes 1.25+
- **Bare-metal nodes** with KVM support (for workers)
- Persistent volume provisioner (for PostgreSQL/Redis/Consul)
- Optional: Ingress controller (nginx-ingress recommended)

## Bare-Metal Node Setup

### Step 1: Verify KVM Support

On each worker node:

```bash
# Check KVM device
ls -la /dev/kvm

# Check vhost-vsock (for VM communication)
ls -la /dev/vhost-vsock

# If missing, load modules
sudo modprobe kvm
sudo modprobe kvm_intel  # or kvm_amd
sudo modprobe vhost_vsock
```

### Step 2: Install Firecracker

```bash
# Download and install Firecracker
FIRECRACKER_VERSION=v1.7.0
curl -fsSL "https://github.com/firecracker-microvm/firecracker/releases/download/${FIRECRACKER_VERSION}/firecracker-${FIRECRACKER_VERSION}-x86_64.tgz" | tar -xz
sudo mv release-${FIRECRACKER_VERSION}-x86_64/firecracker-${FIRECRACKER_VERSION}-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker
```

### Step 3: Prepare Firecracker Assets

```bash
# Create directory
sudo mkdir -p /var/firecracker

# Download kernel with vsock support
sudo curl -fsSL -o /var/firecracker/vmlinux \
  "https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin"

# Prepare rootfs (run from aetherium repo)
sudo ./scripts/prepare-rootfs-with-tools.sh
```

### Step 4: Label Node for Aetherium

```bash
# Label node as KVM-enabled
kubectl label nodes <node-name> aetherium.io/kvm-enabled=true

# Optional: Dedicate node to workers
kubectl taint nodes <node-name> aetherium.io/worker=true:NoSchedule
```

### Step 5: Verify Node Setup

```bash
# Check node labels
kubectl get nodes --show-labels | grep aetherium

# Expected directory structure on node:
/var/firecracker/
├── vmlinux           # Kernel (5.10.x with vsock)
├── rootfs.ext4       # Root filesystem
└── firecracker       # Binary (optional, can be in /usr/local/bin)
```

## Auto-Scaling Setup

For production deployments with auto-scaling, nodes are **automatically provisioned** by the Node Provisioner DaemonSet. No manual setup required!

### How Auto-Scaling Works

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Auto-Scaling Flow                                │
│                                                                          │
│  1. Cluster Autoscaler adds new node                                    │
│              │                                                           │
│              ▼                                                           │
│  2. Node Provisioner DaemonSet detects new node                         │
│              │                                                           │
│              ▼                                                           │
│  3. Init container checks KVM capability                                │
│              │                                                           │
│              ├── No KVM → Skip node (not a worker)                      │
│              │                                                           │
│              ▼ Has KVM                                                   │
│  4. Install Firecracker, download kernel                                │
│              │                                                           │
│              ▼                                                           │
│  5. Label node: aetherium.io/kvm-enabled=true                          │
│              │                                                           │
│              ▼                                                           │
│  6. Worker DaemonSet schedules worker pod                               │
│              │                                                           │
│              ▼                                                           │
│  7. Worker registers with Consul                                        │
│              │                                                           │
│              ▼                                                           │
│  8. API Gateway discovers new worker → Ready for VMs!                   │
└─────────────────────────────────────────────────────────────────────────┘
```

### Option 1: Node Provisioner DaemonSet (Recommended)

Enable the Node Provisioner in Helm values:

```yaml
nodeProvisioner:
  enabled: true
  firecrackerVersion: "v1.7.0"
  kernelUrl: "https://s3.amazonaws.com/spec.ccfc.min/img/quickstart_guide/x86_64/kernels/vmlinux.bin"
  # For rootfs, either:
  # 1. Host your own (recommended for production)
  rootfsUrl: "https://your-bucket.s3.amazonaws.com/aetherium/rootfs.ext4"
  # 2. Or leave empty and build on each node (slower)
  # rootfsUrl: ""
```

The DaemonSet:
- Runs on **all nodes** in the cluster
- Detects KVM capability automatically
- Installs Firecracker and downloads kernel
- Labels KVM-capable nodes for worker scheduling
- Skips non-KVM nodes (cloud control plane, etc.)

### Option 2: Cloud-Init / User Data

For faster node startup, pre-configure nodes via cloud-init:

**AWS EC2 Launch Template:**
```bash
# Use the provided user-data script
cat scripts/cloud-init/worker-node-userdata.yaml
```

**Azure VMSS Custom Data:**
```bash
# Base64 encode the cloud-init script
base64 scripts/cloud-init/worker-node-userdata.yaml
```

**GCP Instance Template:**
```bash
# Use as startup-script metadata
gcloud compute instance-templates create aetherium-worker \
  --metadata-from-file startup-script=scripts/cloud-init/worker-node-userdata.yaml
```

### Option 3: Pre-Baked Machine Images (Fastest)

For production, create custom AMIs/images with everything pre-installed:

```bash
# 1. Build and upload rootfs to cloud storage
sudo ./scripts/build-and-upload-rootfs.sh aws my-bucket

# 2. Create a base VM with cloud-init
# 3. Run the setup script
# 4. Create AMI/image from the VM
# 5. Use in your node pool configuration
```

### Rootfs Distribution

The rootfs (~2GB) contains the base Ubuntu system and tools. Options:

| Method | Pros | Cons |
|--------|------|------|
| Cloud Storage URL | Simple, works everywhere | Download on each node |
| Pre-baked AMI | Fastest startup | Per-region AMIs needed |
| NFS/EFS mount | Single source of truth | Network dependency |
| Build on node | No external deps | Slow (10+ minutes) |

**Recommended: Cloud Storage with CDN**

```bash
# Build and upload rootfs
sudo ./scripts/build-and-upload-rootfs.sh aws my-aetherium-assets

# Update Helm values with the URL
# values.yaml
nodeProvisioner:
  rootfsUrl: "https://my-aetherium-assets.s3.amazonaws.com/aetherium-rootfs-latest.ext4"
```

## Quick Start

### 1. Prepare Worker Nodes

Follow [Bare-Metal Node Setup](#bare-metal-node-setup) on each worker node.

### 2. Build and Push Docker Images

```bash
# Build all images
./scripts/build-images.sh v1.0.0

# Build and push to registry
./scripts/build-images.sh v1.0.0 push
```

### 3. Deploy with Helm

```bash
# Development environment
./scripts/deploy-k8s.sh dev deploy

# Production environment
./scripts/deploy-k8s.sh prod deploy
```

### 4. Verify Deployment

```bash
# Check status
./scripts/deploy-k8s.sh dev status

# Check worker pods
kubectl get pods -n aetherium -l app.kubernetes.io/component=worker

# Check Consul UI
kubectl port-forward svc/aetherium-consul 8500:8500 -n aetherium
# Open http://localhost:8500

# Port forward to access API
kubectl port-forward svc/aetherium-api-gateway 8080:8080 -n aetherium
```

## Service Discovery with Consul

Workers automatically register with Consul on startup and deregister on shutdown.

### How It Works

1. **Worker starts** → Runs pre-flight checks (KVM, Firecracker assets)
2. **Network setup** → Creates bridge, enables IP forwarding
3. **Consul registration** → Registers as `aetherium-worker` service
4. **Health checks** → Consul monitors worker health endpoint
5. **API Gateway** → Discovers workers via Consul catalog

### Consul Service Registration

Workers register with these metadata:

```json
{
  "ID": "aetherium-worker-<node-name>",
  "Name": "aetherium-worker",
  "Tags": ["worker", "firecracker", "kvm"],
  "Address": "<pod-ip>",
  "Port": 8081,
  "Meta": {
    "node": "<node-name>",
    "kvm_enabled": "true"
  },
  "Check": {
    "HTTP": "http://<pod-ip>:8081/health",
    "Interval": "10s"
  }
}
```

### Discovering Workers

From API Gateway or any service:

```bash
# List all workers
curl http://consul:8500/v1/catalog/service/aetherium-worker

# Get healthy workers only
curl http://consul:8500/v1/health/service/aetherium-worker?passing=true
```

## Helm Chart

### Chart Structure

```
helm/aetherium/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default values
├── values-production.yaml  # Production overrides
└── templates/
    ├── _helpers.tpl
    ├── configmap.yaml
    ├── secrets.yaml
    ├── serviceaccount.yaml
    ├── api-gateway-deployment.yaml
    ├── api-gateway-service.yaml
    ├── api-gateway-ingress.yaml
    ├── worker-daemonset.yaml
    ├── consul-statefulset.yaml
    └── NOTES.txt
```

### Installation

```bash
# Add Bitnami repo for dependencies
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install with default values
helm install aetherium ./helm/aetherium -n aetherium --create-namespace

# Install with custom values
helm install aetherium ./helm/aetherium \
  -n aetherium \
  --create-namespace \
  -f ./helm/aetherium/values-production.yaml
```

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.environment` | Environment name | `development` |
| `apiGateway.replicaCount` | API Gateway replicas | `1` |
| `worker.kind` | Worker kind | `DaemonSet` |
| `worker.nodeSelector` | Node selector for workers | `aetherium.io/kvm-enabled: "true"` |
| `consul.enabled` | Deploy Consul | `true` |
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `redis.enabled` | Deploy Redis | `true` |

### Worker DaemonSet Configuration

The worker DaemonSet:
- Runs **one pod per KVM-enabled node**
- Uses **host network** for VM networking
- Mounts **/dev/kvm** and **/dev/vhost-vsock**
- Mounts **/var/firecracker** for kernel/rootfs
- Runs **privileged** for TAP/bridge management
- Has **init container** to verify prerequisites

## Pulumi Infrastructure

### Setup

```bash
cd pulumi

# Install dependencies
npm install

# Login to Pulumi
pulumi login

# Select stack
pulumi stack select dev  # or: staging, prod
```

### Deploy

```bash
# Preview changes
pulumi preview

# Deploy
pulumi up

# View outputs
pulumi stack output
```

### Modules

| Module | Description |
|--------|-------------|
| `index.ts` | Main entry point |
| `namespace.ts` | Creates K8s namespace |
| `infrastructure.ts` | PostgreSQL, Redis, Consul |
| `aetherium.ts` | Helm deployment |
| `bare-metal.ts` | Node preparation DaemonSet |

## Configuration

### External Database

For production, use managed databases:

```yaml
postgresql:
  enabled: false
  external:
    host: aetherium-db.postgres.database.azure.com
    port: 5432
    database: aetherium
    existingSecret: postgres-credentials
```

### External Consul

```yaml
consul:
  enabled: false
  external:
    addr: consul.example.com:8500
```

### Ingress with TLS

```yaml
apiGateway:
  ingress:
    enabled: true
    className: nginx
    annotations:
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: aetherium.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: aetherium-tls
        hosts:
          - aetherium.example.com
```

## Operations

### Scaling Workers

Workers scale by adding KVM-enabled nodes:

```bash
# Add new node to cluster
# Then label it
kubectl label nodes new-node aetherium.io/kvm-enabled=true

# DaemonSet automatically deploys worker
kubectl get pods -n aetherium -l app.kubernetes.io/component=worker -o wide
```

### Monitoring Workers via Consul

```bash
# List all registered workers
kubectl exec -n aetherium aetherium-consul-0 -- \
  consul catalog services

# Check worker health
kubectl exec -n aetherium aetherium-consul-0 -- \
  consul members

# View Consul UI
kubectl port-forward svc/aetherium-consul 8500:8500 -n aetherium
```

### Logs

```bash
# Worker logs
kubectl logs -n aetherium -l app.kubernetes.io/component=worker -f

# API Gateway logs
kubectl logs -n aetherium -l app.kubernetes.io/component=api-gateway -f

# Consul logs
kubectl logs -n aetherium aetherium-consul-0 -f
```

## Troubleshooting

### Worker Pod Stuck in Init

The init container verifies KVM and Firecracker assets:

```bash
# Check init container logs
kubectl logs -n aetherium <worker-pod> -c verify-host

# Common issues:
# - /dev/kvm not found → Enable KVM on host
# - Kernel not found → Run download-vsock-kernel.sh
# - Rootfs not found → Run prepare-rootfs-with-tools.sh
```

### Worker Not Registering with Consul

```bash
# Check worker logs for Consul registration
kubectl logs -n aetherium <worker-pod> | grep -i consul

# Verify Consul is reachable
kubectl exec -n aetherium <worker-pod> -- \
  curl -s http://aetherium-consul:8500/v1/status/leader

# Check environment variables
kubectl exec -n aetherium <worker-pod> -- env | grep CONSUL
```

### VM Creation Fails

```bash
# Check worker logs
kubectl logs -n aetherium <worker-pod> | grep -i firecracker

# Verify KVM access inside pod
kubectl exec -n aetherium <worker-pod> -- ls -la /dev/kvm

# Check Firecracker assets
kubectl exec -n aetherium <worker-pod> -- ls -la /var/firecracker/
```

### Network Issues

Workers use host network for VM TAP devices:

```bash
# Verify host network
kubectl exec -n aetherium <worker-pod> -- ip addr

# Check bridge
kubectl exec -n aetherium <worker-pod> -- ip link show aetherium0

# Check NAT rules
kubectl exec -n aetherium <worker-pod> -- iptables -t nat -L
```

### Common Commands

```bash
# Full status check
./scripts/deploy-k8s.sh <env> status

# Restart workers
kubectl rollout restart daemonset/aetherium-worker -n aetherium

# Force delete stuck pod
kubectl delete pod <pod> -n aetherium --force --grace-period=0

# View all resources
kubectl get all -n aetherium
```
