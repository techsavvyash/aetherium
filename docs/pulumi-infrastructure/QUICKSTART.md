# Pulumi Deployment Quick Start

Get Aetherium running on Kubernetes in 5 minutes.

## Prerequisites

- Kubernetes cluster running (K3s, EKS, AKS, GKE, etc.)
- kubectl configured: `kubectl get nodes`
- Helm 3.x installed
- Pulumi CLI installed: `curl -fsSL https://get.pulumi.com | sh`

## Fast Track

```bash
# 1. Navigate to infrastructure code
cd infrastructure/pulumi/core

# 2. Install dependencies
npm install

# 3. Create or select stack
pulumi stack init development
# or
pulumi stack select development

# 4. Preview what will be created
pulumi preview

# 5. Deploy
pulumi up
# Type 'yes' when prompted

# 6. Monitor deployment
kubectl get pods -n aetherium -w

# 7. View outputs
pulumi stack output
```

## Verify Deployment

```bash
# Check all resources
kubectl get all -n aetherium

# Check database
POD=$(kubectl get pod -n aetherium -l app=postgres -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $POD -n aetherium -- psql -U aetherium -c "SELECT 1"

# Check Redis
POD=$(kubectl get pod -n aetherium -l app=redis -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $POD -n aetherium -- redis-cli ping

# Check Helm release
helm list -n aetherium
```

## Common Commands

```bash
# View current state
pulumi stack

# Export state
pulumi stack export > backup.json

# Destroy all resources
pulumi destroy

# View logs
pulumi logs

# Get specific output
pulumi stack output aetherium
```

## Environment Switching

```bash
# Use development config
pulumi stack select development

# Use production config
pulumi stack select production

# Create new environment
pulumi stack init staging
```

## Configuration

### Development (Local Testing)

```bash
pulumi stack select development
pulumi up
```

- ClusterIP service (local access only)
- 1 replica per service
- Loki disabled

### Production (HA & Monitoring)

```bash
pulumi stack select production
pulumi up
```

- LoadBalancer service (external access)
- Multiple replicas
- Loki enabled
- Consul enabled

## Troubleshooting

### Pods stuck in Pending

```bash
kubectl describe pod -n aetherium <pod-name>
```

Check:
- Storage class: `kubectl get storageclass`
- Node resources: `kubectl describe nodes`
- Events: `kubectl get events -n aetherium`

### Helm chart not found

```bash
# Verify path
ls -la ../../../helm/aetherium/Chart.yaml

# Run from correct directory
cd infrastructure/pulumi/core
pulumi up
```

### TypeScript errors

```bash
# Check compilation
npx tsc --noEmit

# Reinstall dependencies
npm install
```

## Access Services

### Port Forward

```bash
# PostgreSQL
kubectl port-forward -n aetherium svc/postgres 5432:5432

# Redis
kubectl port-forward -n aetherium svc/redis 6379:6379

# API Gateway (if LoadBalancer)
kubectl port-forward -n aetherium svc/aetherium-api-gateway 8080:8080
```

### Get External IP (Production)

```bash
kubectl get svc -n aetherium aetherium-api-gateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

## Cleanup

```bash
# Remove all resources
pulumi destroy

# Remove stack
pulumi stack rm development
```

## Next Steps

1. Configure Helm values: `infrastructure/helm/aetherium/values.yaml`
2. Set up monitoring: Add Prometheus + Grafana
3. Configure GitOps: Use ArgoCD for continuous deployment
4. Add security: Network policies, RBAC, Pod Security Policies

## For More Details

See `README.md` for comprehensive documentation.

## Need Help?

```bash
# Check Pulumi status
pulumi logs

# Describe resources
kubectl describe <resource-type> -n aetherium <resource-name>

# Check events
kubectl get events -n aetherium --sort-by='.lastTimestamp'
```

---

**Start here** → [Full Documentation](./README.md)  
**Issues?** → [Troubleshooting Guide](./README.md#troubleshooting)
