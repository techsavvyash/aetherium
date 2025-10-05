# Integrations Guide

Aetherium provides extensible integration framework for connecting with external platforms.

## Overview

The integration system consists of:
1. **Integration Registry** - Manages registered integrations
2. **Event Bus** - Pub/sub for cross-component communication
3. **Integration Interface** - Standard API for all integrations
4. **Built-in Integrations** - GitHub, Slack (more coming)

---

## Architecture

```
┌──────────────┐     ┌─────────────┐     ┌──────────────┐
│  Event Bus   │────▶│  Registry   │────▶│ Integration  │
│  (Redis)     │     │             │     │  (GitHub)    │
└──────────────┘     └─────────────┘     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │ Integration  │
                     │  (Slack)     │
                     └──────────────┘
```

---

## GitHub Integration

### Setup

1. **Create GitHub App or Personal Access Token**

   **Option A: Personal Access Token (Quick Start)**
   - Go to GitHub Settings → Developer settings → Personal access tokens
   - Generate new token with scopes: `repo`, `write:discussion`
   - Copy token

   **Option B: GitHub App (Production)**
   - Go to GitHub Settings → Developer settings → GitHub Apps
   - Create new GitHub App
   - Set permissions: Repository (Read & Write), Pull Requests (Read & Write)
   - Install app on repositories
   - Generate private key

2. **Configure Aetherium**

   ```bash
   export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
   export GITHUB_WEBHOOK_SECRET=your_webhook_secret
   ```

   Or in `config/production.yaml`:
   ```yaml
   integrations:
     github:
       enabled: true
       token: ${GITHUB_TOKEN}
       webhook_secret: ${GITHUB_WEBHOOK_SECRET}
   ```

3. **Start API Gateway**

   ```bash
   ./bin/api-gateway
   ```

   The GitHub integration will auto-register on startup.

### Features

#### Automatic PR Comments

When a task completes/fails, comment is posted to associated PR:

```go
// Task completion triggers PR comment
event := &types.Event{
    Type: "task.completed",
    Data: map[string]interface{}{
        "task_id": "uuid",
        "pull_request": "https://github.com/owner/repo/pull/123",
    },
}
```

Result:
```
✅ Task uuid completed successfully!
```

#### Create Pull Requests

Via API:
```bash
curl -X POST http://localhost:8080/api/v1/integrations/github/pr \
  -H "Content-Type: application/json" \
  -d '{
    "owner": "myorg",
    "repo": "myrepo",
    "title": "Implement feature X",
    "body": "Automated PR from Aetherium",
    "head": "feature/x",
    "base": "main"
  }'
```

Via Service:
```go
artifact := &types.Artifact{
    Type: "pull_request",
    Content: map[string]interface{}{
        "owner": "myorg",
        "repo": "myrepo",
        "title": "Implement feature X",
        "body": "...",
        "head": "feature/x",
        "base": "main",
    },
}

err := githubIntegration.CreateArtifact(ctx, artifact)
```

#### Webhook Handling

Configure webhook in GitHub:
- Payload URL: `https://your-domain.com/api/v1/webhooks/github`
- Content type: `application/json`
- Secret: Your webhook secret
- Events: Pull request, Issue comment

Supported events:
- `pull_request` - PR opened, closed, merged
- `issue_comment` - Comment on PR/issue
- `push` - Code pushed to branch

### Event Subscriptions

Subscribe to GitHub events:

```go
eventBus.Subscribe(ctx, "github.pull_request.opened", func(ctx context.Context, event *types.Event) error {
    prNumber := event.Data["number"].(float64)
    prURL := event.Data["url"].(string)

    log.Printf("New PR #%.0f: %s", prNumber, prURL)

    // Create VM and run tests
    taskID, _ := taskService.CreateVMTask(ctx, fmt.Sprintf("pr-%.0f", prNumber), 2, 2048)

    return nil
})
```

---

## Slack Integration

### Setup

1. **Create Slack App**
   - Go to https://api.slack.com/apps
   - Create new app
   - Add OAuth scopes:
     - `chat:write` - Send messages
     - `commands` - Slash commands
     - `im:write` - Direct messages
   - Install app to workspace
   - Copy Bot User OAuth Token

2. **Configure Slash Commands**
   - Create slash commands:
     - `/aetherium-status` - Check system status
     - `/aetherium-vm-list` - List VMs
     - `/aetherium-task` - Create task
   - Set Request URL: `https://your-domain.com/api/v1/webhooks/slack`

3. **Enable Event Subscriptions**
   - Set Request URL: `https://your-domain.com/api/v1/webhooks/slack`
   - Subscribe to events:
     - `message.channels` - Channel messages
     - `app_mention` - App mentions

4. **Configure Aetherium**

   ```bash
   export SLACK_BOT_TOKEN=xoxb-xxxxxxxxxxxxx
   export SLACK_SIGNING_SECRET=xxxxxxxxxxxxx
   ```

### Features

#### Send Notifications

```go
notification := &types.Notification{
    Type: "message",
    Target: "#deployments",
    Message: "🚀 New VM created: my-vm",
}

err := slackIntegration.SendNotification(ctx, notification)
```

#### Interactive Messages

```go
slackIntegration.SendInteractiveMessage(
    ctx,
    "#general",
    "Task completed. What next?",
    []slack.Action{
        {Text: "Approve", ActionID: "approve", Value: "task-uuid"},
        {Text: "Reject", ActionID: "reject", Value: "task-uuid"},
    },
)
```

#### Event Notifications

Auto-notify on events:

```go
eventBus.Subscribe(ctx, "vm.created", func(ctx context.Context, event *types.Event) error {
    vmID := event.Data["vm_id"].(string)
    vmName := event.Data["vm_name"].(string)

    notification := &types.Notification{
        Type: "message",
        Target: "#vms",
        Message: fmt.Sprintf("🚀 New VM: `%s` (ID: `%s`)", vmName, vmID),
    }

    return slackIntegration.SendNotification(ctx, notification)
})
```

### Slash Commands

#### `/aetherium-status`

Returns system health:
```
/aetherium-status

Response:
Aetherium is running! 🚀
- GitHub: ✅ healthy
- Loki: ✅ healthy
- Event Bus: ✅ healthy
```

#### `/aetherium-vm-list`

Lists active VMs:
```
/aetherium-vm-list

Response:
Active VMs:
• my-vm (uuid) - running
• test-vm (uuid) - running
Total: 2 VMs
```

#### `/aetherium-task`

Creates a task:
```
/aetherium-task create vm name=quick-test vcpus=2

Response:
Creating task: create vm name=quick-test vcpus=2
Task ID: uuid
```

---

## Event Bus

### Redis Event Bus

Production-ready pub/sub:

```go
// Initialize
eventBus, err := redis.NewRedisEventBus(&redis.Config{
    Addr: "localhost:6379",
})

// Publish event
event := &types.Event{
    ID: uuid.New().String(),
    Type: "vm.created",
    Timestamp: time.Now(),
    Data: map[string]interface{}{
        "vm_id": "uuid",
        "vm_name": "my-vm",
    },
}

eventBus.Publish(ctx, "vm.created", event)

// Subscribe
subscriptionID, err := eventBus.Subscribe(ctx, "vm.created", func(ctx context.Context, event *types.Event) error {
    log.Printf("VM created: %s", event.Data["vm_id"])
    return nil
})
```

### Event Topics

Standard topics:
- `task.created` - Task submitted
- `task.started` - Task processing started
- `task.completed` - Task completed successfully
- `task.failed` - Task failed
- `vm.created` - VM created
- `vm.started` - VM started
- `vm.stopped` - VM stopped
- `vm.failed` - VM creation failed
- `integration.webhook_received` - Webhook received

Custom topics:
```go
eventBus.Publish(ctx, "custom.my_event", event)
eventBus.Subscribe(ctx, "custom.my_event", handler)
```

---

## Custom Integrations

### Create Integration

1. **Implement Interface**

```go
package myintegration

import (
    "context"
    "github.com/aetherium/aetherium/pkg/integrations"
    "github.com/aetherium/aetherium/pkg/types"
)

type MyIntegration struct {
    config *Config
}

type Config struct {
    APIKey string
    BaseURL string
}

func New() *MyIntegration {
    return &MyIntegration{}
}

func (m *MyIntegration) Name() string {
    return "myintegration"
}

func (m *MyIntegration) Initialize(ctx context.Context, config integrations.Config) error {
    m.config = &Config{
        APIKey: config.Options["api_key"].(string),
        BaseURL: config.Options["base_url"].(string),
    }
    return nil
}

func (m *MyIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
    // Process event
    return nil
}

func (m *MyIntegration) SendNotification(ctx context.Context, notification *types.Notification) error {
    // Send notification
    return nil
}

func (m *MyIntegration) CreateArtifact(ctx context.Context, artifact *types.Artifact) error {
    // Create artifact
    return nil
}

func (m *MyIntegration) Health(ctx context.Context) error {
    // Health check
    return nil
}

func (m *MyIntegration) Close() error {
    return nil
}
```

2. **Register Integration**

```go
registry := integrations.NewRegistry()

myInt := myintegration.New()
err := myInt.Initialize(ctx, integrations.Config{
    Options: map[string]interface{}{
        "api_key": "xxx",
        "base_url": "https://api.example.com",
    },
})

registry.Register(myInt)
```

3. **Subscribe to Events**

```go
eventBus.Subscribe(ctx, "task.completed", func(ctx context.Context, event *types.Event) error {
    integration, _ := registry.Get("myintegration")
    return integration.HandleEvent(ctx, event)
})
```

---

## Configuration

### Full Integration Config

```yaml
# config/production.yaml

integrations:
  github:
    enabled: true
    token: ${GITHUB_TOKEN}
    webhook_secret: ${GITHUB_WEBHOOK_SECRET}
    base_url: https://api.github.com

  slack:
    enabled: true
    bot_token: ${SLACK_BOT_TOKEN}
    signing_secret: ${SLACK_SIGNING_SECRET}
    base_url: https://slack.com/api

  # Custom integration
  myintegration:
    enabled: true
    api_key: ${MY_INTEGRATION_KEY}
    base_url: https://api.example.com

events:
  provider: redis
  redis:
    addr: localhost:6379
    db: 1
```

---

## Monitoring

### Health Checks

```bash
curl http://localhost:8080/api/v1/health

{
  "status": "ok",
  "components": {
    "github": "healthy",
    "slack": "healthy",
    "event_bus": "healthy"
  }
}
```

### Integration Logs

Query integration activity:

```bash
POST /logs/query
{
  "component": "github",
  "level": "info",
  "limit": 100
}
```

### Event Metrics

Track event throughput:

```go
// Prometheus metrics
event_bus_messages_published_total{topic="vm.created"}
event_bus_messages_consumed_total{topic="vm.created",integration="slack"}
integration_requests_total{integration="github",method="create_pr"}
```

---

## Best Practices

### Error Handling

```go
func (g *GitHubIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic in GitHub integration: %v", r)
        }
    }()

    if err := g.processEvent(ctx, event); err != nil {
        // Log but don't fail - integration errors shouldn't break main flow
        log.Printf("GitHub integration error: %v", err)
        return nil // Return nil to prevent retry
    }

    return nil
}
```

### Rate Limiting

```go
// Rate limit API calls
limiter := rate.NewLimiter(rate.Every(time.Second), 5) // 5 req/sec

func (g *GitHubIntegration) createPullRequest(...) error {
    if err := limiter.Wait(ctx); err != nil {
        return err
    }

    // Make API call
    return g.makeRequest(...)
}
```

### Idempotency

```go
// Use event IDs to prevent duplicate processing
processedEvents := make(map[string]bool)

func (m *MyIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
    if processedEvents[event.ID] {
        return nil // Already processed
    }

    // Process event
    // ...

    processedEvents[event.ID] = true
    return nil
}
```

---

## Troubleshooting

### GitHub Integration Not Working

1. **Check token permissions**
   ```bash
   curl -H "Authorization: Bearer $GITHUB_TOKEN" \
        https://api.github.com/user
   ```

2. **Verify webhook secret**
   ```bash
   # Check signature validation in logs
   grep "webhook signature" /var/log/aetherium/api-gateway.log
   ```

3. **Test PR creation manually**
   ```bash
   curl -X POST http://localhost:8080/api/v1/integrations/github/pr \
        -d '{"owner":"test","repo":"test","title":"Test",...}'
   ```

### Slack Integration Not Responding

1. **Test bot token**
   ```bash
   curl -X POST https://slack.com/api/auth.test \
        -H "Authorization: Bearer $SLACK_BOT_TOKEN"
   ```

2. **Check channel permissions**
   - Ensure bot is invited to channel
   - Verify `chat:write` scope

3. **Validate signing secret**
   ```bash
   # Logs should show signature verification
   grep "slack signature" /var/log/aetherium/api-gateway.log
   ```

### Event Bus Issues

1. **Check Redis connection**
   ```bash
   redis-cli -h localhost ping
   ```

2. **Monitor event flow**
   ```bash
   redis-cli MONITOR | grep "PUBLISH\|SUBSCRIBE"
   ```

3. **Debug subscriptions**
   ```bash
   redis-cli PUBSUB CHANNELS
   ```
