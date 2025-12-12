# Production Deployment Guide

Complete guide for deploying Aetherium in production environments.

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Clients   │────▶│ API Gateway  │────▶│    Redis    │
│  (REST/WS)  │     │  (Port 8080) │     │   (Queue)   │
└─────────────┘     └──────────────┘     └─────────────┘
                            │                     │
                            ▼                     ▼
                    ┌──────────────┐     ┌─────────────┐
                    │  PostgreSQL  │     │   Workers   │
                    │   (State)    │     │ (Multiple)  │
                    └──────────────┘     └─────────────┘
                                                 │
                            ┌────────────────────┼────────────┐
                            ▼                    ▼            ▼
                    ┌─────────────┐     ┌─────────────┐  ┌──────┐
                    │   Loki       │     │  Firecracker│  │ VMs  │
                    │  (Logging)   │     │    VMM      │  └──────┘
                    └─────────────┘     └─────────────┘
```

---

## Prerequisites

### System Requirements

**API Gateway:**
- CPU: 2+ cores
- RAM: 2GB minimum, 4GB recommended
- Disk: 20GB
- OS: Linux (Ubuntu 22.04+ recommended)

**Worker Nodes:**
- CPU: 8+ cores (for VM operations)
- RAM: 16GB minimum, 32GB+ recommended
- Disk: 100GB+ (for VM rootfs and storage)
- OS: Linux with KVM support
- Kernel: 4.14+ (for Firecracker)

**Infrastructure:**
- PostgreSQL 15+
- Redis 7+
- Grafana Loki (optional, for logging)

### Software Dependencies

```bash
# Install Firecracker
wget https://github.com/firecracker-microvm/firecracker/releases/download/v1.5.0/firecracker-v1.5.0-x86_64.tgz
tar xvf firecracker-v1.5.0-x86_64.tgz
sudo cp release-v1.5.0-x86_64/firecracker-v1.5.0-x86_64 /usr/local/bin/firecracker
sudo chmod +x /usr/local/bin/firecracker

# Install required kernel modules
sudo modprobe vhost_vsock
sudo modprobe kvm

# Add user to kvm group
sudo usermod -aG kvm $USER
```

---

## Option 1: Docker Compose Deployment

### Quick Start

1. **Clone and Build**

```bash
git clone https://github.com/yourorg/aetherium.git
cd aetherium

# Build binaries
make build

# Build Docker images
docker build -t aetherium/api-gateway -f docker/Dockerfile.api-gateway .
docker build -t aetherium/worker -f docker/Dockerfile.worker .
```

2. **Configure Environment**

```bash
cp .env.example .env

# Edit .env
vim .env
```

```env
# Database
POSTGRES_PASSWORD=your_secure_password
DB_PASSWORD=your_secure_password

# Redis
REDIS_PASSWORD=your_redis_password

# Integrations
GITHUB_TOKEN=ghp_xxxxx
GITHUB_WEBHOOK_SECRET=xxxxx
SLACK_BOT_TOKEN=xoxb-xxxxx
SLACK_SIGNING_SECRET=xxxxx

# API
JWT_SECRET=your_jwt_secret
API_KEY_1=your_api_key

# Loki
LOKI_URL=http://loki:3100
```

3. **Deploy**

```bash
docker-compose up -d
```

### Docker Compose File

```yaml
version: '3.8'

services:
  # PostgreSQL Database
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: aetherium
      POSTGRES_USER: aetherium
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U aetherium"]
      interval: 10s
      timeout: 5s
      retries: 5

  # Redis Queue
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s

  # Loki Logging
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    volumes:
      - ./config/loki-config.yaml:/etc/loki/local-config.yaml
      - loki_data:/loki

  # Grafana Dashboard
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Admin
    volumes:
      - grafana_data:/var/lib/grafana

  # API Gateway
  api-gateway:
    image: aetherium/api-gateway:latest
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - POSTGRES_HOST=postgres
      - POSTGRES_USER=aetherium
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=aetherium
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - LOKI_URL=http://loki:3100
      - GITHUB_TOKEN=${GITHUB_TOKEN}
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - SLACK_BOT_TOKEN=${SLACK_BOT_TOKEN}
      - SLACK_SIGNING_SECRET=${SLACK_SIGNING_SECRET}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      loki:
        condition: service_started

  # Worker (runs on host for Firecracker access)
  worker:
    image: aetherium/worker:latest
    privileged: true
    volumes:
      - /var/firecracker:/var/firecracker
      - /dev/kvm:/dev/kvm
    environment:
      - POSTGRES_HOST=postgres
      - POSTGRES_USER=aetherium
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=aetherium
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    network_mode: host

volumes:
  postgres_data:
  redis_data:
  loki_data:
  grafana_data:
```

---

## Option 2: Kubernetes Deployment

### Prerequisites

- Kubernetes 1.24+
- Helm 3+
- Persistent Volume provisioner
- Firecracker-compatible nodes (bare metal or nested virtualization)

### Deploy with Helm

1. **Add Helm Repository**

```bash
helm repo add aetherium https://charts.aetherium.io
helm repo update
```

2. **Create Values File**

```yaml
# values-production.yaml

replicaCount:
  apiGateway: 3
  worker: 5

image:
  registry: docker.io
  repository: aetherium
  tag: latest
  pullPolicy: IfNotPresent

postgresql:
  enabled: true
  auth:
    username: aetherium
    password: changeme
    database: aetherium
  primary:
    persistence:
      enabled: true
      size: 50Gi

redis:
  enabled: true
  auth:
    password: changeme
  master:
    persistence:
      enabled: true
      size: 10Gi

loki:
  enabled: true
  persistence:
    enabled: true
    size: 100Gi

config:
  tools:
    default:
      - git
      - nodejs@20
      - bun@latest
      - claude-code@latest
    timeout: 20m

  integrations:
    github:
      enabled: true
      tokenSecret: github-token
    slack:
      enabled: true
      tokenSecret: slack-token

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: api.aetherium.io
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: aetherium-tls
      hosts:
        - api.aetherium.io
```

3. **Create Secrets**

```bash
# GitHub
kubectl create secret generic github-token \
  --from-literal=token=ghp_xxxxx \
  --from-literal=webhook-secret=xxxxx

# Slack
kubectl create secret generic slack-token \
  --from-literal=bot-token=xoxb-xxxxx \
  --from-literal=signing-secret=xxxxx

# API Keys
kubectl create secret generic api-keys \
  --from-literal=jwt-secret=xxxxx \
  --from-literal=api-key-1=xxxxx
```

4. **Install**

```bash
helm install aetherium aetherium/aetherium \
  -f values-production.yaml \
  --namespace aetherium \
  --create-namespace
```

### Worker DaemonSet

Workers need special permissions for Firecracker:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: aetherium-worker
  namespace: aetherium
spec:
  selector:
    matchLabels:
      app: aetherium-worker
  template:
    metadata:
      labels:
        app: aetherium-worker
    spec:
      hostNetwork: true
      hostPID: true
      nodeSelector:
        aetherium.io/worker: "true"
      containers:
      - name: worker
        image: aetherium/worker:latest
        securityContext:
          privileged: true
        volumeMounts:
        - name: firecracker
          mountPath: /var/firecracker
        - name: kvm
          mountPath: /dev/kvm
        env:
        - name: POSTGRES_HOST
          value: aetherium-postgresql
        - name: REDIS_ADDR
          value: aetherium-redis-master:6379
      volumes:
      - name: firecracker
        hostPath:
          path: /var/firecracker
          type: DirectoryOrCreate
      - name: kvm
        hostPath:
          path: /dev/kvm
```

---

## Option 3: Bare Metal Deployment

### Setup Script

```bash
#!/bin/bash
set -e

echo "=== Aetherium Production Setup ==="

# 1. Install dependencies
apt-get update
apt-get install -y \
  postgresql-15 \
  redis-server \
  firecracker \
  build-essential \
  curl \
  wget

# 2. Configure PostgreSQL
sudo -u postgres psql <<EOF
CREATE DATABASE aetherium;
CREATE USER aetherium WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE aetherium TO aetherium;
EOF

# 3. Configure Redis
cat > /etc/redis/redis.conf <<EOF
requirepass your_redis_password
maxmemory 2gb
maxmemory-policy allkeys-lru
EOF

systemctl restart redis

# 4. Run migrations
cd /opt/aetherium
./bin/migrate -database "postgres://aetherium:your_password@localhost/aetherium?sslmode=disable" \
              -path ./migrations up

# 5. Prepare rootfs
sudo ./scripts/prepare-rootfs-with-tools.sh

# 6. Configure systemd services
cat > /etc/systemd/system/aetherium-api-gateway.service <<EOF
[Unit]
Description=Aetherium API Gateway
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=aetherium
WorkingDirectory=/opt/aetherium
EnvironmentFile=/etc/aetherium/env
ExecStart=/opt/aetherium/bin/api-gateway
Restart=always

[Install]
WantedBy=multi-user.target
EOF

cat > /etc/systemd/system/aetherium-worker.service <<EOF
[Unit]
Description=Aetherium Worker
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/aetherium
EnvironmentFile=/etc/aetherium/env
ExecStart=/opt/aetherium/bin/worker
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# 7. Start services
systemctl daemon-reload
systemctl enable aetherium-api-gateway aetherium-worker
systemctl start aetherium-api-gateway aetherium-worker

echo "✓ Aetherium installed successfully"
```

---

## Configuration Management

### Environment Variables

```bash
# /etc/aetherium/env

# Database
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=aetherium
POSTGRES_PASSWORD=secure_password
POSTGRES_DB=aetherium

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=redis_password

# Loki
LOKI_URL=http://localhost:3100

# Integrations
GITHUB_TOKEN=ghp_xxxxx
GITHUB_WEBHOOK_SECRET=xxxxx
SLACK_BOT_TOKEN=xoxb-xxxxx
SLACK_SIGNING_SECRET=xxxxx

# API
PORT=8080
JWT_SECRET=jwt_secret
API_KEY_1=api_key_1

# Tools
AETHERIUM_DEFAULT_TOOLS=git,nodejs,bun,claude-code
AETHERIUM_TOOL_TIMEOUT=20m
```

### Config File

```yaml
# /etc/aetherium/config.yaml

server:
  host: 0.0.0.0
  port: 8080
  mode: production

database:
  host: ${POSTGRES_HOST}
  port: 5432
  user: ${POSTGRES_USER}
  password: ${POSTGRES_PASSWORD}
  database: ${POSTGRES_DB}
  sslmode: require

# ... (rest of config/production.yaml)
```

---

## Monitoring & Observability

### Prometheus Metrics

Expose metrics endpoint:

```go
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
```

**Key Metrics:**
- `aetherium_vms_total` - Total VMs created
- `aetherium_tasks_total{status}` - Tasks by status
- `aetherium_executions_total{exit_code}` - Executions by exit code
- `aetherium_api_requests_total{method,endpoint,status}` - API requests

### Grafana Dashboard

Import dashboard:

```json
{
  "dashboard": {
    "title": "Aetherium Overview",
    "panels": [
      {
        "title": "VM Count",
        "targets": [
          {"expr": "aetherium_vms_total"}
        ]
      },
      {
        "title": "Task Success Rate",
        "targets": [
          {"expr": "rate(aetherium_tasks_total{status=\"completed\"}[5m])"}
        ]
      }
    ]
  }
}
```

### Loki Queries

```logql
# All errors
{service="aetherium"} |= "error" | json

# VM creation logs
{service="aetherium", component="worker"} |= "Creating VM"

# Failed executions
{service="aetherium"} | json | exit_code != "0"
```

---

## Security

### TLS/SSL

```bash
# Generate certificate
certbot certonly --standalone -d api.aetherium.io

# Configure API Gateway
export TLS_CERT=/etc/letsencrypt/live/api.aetherium.io/fullchain.pem
export TLS_KEY=/etc/letsencrypt/live/api.aetherium.io/privkey.pem
```

### Firewall Rules

```bash
# Allow API Gateway
ufw allow 8080/tcp

# Allow PostgreSQL (internal only)
ufw allow from 10.0.0.0/8 to any port 5432

# Allow Redis (internal only)
ufw allow from 10.0.0.0/8 to any port 6379

# Enable firewall
ufw enable
```

### Network Policies (Kubernetes)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: aetherium-api-gateway
spec:
  podSelector:
    matchLabels:
      app: api-gateway
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgresql
    ports:
    - protocol: TCP
      port: 5432
```

---

## Backup & Recovery

### Database Backup

```bash
# Automated backup script
#!/bin/bash
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/var/backups/aetherium

pg_dump -h localhost -U aetherium aetherium | \
  gzip > $BACKUP_DIR/aetherium_$TIMESTAMP.sql.gz

# Retain last 30 days
find $BACKUP_DIR -mtime +30 -delete
```

### Rootfs Backup

```bash
# Backup rootfs
cp /var/firecracker/rootfs.ext4 \
   /var/backups/aetherium/rootfs_$(date +%Y%m%d).ext4
```

### Recovery

```bash
# Restore database
gunzip < /var/backups/aetherium/aetherium_20251005.sql.gz | \
  psql -h localhost -U aetherium aetherium

# Restore rootfs
cp /var/backups/aetherium/rootfs_20251005.ext4 \
   /var/firecracker/rootfs.ext4
```

---

## Scaling

### Horizontal Scaling

**API Gateway:**
- Deploy multiple instances behind load balancer
- Stateless design allows easy scaling
- Use sticky sessions for WebSocket connections

**Workers:**
- Add more worker nodes as needed
- Each worker can handle 10-20 concurrent VMs
- Auto-scale based on queue depth

### Load Balancer Config

```nginx
upstream aetherium_api {
    least_conn;
    server api-gateway-1:8080;
    server api-gateway-2:8080;
    server api-gateway-3:8080;
}

server {
    listen 443 ssl;
    server_name api.aetherium.io;

    ssl_certificate /etc/ssl/certs/aetherium.crt;
    ssl_certificate_key /etc/ssl/private/aetherium.key;

    location / {
        proxy_pass http://aetherium_api;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /logs/stream {
        proxy_pass http://aetherium_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

---

## Troubleshooting

### Common Issues

1. **VM Creation Fails**
   - Check KVM access: `ls -l /dev/kvm`
   - Verify vsock module: `lsmod | grep vhost_vsock`
   - Check rootfs: `ls -lh /var/firecracker/rootfs.ext4`

2. **Worker Not Processing Tasks**
   - Check Redis connection: `redis-cli -h localhost ping`
   - Verify queue registration: Check worker logs
   - Test database: `psql -h localhost -U aetherium -c '\dt'`

3. **API Gateway Timeout**
   - Increase timeout: `TIMEOUT=120s`
   - Check database pool: Increase max connections
   - Monitor resource usage: `htop`, `iotop`

### Debug Mode

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Restart services
systemctl restart aetherium-api-gateway aetherium-worker

# View logs
journalctl -u aetherium-api-gateway -f
journalctl -u aetherium-worker -f
```

---

## Production Checklist

- [ ] PostgreSQL configured with proper credentials
- [ ] Redis configured with password
- [ ] Firecracker installed and KVM accessible
- [ ] Rootfs prepared with default tools
- [ ] TLS certificates configured
- [ ] Firewall rules in place
- [ ] Monitoring and logging configured
- [ ] Backup automation configured
- [ ] Integration secrets configured
- [ ] Load balancer configured (if using)
- [ ] Health checks passing
- [ ] Documentation updated
