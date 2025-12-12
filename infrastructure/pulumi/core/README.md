# Aetherium Pulumi Infrastructure

This directory contains Pulumi infrastructure-as-code for deploying Aetherium to Kubernetes.

## Overview

The Pulumi program provisions:

1. **Kubernetes Namespace** - Isolated environment for Aetherium
2. **Infrastructure Components** - PostgreSQL, Redis, and optional Loki
3. **Aetherium Services** - Deployed via Helm chart

## Prerequisites

```bash
# Install Pulumi
curl -fsSL https://get.pulumi.com | sh

# Install Node.js dependencies
npm install

# Login to Pulumi
pulumi login
```

## Quick Start

```bash
# Select stack
pulumi stack select dev

# Preview changes
pulumi preview

# Deploy
pulumi up

# View outputs
pulumi stack output
```

## Stacks

| Stack | Purpose | Cloud Provider |
|-------|---------|----------------|
| `dev` | Development/local | Local (kind/minikube) |
| `staging` | Staging environment | Azure/AWS/GCP |
| `prod` | Production | Azure/AWS/GCP |

### Configuration

```bash
# Set config for a stack
pulumi config set kubernetes:clusterName my-cluster
pulumi config set cloud:provider azure

# Set secrets
pulumi config set --secret postgresql:password my-secret-password
```

## Project Structure

```
pulumi/
├── Pulumi.yaml           # Project definition
├── Pulumi.dev.yaml       # Dev stack config
├── Pulumi.prod.yaml      # Prod stack config
├── package.json          # Node.js dependencies
├── tsconfig.json         # TypeScript config
├── index.ts              # Main entry point
├── namespace.ts          # Namespace creation
├── infrastructure.ts     # PostgreSQL, Redis, Loki
└── aetherium.ts          # Helm deployment
```

## Modules

### namespace.ts

Creates the Kubernetes namespace with appropriate labels.

### infrastructure.ts

Deploys infrastructure components:
- **PostgreSQL** - StatefulSet with PVC
- **Redis** - StatefulSet with PVC
- **Loki** (optional) - For centralized logging

### aetherium.ts

Deploys Aetherium services using the Helm chart:
- API Gateway Deployment
- Worker DaemonSet

## Outputs

After deployment, Pulumi exports:

```typescript
{
  namespace: "aetherium",
  environment: "production",
  clusterName: "aetherium-prod",
  services: {
    apiGateway: { ... },
    worker: { ... }
  },
  endpoints: {
    postgres: "postgres:5432",
    redis: "redis:6379"
  }
}
```

## Commands

```bash
# Preview changes
pulumi preview

# Deploy
pulumi up

# View logs
pulumi logs

# View stack
pulumi stack

# Destroy (careful!)
pulumi destroy
```

## Integration with Helm

The Pulumi program uses the Helm chart in `../helm/aetherium/`. It passes configuration from Pulumi to Helm values, allowing infrastructure and application to be managed together.

```typescript
// From aetherium.ts
const helmRelease = new k8s.helm.v3.Release("aetherium", {
    chart: "../helm/aetherium",
    values: {
        // Values passed from Pulumi config
    },
});
```
