# Aetherium Kubernetes Deployment Guide

This guide covers deploying Aetherium to Kubernetes using Helm charts and Pulumi infrastructure-as-code.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Architecture Overview](#architecture-overview)
- [Quick Start](#quick-start)
- [Helm Chart](#helm-chart)
- [Pulumi Infrastructure](#pulumi-infrastructure)
- [Node Requirements](#node-requirements)
- [Configuration](#configuration)
- [Operations](#operations)
- [Troubleshooting](#troubleshooting)

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
- At least one node with KVM support (for Firecracker workers)
- Persistent volume provisioner (for PostgreSQL/Redis)
- Optional: Ingress controller (nginx-ingress recommended)

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                            │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    aetherium namespace                    │   │
│  │                                                           │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐      │   │
│  │  │ API Gateway │  │ API Gateway │  │ API Gateway │      │   │
│  │  │  (replica)  │  │  (replica)  │  │  (replica)  │      │   │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘      │   │
│  │         └────────────────┼────────────────┘              │   │
│  │                          ▼                                │   │
│  │                   ┌─────────────┐                        │   │
│  │                   │   Service   │                        │   │
│  │                   │ (ClusterIP) │                        │   │
│  │                   └──────┬──────┘                        │   │
│  │                          │                                │   │
│  └──────────────────────────┼────────────────────────────────┘   │
│                             │                                    │
│  ┌──────────────────────────┼────────────────────────────────┐   │
│  │           Worker Nodes (KVM-enabled)                       │   │
│  │                          │                                 │   │
│  │  ┌───────────────────────┼───────────────────────────┐    │   │
│  │  │         DaemonSet: aetherium-worker               │    │   │
│  │  │                       │                            │    │   │
│  │  │  Node 1           Node 2           Node 3         │    │   │
│  │  │  ┌──────┐         ┌──────┐         ┌──────┐      │    │   │
│  │  │  │Worker│         │Worker│         │Worker│      │    │   │
│  │  │  │ Pod  │         │ Pod  │         │ Pod  │      │    │   │
│  │  │  │ /kvm │         │ /kvm │         │ /kvm │      │    │   │
│  │  │  └──────┘         └──────┘         └──────┘      │    │   │
│  │  │     │                 │                 │          │    │   │
│  │  │  ┌──┴──┐           ┌──┴──┐           ┌──┴──┐     │    │   │
│  │  │  │VMs  │           │VMs  │           │VMs  │     │    │   │
│  │  │  └─────┘           └─────┘           └─────┘     │    │   │
│  │  └────────────────────────────────────────────────────┘    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │                   Data Layer                              │    │
│  │  ┌─────────────────┐      ┌─────────────────┐           │    │
│  │  │   PostgreSQL    │      │      Redis      │           │    │
│  │  │  (StatefulSet)  │      │  (StatefulSet)  │           │    │
│  │  │    + PVC        │      │     + PVC       │           │    │
│  │  └─────────────────┘      └─────────────────┘           │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

## Quick Start

### 1. Build and Push Docker Images

```bash
# Build all images
./scripts/build-images.sh v1.0.0

# Build and push to registry
./scripts/build-images.sh v1.0.0 push
```

### 2. Deploy with Helm

```bash
# Development environment
./scripts/deploy-k8s.sh dev deploy

# Production environment
./scripts/deploy-k8s.sh prod deploy
```

### 3. Verify Deployment

```bash
# Check status
./scripts/deploy-k8s.sh dev status

# Port forward to access API
kubectl port-forward svc/aetherium-api-gateway 8080:8080 -n aetherium-dev
```

## Helm Chart

### Chart Structure

```
helm/aetherium/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default values
├── values-production.yaml  # Production overrides
└── templates/
    ├── _helpers.tpl        # Template helpers
    ├── configmap.yaml      # Application config
    ├── secrets.yaml        # Secrets
    ├── serviceaccount.yaml # Service account
    ├── api-gateway-deployment.yaml
    ├── api-gateway-service.yaml
    ├── api-gateway-ingress.yaml
    ├── worker-daemonset.yaml
    ├── worker-deployment.yaml
    └── NOTES.txt           # Post-install notes
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
  -f ./helm/aetherium/values-production.yaml \
  --set secrets.postgres.password=my-secure-password
```

### Upgrade

```bash
# Upgrade with new values
helm upgrade aetherium ./helm/aetherium -n aetherium \
  -f ./helm/aetherium/values-production.yaml

# Upgrade with specific image tag
helm upgrade aetherium ./helm/aetherium -n aetherium \
  --set apiGateway.image.tag=v1.1.0 \
  --set worker.image.tag=v1.1.0
```

### Rollback

```bash
# Rollback to previous revision
helm rollback aetherium -n aetherium

# Rollback to specific revision
helm rollback aetherium 3 -n aetherium
```

### Key Values

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.environment` | Environment name | `development` |
| `apiGateway.replicaCount` | API Gateway replicas | `1` |
| `apiGateway.service.type` | Service type | `ClusterIP` |
| `worker.kind` | Worker kind (DaemonSet/Deployment) | `DaemonSet` |
| `worker.hostNetwork` | Use host network | `true` |
| `postgresql.enabled` | Deploy PostgreSQL | `true` |
| `redis.enabled` | Deploy Redis | `true` |

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

### Deploy Infrastructure

```bash
# Preview changes
pulumi preview

# Deploy
pulumi up

# Destroy (careful!)
pulumi destroy
```

### Stack Configuration

**Development (Pulumi.dev.yaml):**
```yaml
config:
  aetherium-infra:environment: development
  kubernetes:clusterName: aetherium-dev
  kubernetes:nodeCount: "1"
  cloud:provider: local
```

**Production (Pulumi.prod.yaml):**
```yaml
config:
  aetherium-infra:environment: production
  kubernetes:clusterName: aetherium-prod
  kubernetes:nodeCount: "5"
  kubernetes:nodeSize: Standard_D8s_v3
  cloud:provider: azure
```

## Node Requirements

### KVM-Enabled Nodes

Workers require KVM access for Firecracker VMs. Label appropriate nodes:

```bash
# Label nodes with KVM support
kubectl label nodes <node-name> aetherium.io/kvm-enabled=true

# Verify KVM is available (on the node)
ls -la /dev/kvm
ls -la /dev/vhost-vsock
```

### Node Preparation

On each worker node, ensure:

1. **KVM Module Loaded:**
```bash
sudo modprobe kvm
sudo modprobe kvm_intel  # or kvm_amd
```

2. **Vsock Module Loaded:**
```bash
sudo modprobe vhost_vsock
```

3. **Firecracker Assets:**
```bash
# Directory structure
/var/firecracker/
├── vmlinux           # Kernel with vsock support
└── rootfs.ext4       # Root filesystem template
```

### Taint Worker Nodes (Optional)

To dedicate nodes to Aetherium workers:

```bash
# Taint node
kubectl taint nodes <node-name> aetherium.io/worker=true:NoSchedule

# The Helm chart includes matching tolerations
```

## Configuration

### External Database

For production, use managed databases:

```yaml
# values-production.yaml
postgresql:
  enabled: false
  external:
    host: aetherium-db.postgres.database.azure.com
    port: 5432
    database: aetherium
    existingSecret: postgres-credentials

redis:
  enabled: false
  external:
    host: aetherium-cache.redis.cache.windows.net
    port: 6380
```

### Ingress Configuration

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

### Resource Limits

```yaml
apiGateway:
  resources:
    requests:
      memory: "256Mi"
      cpu: "250m"
    limits:
      memory: "512Mi"
      cpu: "500m"

worker:
  resources:
    requests:
      memory: "1Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "4000m"
```

## Operations

### Scaling

```bash
# Scale API Gateway (if using Deployment)
kubectl scale deployment aetherium-api-gateway -n aetherium --replicas=5

# Workers scale automatically with DaemonSet (add more KVM nodes)
kubectl label nodes new-node aetherium.io/kvm-enabled=true
```

### Monitoring

```bash
# View logs
kubectl logs -n aetherium -l app.kubernetes.io/component=api-gateway -f
kubectl logs -n aetherium -l app.kubernetes.io/component=worker -f

# Check resource usage
kubectl top pods -n aetherium

# View events
kubectl get events -n aetherium --sort-by='.lastTimestamp'
```

### Backup

```bash
# Backup PostgreSQL
kubectl exec -n aetherium aetherium-postgresql-0 -- \
  pg_dump -U aetherium aetherium > backup.sql

# Backup configuration
helm get values aetherium -n aetherium > values-backup.yaml
```

## Troubleshooting

### Worker Pod CrashLoopBackOff

1. Check if KVM is available:
```bash
kubectl exec -n aetherium -it <worker-pod> -- ls -la /dev/kvm
```

2. Check for missing Firecracker assets:
```bash
kubectl exec -n aetherium -it <worker-pod> -- ls -la /var/firecracker/
```

3. View worker logs:
```bash
kubectl logs -n aetherium <worker-pod> --previous
```

### Network Issues

1. Workers use host network; check node firewall rules
2. Verify bridge interface exists on worker nodes:
```bash
kubectl exec -n aetherium -it <worker-pod> -- ip addr show aetherium0
```

### PostgreSQL Connection Issues

1. Check PostgreSQL pod status:
```bash
kubectl get pods -n aetherium -l app.kubernetes.io/name=postgresql
```

2. Verify credentials:
```bash
kubectl get secret -n aetherium postgres-credentials -o yaml
```

3. Test connection from API Gateway:
```bash
kubectl exec -n aetherium -it <api-gateway-pod> -- \
  nc -zv postgres 5432
```

### Common Commands

```bash
# Full status check
./scripts/deploy-k8s.sh <env> status

# Restart all pods
kubectl rollout restart deployment -n aetherium
kubectl rollout restart daemonset -n aetherium

# Force delete stuck pod
kubectl delete pod <pod> -n aetherium --force --grace-period=0

# View all resources
kubectl get all -n aetherium
```
