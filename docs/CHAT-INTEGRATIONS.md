# Chat Platform Integrations

Aetherium supports triggering workflows and managing VMs directly from Slack and Discord. This document explains how to set up and use these integrations.

## Table of Contents

1. [Overview](#overview)
2. [Flows System](#flows-system)
3. [Slack Integration](#slack-integration)
4. [Discord Integration](#discord-integration)
5. [Available Commands](#available-commands)
6. [Built-in Flows](#built-in-flows)
7. [Custom Flows](#custom-flows)
8. [Examples](#examples)
9. [Troubleshooting](#troubleshooting)

## Overview

### What Can You Do?

- **Trigger Flows**: Execute pre-defined multi-step workflows
- **Manage VMs**: Create, list, and monitor virtual machines
- **Check Status**: View cluster statistics and health
- **Execute Commands**: Run commands in VMs

### Supported Platforms

- **Slack**: Slash commands and interactive messages
- **Discord**: Slash commands with rich embeds

## Flows System

Flows are multi-step workflows that can be triggered from chat platforms. Each flow consists of:

- **Steps**: Individual actions (create VM, execute command, wait, etc.)
- **Parameters**: Configurable inputs
- **Dependencies**: Steps that must complete before others

### Flow Step Types

| Type | Description | Example |
|------|-------------|---------|
| `create_vm` | Create and start a VM | Create a 2 vCPU, 2GB VM with Go installed |
| `execute_cmd` | Execute command in VM | Run `git clone` or `npm install` |
| `delete_vm` | Stop and delete VM | Clean up resources after testing |
| `wait` | Wait for duration | Wait 30 seconds for service to start |
| `notify` | Send notification | Alert when flow completes |

## Slack Integration

### Setup

#### 1. Create Slack App

1. Go to [https://api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App" → "From scratch"
3. Name it "Aetherium Bot" and select your workspace

#### 2. Configure Bot Permissions

Go to **OAuth & Permissions** and add these Bot Token Scopes:

- `chat:write` - Send messages
- `commands` - Create slash commands
- `im:write` - Send direct messages

#### 3. Create Slash Commands

Go to **Slash Commands** and create:

| Command | Request URL | Description |
|---------|-------------|-------------|
| `/aether` | `https://your-domain.com/webhooks/slack` | Main Aetherium command |
| `/aether-status` | `https://your-domain.com/webhooks/slack` | Show cluster status |
| `/aether-vms` | `https://your-domain.com/webhooks/slack` | List VMs |
| `/aether-flow` | `https://your-domain.com/webhooks/slack` | Execute flow |

#### 4. Enable Interactivity

Go to **Interactivity & Shortcuts**:

- Enable Interactivity
- Request URL: `https://your-domain.com/slack/interactions`

#### 5. Install to Workspace

1. Go to **Install App**
2. Click "Install to Workspace"
3. Copy the **Bot User OAuth Token**

#### 6. Configure Aetherium

Set environment variables:

```bash
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
export SLACK_SIGNING_SECRET="your-signing-secret"
```

Or add to `.env`:

```env
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_SIGNING_SECRET=your-signing-secret
```

### Usage

#### Basic Commands

```bash
# Show cluster status
/aether status

# List all VMs
/aether vms

# List available flows
/aether flows

# Show help
/aether help
```

#### Execute Flows

```bash
# Quick test with command
/aether flow quick-test command=node args="--version"

# Clone and build repository
/aether flow git-build repo_url=https://github.com/user/repo

# Run benchmark
/aether flow benchmark benchmark_command="npm run bench"
```

#### Interactive Buttons

When you list flows with `/aether flows`, each flow has a "Run" button. Click it to execute the flow with default parameters.

## Discord Integration

### Setup

#### 1. Create Discord Application

1. Go to [https://discord.com/developers/applications](https://discord.com/developers/applications)
2. Click "New Application"
3. Name it "Aetherium"

#### 2. Create Bot

1. Go to **Bot** tab
2. Click "Add Bot"
3. Copy the **Bot Token**
4. Enable "Message Content Intent" (if needed)

#### 3. Get Application Credentials

From **General Information**:

- Copy **Application ID**
- Copy **Public Key**

#### 4. Configure Interactions Endpoint

1. Go to **General Information**
2. Set **Interactions Endpoint URL**: `https://your-domain.com/webhooks/discord`
3. Discord will verify the endpoint (must be publicly accessible)

#### 5. Invite Bot to Server

1. Go to **OAuth2** → **URL Generator**
2. Select scopes: `bot`, `applications.commands`
3. Select bot permissions: `Send Messages`, `Use Slash Commands`
4. Copy generated URL and open in browser
5. Select server and authorize

#### 6. Configure Aetherium

Set environment variables:

```bash
export DISCORD_BOT_TOKEN="your-bot-token"
export DISCORD_APPLICATION_ID="your-application-id"
export DISCORD_PUBLIC_KEY="your-public-key"
```

Or add to `.env`:

```env
DISCORD_BOT_TOKEN=your-bot-token
DISCORD_APPLICATION_ID=your-application-id
DISCORD_PUBLIC_KEY=your-public-key
```

#### 7. Restart API Gateway

The Discord slash commands will be automatically registered when the API Gateway starts.

### Usage

Discord uses native slash commands with autocomplete:

```bash
# Show cluster status
/aether status

# List all VMs
/aether vms

# List available flows
/aether flows

# Execute flow
/aether flow flow_id:quick-test params:"command=node"
```

All responses use rich embeds with colors and formatting.

## Available Commands

### Status Command

Shows cluster statistics:

**Slack**: `/aether status`
**Discord**: `/aether status`

**Output**:
- Total VMs
- Running VMs
- Pending tasks

### List VMs Command

Shows all virtual machines with details:

**Slack**: `/aether vms`
**Discord**: `/aether vms`

**Output** for each VM:
- Name and status
- VM ID
- vCPU count and memory
- Created timestamp

### List Flows Command

Shows available flows with parameters:

**Slack**: `/aether flows`
**Discord**: `/aether flows`

**Output** for each flow:
- Flow ID and name
- Description
- Required and optional parameters
- Interactive "Run" button (Slack only)

### Execute Flow Command

Triggers a flow execution:

**Slack**: `/aether flow <flow-id> [key=value ...]`
**Discord**: `/aether flow flow_id:<flow-id> params:"key=value ..."`

**Parameters**:
- Flow-specific parameters in `key=value` format
- Multiple parameters separated by spaces

**Output**:
- Execution ID
- Flow ID
- Status (pending/running)

## Built-in Flows

### 1. Quick Test

Execute a command in a fresh VM and clean up.

**ID**: `quick-test`

**Parameters**:
- `command` (required): Command to execute
- `args` (optional): Command arguments

**Steps**:
1. Create VM (1 vCPU, 512 MB, with Git and Node.js)
2. Execute command
3. Delete VM

**Example**:
```bash
# Slack
/aether flow quick-test command=node args="--version"

# Discord
/aether flow flow_id:quick-test params:"command=node args=--version"

# Via API
curl -X POST http://localhost:8080/api/v1/flows/quick-test/execute \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {
      "command": "node",
      "args": ["--version"]
    }
  }'
```

### 2. Git Clone and Build

Clone a repository and run build commands.

**ID**: `git-build`

**Parameters**:
- `repo_url` (required): Git repository URL
- `build_command` (optional): Build command (default: `npm install && npm run build`)

**Steps**:
1. Create VM (2 vCPUs, 2 GB, with Git, Node.js, and Bun)
2. Clone repository
3. Run build command
4. Delete VM

**Example**:
```bash
# Slack
/aether flow git-build repo_url=https://github.com/vercel/next.js

# With custom build
/aether flow git-build repo_url=https://github.com/user/repo build_command="pnpm install && pnpm build"

# Discord
/aether flow flow_id:git-build params:"repo_url=https://github.com/vercel/next.js"
```

### 3. Benchmark

Run a benchmark command in an isolated VM.

**ID**: `benchmark`

**Parameters**:
- `benchmark_command` (required): Benchmark command to execute

**Steps**:
1. Create VM (2 vCPUs, 1 GB, with Git, Node.js, and Python)
2. Execute benchmark
3. Delete VM

**Example**:
```bash
# Slack
/aether flow benchmark benchmark_command="npm run bench"

# Discord
/aether flow flow_id:benchmark params:"benchmark_command=npm run bench"
```

## Custom Flows

You can create custom flows by defining them in code or via API (future).

### Flow Definition Structure

```go
&flows.FlowDefinition{
    ID:          "my-flow",
    Name:        "My Custom Flow",
    Description: "Description of what this flow does",
    Parameters: []flows.FlowParameter{
        {
            Name:        "param1",
            Description: "Parameter description",
            Required:    true,
        },
        {
            Name:        "param2",
            Description: "Optional parameter",
            Required:    false,
            Default:     "default-value",
        },
    },
    Steps: []flows.FlowStep{
        {
            ID:   "step1",
            Name: "Step 1 Name",
            Type: flows.StepTypeCreateVM,
            Config: map[string]interface{}{
                "vcpus":     2,
                "memory_mb": 2048,
                "tools":     []string{"go", "python"},
            },
        },
        {
            ID:        "step2",
            Name:      "Step 2 Name",
            Type:      flows.StepTypeExecuteCmd,
            DependsOn: []string{"step1"},
            Config: map[string]interface{}{
                "command": "go",
                "args":    []string{"version"},
            },
        },
        {
            ID:              "step3",
            Name:            "Cleanup",
            Type:            flows.StepTypeDeleteVM,
            DependsOn:       []string{"step2"},
            ContinueOnError: true,
        },
    },
}
```

### Registering Custom Flows

In your API Gateway initialization:

```go
// After creating flowExecutor
customFlow := &flows.FlowDefinition{ /* ... */ }
flowExecutor.RegisterFlow(customFlow)
```

## Examples

### Example 1: Test Node.js Version

**Slack**:
```
/aether flow quick-test command=node args="--version"
```

**Response**:
```
✅ Flow quick-test started
Execution ID: 550e8400-e29b-41d4-a716-446655440000
Status: running
```

### Example 2: Clone and Test Repository

**Slack**:
```
/aether flow git-build repo_url=https://github.com/user/my-app build_command="npm test"
```

**Discord**:
```
/aether flow flow_id:git-build params:"repo_url=https://github.com/user/my-app build_command=npm test"
```

### Example 3: List VMs and Status

**Slack**:
```
/aether vms
```

**Response**:
```
📋 Virtual Machines

🟢 dev-vm-1
ID: vm-123456
Resources: 2 vCPUs, 2048 MB RAM
Status: running

🔴 test-vm-2
ID: vm-789012
Resources: 1 vCPUs, 512 MB RAM
Status: stopped
```

### Example 4: Check Cluster Status

**Discord**:
```
/aether status
```

**Response** (rich embed):
```
🚀 Aetherium Status

Total VMs: 5
Running VMs: 3
```

## Troubleshooting

### Slack Issues

#### "Webhook URL didn't respond in time"

**Cause**: Aetherium took too long to respond

**Solution**:
- Flow executions are async, response is immediate
- Check Aetherium API Gateway logs
- Verify Aetherium is publicly accessible

#### "Command not found"

**Cause**: Slash command not registered or wrong URL

**Solution**:
- Verify slash commands are created in Slack app settings
- Check Request URL matches your deployment
- Ensure Aetherium API Gateway is running

#### "Verification failed"

**Cause**: Signing secret mismatch

**Solution**:
- Check `SLACK_SIGNING_SECRET` matches Slack app
- Restart API Gateway after changing secrets

### Discord Issues

#### "Interaction Failed"

**Cause**: Aetherium didn't respond or verification failed

**Solution**:
- Verify `DISCORD_PUBLIC_KEY` is correct
- Check Interactions Endpoint URL is accessible
- View Discord Developer Portal → Your App → Interactions for errors

#### "Application did not respond"

**Cause**: Command handler error

**Solution**:
- Check Aetherium API Gateway logs
- Verify Discord integration is initialized
- Test endpoint: `curl https://your-domain.com/webhooks/discord`

#### "Unknown command"

**Cause**: Commands not registered

**Solution**:
- Restart API Gateway (commands register on startup)
- Check logs for "Discord commands registered"
- Verify bot has `applications.commands` scope

### General Issues

#### "Integration not initialized"

**Cause**: Missing environment variables

**Solution**:
```bash
# Check configuration
env | grep SLACK
env | grep DISCORD

# Restart with correct values
export SLACK_BOT_TOKEN="xoxb-..."
export DISCORD_BOT_TOKEN="..."
./bin/api-gateway
```

#### "Flow not found"

**Cause**: Invalid flow ID

**Solution**:
- List flows: `/aether flows`
- Use exact flow ID from list
- Check for typos

#### "Failed to execute flow"

**Cause**: Invalid parameters or missing required params

**Solution**:
- Check required parameters with `/aether flows`
- Verify parameter format: `key=value`
- View flow details via API: `GET /api/v1/flows/{id}`

## REST API Endpoints

All chat commands also have equivalent REST API endpoints:

### List Flows

```bash
GET /api/v1/flows

Response:
{
  "flows": [
    {
      "id": "quick-test",
      "name": "Quick Test",
      "description": "...",
      "parameters": [...]
    }
  ],
  "total": 3
}
```

### Get Flow Details

```bash
GET /api/v1/flows/{id}

Response:
{
  "id": "quick-test",
  "name": "Quick Test",
  "steps": [...],
  "parameters": [...]
}
```

### Execute Flow

```bash
POST /api/v1/flows/{id}/execute
Content-Type: application/json

{
  "parameters": {
    "command": "node",
    "args": "--version"
  },
  "trigger_by": "user-123"
}

Response: 202 Accepted
{
  "execution_id": "550e8400-e29b-41d4-a716-446655440000",
  "flow_id": "quick-test",
  "status": "pending",
  "started_at": "2025-11-11T10:00:00Z"
}
```

## Security Considerations

### Webhook Verification

- **Slack**: Requests are verified using signing secret (HMAC-SHA256)
- **Discord**: Requests are verified using Ed25519 signature

### Authentication

Currently, chat integrations have no user-level authentication. Anyone in the workspace/server can:
- View cluster status
- List VMs
- Execute flows

**For production**:
- Restrict slash commands to specific channels
- Implement user permission checks
- Use Slack Enterprise Grid or Discord role permissions

### Rate Limiting

- Implement rate limiting per user/channel
- Set maximum concurrent flows per user
- Monitor for abuse patterns

## Next Steps

1. **Create Custom Flows**: Define workflows specific to your use case
2. **Add Notifications**: Integrate flow completion notifications
3. **Implement Auth**: Add user-level permissions
4. **Monitor Usage**: Track flow executions and performance
5. **Add More Platforms**: Microsoft Teams, Mattermost, etc.

## Resources

- [Slack API Documentation](https://api.slack.com/)
- [Discord Developer Portal](https://discord.com/developers/docs)
- [Aetherium API Reference](./API-GATEWAY.md)
- [Flow System Guide](./FLOWS.md) (to be created)
