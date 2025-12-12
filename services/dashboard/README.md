# Aetherium Dashboard

A Next.js dashboard for the Aetherium distributed task execution platform.

## Features

- **Dashboard Overview** - System health, VM counts, queue statistics
- **VM Management** - Create, view, and delete Firecracker microVMs
- **Command Execution** - Run commands inside VMs with output display
- **Workers Monitoring** - View worker instances and their task statistics
- **Task Scheduler** - Monitor task queue, filter by status and type
- **API Explorer** - Custom REST API requests with preset actions

## Getting Started

### Prerequisites

- Node.js 18+
- Aetherium API Gateway running (default: http://localhost:8080)

### Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Configuration

Set the API URL via environment variable:

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
```

For Kubernetes deployments, you can use the API Gateway service:

```bash
NEXT_PUBLIC_API_URL=http://aetherium-api-gateway.aetherium.svc.cluster.local:8080
```

### Build for Production

```bash
npm run build
npm start
```

## Pages

| Page | Path | Description |
|------|------|-------------|
| Dashboard | `/` | Overview with stats and recent VMs |
| VMs | `/vms` | Create and manage virtual machines |
| Workers | `/workers` | Monitor worker instances |
| Tasks | `/tasks` | View and filter task queue |
| API Explorer | `/api-explorer` | Test API endpoints |
| Settings | `/settings` | Configuration and health check |

## Tech Stack

- Next.js 15 with App Router
- TypeScript
- Tailwind CSS v4
- shadcn/ui components
- Lucide icons
