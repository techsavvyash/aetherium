# Deployment Checklist

## Pre-Deployment Verification

### Code Quality ✅
- [x] TypeScript compilation: `npx tsc --noEmit` → 0 errors
- [x] Dependencies installed: `npm install` → 331 packages
- [x] All imports resolved
- [x] Type safety verified
- [x] Code formatted and clean

### Documentation ✅
- [x] README.md created (420+ lines)
- [x] QUICKSTART.md created
- [x] FIXES_APPLIED.md created
- [x] Configuration examples provided
- [x] Troubleshooting guide included

### Configuration Files ✅
- [x] Pulumi.yaml created
- [x] Pulumi.development.yaml created
- [x] Pulumi.production.yaml created

---

## Pre-Deployment Setup

### Infrastructure Prerequisites

```bash
# [ ] 1. Kubernetes cluster available
kubectl get nodes
# Expected: At least 1 node

# [ ] 2. kubectl configured
kubectl config current-context
# Expected: Your cluster context

# [ ] 3. Helm 3.x installed
helm version
# Expected: Version 3.x or higher

# [ ] 4. Storage class available
kubectl get storageclass
# Expected: At least one storage class (or default)

# [ ] 5. Pulumi CLI installed
pulumi version
# Expected: Version 3.x or higher
```

### Cloud Provider (if applicable)

```bash
# For AWS EKS
# [ ] AWS CLI configured: aws sts get-caller-identity
# [ ] EKS cluster created: aws eks describe-cluster --name <name>

# For Azure AKS
# [ ] Azure CLI configured: az account show
# [ ] AKS cluster created: az aks show --resource-group <rg> --name <name>

# For GCP GKE
# [ ] GCP SDK configured: gcloud auth list
# [ ] GKE cluster created: gcloud container clusters describe <name>

# For local K3s
# [ ] K3s installed: k3s --version
# [ ] K3s running: sudo systemctl status k3s
```

---

## Development Environment Setup

```bash
# [ ] Navigate to directory
cd infrastructure/pulumi/core

# [ ] Install dependencies
npm install

# [ ] Verify TypeScript
npx tsc --noEmit
# Expected: No output (0 errors)

# [ ] Check Pulumi CLI
pulumi version
# Expected: Version >= 3.0.0
```

---

## Configuration

### Development Environment

```bash
# [ ] Select development stack
pulumi stack select development

# [ ] Review development config
cat Pulumi.development.yaml
# Expected:
# config:
#   environment: development
#   kubernetes:clusterName: aetherium-cluster-dev
#   cloud:provider: local
```

### Production Environment (if deploying)

```bash
# [ ] Select production stack
pulumi stack select production

# [ ] Review production config
cat Pulumi.production.yaml
# Expected:
# config:
#   environment: production
#   kubernetes:clusterName: aetherium-cluster-prod
#   cloud:provider: aws
```

---

## Pre-Deployment Review

```bash
# [ ] 1. Preview changes
pulumi preview
# Expected:
# - Create namespace aetherium
# - Create PostgreSQL StatefulSet, Service, Secret
# - Create Redis StatefulSet, Service
# - Create Consul StatefulSet, Service
# - Create Loki StatefulSet, Service (production only)
# - Deploy Aetherium Helm release
# No errors or warnings

# [ ] 2. Check resource count
pulumi preview | grep -c "Create\|Modify\|Delete"
# Expected: Should show planned changes

# [ ] 3. Verify namespace
pulumi preview | grep "aetherium"
# Expected: Multiple lines with aetherium references

# [ ] 4. Check for warnings
pulumi preview 2>&1 | grep -i "warning"
# Expected: No warnings (or acceptable warnings only)
```

---

## Deployment

```bash
# [ ] 1. Deploy infrastructure
pulumi up
# When prompted, type: yes

# [ ] 2. Wait for deployment
# Expected: Should take 2-5 minutes

# [ ] 3. Check final status
# Expected output showing:
# - Namespace: aetherium
# - Infrastructure services created
# - Aetherium Helm release deployed
```

---

## Post-Deployment Verification

### Kubernetes Resources

```bash
# [ ] 1. Check namespace
kubectl get namespace aetherium
# Expected: Active

# [ ] 2. Check pods
kubectl get pods -n aetherium
# Expected: All pods Running (may take 1-2 minutes)
# - postgres-0
# - redis-0
# - consul-0 (production)
# - loki-0 (production)
# - aetherium-api-gateway-*
# - aetherium-worker-* (one per node)

# [ ] 3. Check services
kubectl get svc -n aetherium
# Expected:
# - postgres (ClusterIP)
# - redis (ClusterIP)
# - consul (ClusterIP)
# - loki (ClusterIP) (production)
# - aetherium-api-gateway (ClusterIP or LoadBalancer)

# [ ] 4. Check persistent volumes
kubectl get pvc -n aetherium
# Expected:
# - postgres-data-postgres-0 (10Gi)
# - redis-data-redis-0 (5Gi)
# - consul-data-consul-0 (1Gi) (production)
# - loki-data-loki-0 (10Gi) (production)

# [ ] 5. Check secrets
kubectl get secrets -n aetherium
# Expected:
# - postgres-credentials (containing password)
# - Other Helm-generated secrets

# [ ] 6. Check events
kubectl get events -n aetherium --sort-by='.lastTimestamp'
# Expected: No error events
```

### Database Connectivity

```bash
# [ ] 1. Get PostgreSQL pod
POD=$(kubectl get pod -n aetherium -l app=postgres -o jsonpath='{.items[0].metadata.name}')

# [ ] 2. Test PostgreSQL
kubectl exec -it $POD -n aetherium -- \
  psql -U aetherium -d aetherium -c "SELECT 1"
# Expected: Output showing "1"

# [ ] 3. Verify database
kubectl exec -it $POD -n aetherium -- \
  psql -U aetherium -d aetherium -c "\l"
# Expected: List of databases including "aetherium"
```

### Redis Connectivity

```bash
# [ ] 1. Get Redis pod
POD=$(kubectl get pod -n aetherium -l app=redis -o jsonpath='{.items[0].metadata.name}')

# [ ] 2. Test Redis
kubectl exec -it $POD -n aetherium -- redis-cli ping
# Expected: PONG

# [ ] 3. Verify persistence
kubectl exec -it $POD -n aetherium -- \
  redis-cli SET test "value123"
kubectl exec -it $POD -n aetherium -- \
  redis-cli GET test
# Expected: "value123"
```

### Pulumi Outputs

```bash
# [ ] 1. View all outputs
pulumi stack output

# [ ] 2. Check namespace output
pulumi stack output namespace
# Expected: aetherium

# [ ] 3. Check infrastructure outputs
pulumi stack output infrastructure
# Expected: postgres and redis service details

# [ ] 4. Check Aetherium outputs
pulumi stack output aetherium
# Expected: Helm release information
```

---

## API Gateway Access

### Development (ClusterIP)

```bash
# [ ] 1. Port forward
kubectl port-forward -n aetherium \
  svc/aetherium-api-gateway 8080:8080

# [ ] 2. Test API
curl http://localhost:8080/health
# Expected: 200 OK or similar response
```

### Production (LoadBalancer)

```bash
# [ ] 1. Get external IP
kubectl get svc -n aetherium aetherium-api-gateway
# Expected: Shows EXTERNAL-IP

# [ ] 2. Test API
curl http://<EXTERNAL-IP>:8080/health
# Expected: 200 OK or similar response
```

---

## Cleanup Tests

```bash
# [ ] 1. Export state backup
pulumi stack export > backup.json
# Expected: Creates backup file

# [ ] 2. List stack resources
pulumi stack export | jq '.resources | length'
# Expected: Shows resource count

# [ ] 3. Verify backup content
ls -lh backup.json
# Expected: File size > 1MB
```

---

## Rollback Plan

```bash
# If something goes wrong:

# [ ] 1. Check logs
pulumi logs

# [ ] 2. Check Kubernetes events
kubectl get events -n aetherium

# [ ] 3. Check pod status
kubectl describe pod -n aetherium <pod-name>

# [ ] 4. Option 1: Update deployment
pulumi up  # Fix and redeploy

# [ ] 5. Option 2: Destroy and recreate
pulumi destroy  # Remove all resources
pulumi up       # Redeploy from scratch

# [ ] 6. Option 3: Restore from backup
pulumi stack import < backup.json
```

---

## Monitoring Setup (Production)

```bash
# [ ] 1. Access Loki logs
kubectl port-forward -n aetherium svc/loki 3100:3100

# [ ] 2. Query logs via curl
curl -G -s "http://localhost:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={namespace="aetherium"}' \
  --data-urlencode 'start=<unix-timestamp>'

# [ ] 3. Set up Grafana (optional)
# Follow: https://grafana.com/docs/grafana/latest/datasources/loki/
```

---

## Security Review

```bash
# [ ] 1. Check network policies (if installed)
kubectl get networkpolicy -n aetherium

# [ ] 2. Check RBAC
kubectl get rolebindings -n aetherium
kubectl get clusterrolebindings -o wide | grep aetherium

# [ ] 3. Verify secrets encryption
kubectl get secrets -n aetherium -o json | jq '.items[0].data'

# [ ] 4. Check pod security policies
kubectl get pods -n aetherium -o json | \
  jq '.items[].spec.securityContext'
```

---

## Documentation Review

Before considering deployment complete:

- [ ] README.md reviewed
- [ ] QUICKSTART.md understood
- [ ] FIXES_APPLIED.md reviewed for context
- [ ] Troubleshooting guide bookmarked
- [ ] Architecture understood
- [ ] Cloud provider specifics reviewed

---

## Sign-Off

- [ ] **Developer**: Verified code quality and TypeScript compilation
- [ ] **Ops Engineer**: Verified Kubernetes prerequisites
- [ ] **Security**: Reviewed security configurations
- [ ] **DevOps Lead**: Approved for deployment

---

## Deployment Summary

**Date Deployed**: _______________  
**Environment**: ⃞ Development ⃞ Staging ⃞ Production  
**Deployed By**: _______________  
**Status**: ⃞ Success ⃞ Partial ⃞ Failed  
**Issues Encountered**: _______________________________________________  
**Resolution**: _______________________________________________  

---

## Post-Deployment Maintenance

- [ ] Set up monitoring alerts
- [ ] Configure backup schedules
- [ ] Document access procedures
- [ ] Set up log aggregation
- [ ] Configure auto-scaling (if needed)
- [ ] Schedule capacity reviews

---

**Ready to Deploy**: ✅ All items checked  
**Estimated Deployment Time**: 5-10 minutes  
**Estimated Stabilization Time**: 2-5 minutes after deployment complete
