# Aetherium Infrastructure - Pulumi IaC

Infrastructure as Code for deploying Aetherium (distributed task execution platform) using Pulumi and Kubernetes.

## Overview

This Pulumi project deploys:
- **Infrastructure Layer**: PostgreSQL, Redis, Consul (for service discovery), Loki (logging)
- **Application Layer**: Aetherium API Gateway and Workers (via Helm)
- **Node Layer**: Bare-metal node preparation for Firecracker workloads

## Project Structure

```
.
├── index.ts                 # Main orchestration entry point
├── namespace.ts             # Kubernetes namespace creation
├── infrastructure.ts        # Database and infrastructure services (PostgreSQL, Redis, Consul, Loki)
├── aetherium.ts            # Aetherium deployment (API Gateway, Workers)
├── bare-metal.ts           # Bare-metal node preparation
├── node-pools.ts           # Cloud provider node pool configurations
├── package.json            # Node dependencies
├── tsconfig.json           # TypeScript configuration
├── Pulumi.yaml             # Base Pulumi configuration
├── Pulumi.development.yaml # Development environment config
└── Pulumi.production.yaml  # Production environment config
```

## Prerequisites

### Local Requirements
- Node.js >= 16
- Pulumi CLI >= 3.0
- kubectl configured and connected to your cluster
- Helm 3.x

### Installation

```bash
# Install Pulumi (macOS)
brew install pulumi

# Or visit: https://www.pulumi.com/docs/install/

# Install dependencies
npm install
```

### Kubernetes Cluster

You need an existing Kubernetes cluster. Options:

#### Local Development
```bash
# Create a local K3s cluster
curl -sfL https://get.k3s.io | sh -

# Or use kind
kind create cluster --name aetherium
```

#### Cloud Providers
- **AWS**: Create an EKS cluster
- **Azure**: Create an AKS cluster
- **GCP**: Create a GKE cluster

### Configure kubectl

```bash
# For local K3s
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# For cloud providers, follow provider-specific instructions
```

## Configuration

Configuration is managed through Pulumi config files:

### Environment Variables

```bash
# Set for your environment
export PULUMI_STACK=development  # or production
```

### Config File Structure

Each environment has a `Pulumi.<environment>.yaml`:

```yaml
config:
  environment: development        # development, staging, production
  kubernetes:clusterName: aetherium-cluster-dev
  cloud:provider: local           # aws, azure, gcp, local
```

### Setting Configuration

```bash
# Set config directly
pulumi config set environment production

# Or use environment-specific files
# Config automatically loaded from Pulumi.<stack>.yaml
```

## Deployment

### 1. Initialize Pulumi Stack

```bash
# Create a new stack (if needed)
pulumi stack init development

# Or select existing
pulumi stack select development
```

### 2. Preview Changes

```bash
# See what will be created
pulumi preview
```

### 3. Deploy

```bash
# Deploy the infrastructure
pulumi up

# Select "yes" when prompted
```

### 4. Export Outputs

```bash
# View outputs
pulumi stack output

# Get specific output
pulumi stack output namespace
```

## Architecture

### Namespace Layer
- Creates isolated `aetherium` namespace
- Sets appropriate labels and annotations

### Infrastructure Layer (infrastructure.ts)
```
┌─────────────────────────────────────┐
│    Kubernetes Cluster               │
│  ┌───────────────────────────────┐  │
│  │   aetherium Namespace         │  │
│  │                               │  │
│  │  ┌──────────────────────────┐ │  │
│  │  │ PostgreSQL StatefulSet   │ │  │
│  │  │ (1 replica, 10Gi PVC)    │ │  │
│  │  └──────────────────────────┘ │  │
│  │                               │  │
│  │  ┌──────────────────────────┐ │  │
│  │  │ Redis StatefulSet        │ │  │
│  │  │ (1 replica, 5Gi PVC)     │ │  │
│  │  └──────────────────────────┘ │  │
│  │                               │  │
│  │  ┌──────────────────────────┐ │  │
│  │  │ Consul StatefulSet       │ │  │
│  │  │ (Service Discovery)      │ │  │
│  │  └──────────────────────────┘ │  │
│  │                               │  │
│  │  ┌──────────────────────────┐ │  │
│  │  │ Loki StatefulSet (Prod)  │ │  │
│  │  │ (Centralized Logging)    │ │  │
│  │  └──────────────────────────┘ │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

### Application Layer (aetherium.ts)
Deploys Aetherium via Helm chart with environment-specific values:
- API Gateway: 1 pod (dev), 3 pods (prod) - ClusterIP (dev) / LoadBalancer (prod)
- Workers: DaemonSet on KVM-enabled nodes
- Firecracker integration with privileged access
- Proper resource limits and requests

### Bare-Metal Node Layer (bare-metal.ts)
- Node Preparation DaemonSet for KVM setup
- Automatic kernel module loading
- Firecracker binary installation

## Configuration Details

### PostgreSQL
- Image: `postgres:15-alpine`
- Port: 5432
- Storage: 10Gi PersistentVolumeClaim
- Credentials: Stored in Kubernetes Secret
- Resources: 256Mi RAM (request), 1Gi (limit)

### Redis
- Image: `redis:7-alpine`
- Port: 6379
- Storage: 5Gi PersistentVolumeClaim
- AOF Persistence: Enabled
- Resources: 128Mi RAM (request), 512Mi (limit)

### Consul
- Image: `hashicorp/consul:1.17`
- Port: 8500 (HTTP API)
- Purpose: Service discovery for Aetherium workers
- Enabled by default (disable with config)

### Loki
- Image: `grafana/loki:2.9.0`
- Port: 3100
- Storage: 10Gi PersistentVolumeClaim
- Enabled only in production

## Environment-Specific Behavior

### Development (`environment=development`)
- Single replicas for all services
- ClusterIP service for API Gateway (local access only)
- Loki disabled (use stdout logging)
- Consul disabled
- Worker concurrency: 5

### Production (`environment=production`)
- Multiple replicas (HA)
- LoadBalancer service for API Gateway (external access)
- Loki enabled (centralized logging)
- Consul enabled (service discovery)
- Worker concurrency: 20

## Cloud Provider Specific Configuration

### node-pools.ts

Provides cloud-specific node pool configurations:

**AWS:**
- Metal instance types (m5.metal, c5.metal, etc.)
- Amazon Linux 2 AMI with kernel 5.10+
- Cloud-init user data

**Azure:**
- Dsv3 series (nested virtualization support)
- Custom data script

**GCP:**
- N2 series with nested virtualization
- Startup script

## Troubleshooting

### Common Issues

#### 1. Helm Chart Path Not Found
```
error: failed to load chart from '../helm/aetherium'
```

**Solution**: Ensure you're running from the correct directory:
```bash
cd infrastructure/pulumi/core
pulumi up
```

#### 2. Secret Password Issues
```
error: POSTGRES_PASSWORD must be a string
```

**Solution**: Already fixed in this version. The stringData now uses `apply()` to handle secrets.

#### 3. Node Type Errors
```
error: Cannot assign type Output<string> to string
```

**Solution**: All namespace parameters now expect `string`, not `Output<string>`. The orchestration in `index.ts` handles the conversion.

#### 4. Pod Stuck in Pending
```bash
kubectl describe pod -n aetherium <pod-name>
```

Check:
- Storage class available: `kubectl get storageclass`
- Node selectors (workers require KVM label)
- Resource requests vs available

### Debug Commands

```bash
# Check namespace
kubectl get namespace aetherium -o yaml

# Check pods
kubectl get pods -n aetherium

# Check secrets
kubectl get secrets -n aetherium

# Check services
kubectl get svc -n aetherium

# Check persistent volumes
kubectl get pvc -n aetherium

# View Pulumi state
pulumi stack export

# View logs
kubectl logs -n aetherium <pod-name>
```

## Maintenance

### Scaling

#### PostgreSQL Replicas
```bash
# Currently supports single replica only (StatefulSet)
# For HA, use cloud provider managed PostgreSQL
```

#### Worker Replicas
```bash
# Workers scale via DaemonSet (1 per KVM-enabled node)
# Scale by adding more nodes or adjusting node selectors
```

### Upgrades

```bash
# Backup current state
pulumi stack export > backup.json

# Update dependencies
npm update

# Preview changes
pulumi preview

# Deploy
pulumi up
```

### Disaster Recovery

```bash
# Export state
pulumi stack export > stack-backup.json

# Restore from backup
pulumi stack import < stack-backup.json

# Or destroy and recreate
pulumi destroy
pulumi up
```

## Security Considerations

### Secrets Management

Postgres password is stored as Pulumi secret:
```typescript
const postgresPassword = pulumi.secret("aetherium-secret-password");
```

In production:
- Use AWS Secrets Manager, Azure Key Vault, or GCP Secret Manager
- Don't store passwords in code
- Rotate passwords regularly

### Network Policies

Add network policies to restrict traffic:
```bash
# Example: Restrict postgres to aetherium namespace only
kubectl apply -f - <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgres-isolation
  namespace: aetherium
spec:
  podSelector:
    matchLabels:
      app: postgres
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: aetherium
EOF
```

### RBAC

Created service accounts:
- `aetherium-node-admin`: For node labeling job

Add proper RBAC rules in production.

## Monitoring & Logging

### Access Logs in Production

```bash
# Loki logs
kubectl port-forward -n aetherium svc/loki 3100:3100

# Then query via Grafana (if installed)
```

### Prometheus Metrics

Consider adding:
- Prometheus for metrics
- Alert Manager for alerting
- Grafana for visualization

## Cleanup

### Remove Deployment

```bash
# Destroy all resources
pulumi destroy

# Select "yes" to confirm
```

### Remove Stack

```bash
# Remove stack from Pulumi Cloud
pulumi stack rm <stack-name>
```

## Next Steps

1. Customize Helm values for your environment
2. Add network policies for security
3. Implement monitoring and alerting
4. Set up GitOps (ArgoCD) for continuous deployment
5. Add backup policies for PostgreSQL

## References

- [Pulumi Documentation](https://www.pulumi.com/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)
- [Firecracker Documentation](https://github.com/firecracker-microvm/firecracker)

## Support

For issues or questions:
1. Check troubleshooting section above
2. Review Pulumi logs: `pulumi logs`
3. Check Kubernetes events: `kubectl describe <resource>`
4. Open an issue on GitHub
