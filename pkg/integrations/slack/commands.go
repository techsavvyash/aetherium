package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/aetherium/aetherium/pkg/flows"
	"github.com/aetherium/aetherium/pkg/service"
)

// CommandHandler handles Slack slash commands with access to services
type CommandHandler struct {
	slack        *SlackIntegration
	taskService  *service.TaskService
	flowExecutor *flows.FlowExecutor
}

// NewCommandHandler creates a new Slack command handler
func NewCommandHandler(slack *SlackIntegration, taskService *service.TaskService, flowExecutor *flows.FlowExecutor) *CommandHandler {
	return &CommandHandler{
		slack:        slack,
		taskService:  taskService,
		flowExecutor: flowExecutor,
	}
}

// HandleCommand processes a slash command and returns a response
func (h *CommandHandler) HandleCommand(ctx context.Context, command string, text string, userID string, channelID string) (interface{}, error) {
	switch command {
	case "/aetherium", "/aether":
		return h.handleMainCommand(ctx, text, userID, channelID)
	case "/aetherium-status", "/aether-status":
		return h.handleStatusCommand(ctx, userID, channelID)
	case "/aetherium-vm-list", "/aether-vms":
		return h.handleVMListCommand(ctx, userID, channelID)
	case "/aetherium-flow", "/aether-flow":
		return h.handleFlowCommand(ctx, text, userID, channelID)
	case "/aetherium-help", "/aether-help":
		return h.handleHelpCommand(ctx, userID, channelID)
	default:
		return h.buildErrorResponse(fmt.Sprintf("Unknown command: %s", command)), nil
	}
}

// handleMainCommand handles the main /aetherium command with subcommands
func (h *CommandHandler) handleMainCommand(ctx context.Context, text string, userID string, channelID string) (interface{}, error) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return h.handleHelpCommand(ctx, userID, channelID)
	}

	subcommand := parts[0]
	args := parts[1:]

	switch subcommand {
	case "status":
		return h.handleStatusCommand(ctx, userID, channelID)
	case "vms", "vm-list":
		return h.handleVMListCommand(ctx, userID, channelID)
	case "flow", "run":
		return h.handleFlowCommand(ctx, strings.Join(args, " "), userID, channelID)
	case "flows", "list-flows":
		return h.handleListFlowsCommand(ctx, userID, channelID)
	case "help":
		return h.handleHelpCommand(ctx, userID, channelID)
	default:
		return h.buildErrorResponse(fmt.Sprintf("Unknown subcommand: %s", subcommand)), nil
	}
}

// handleStatusCommand shows cluster status
func (h *CommandHandler) handleStatusCommand(ctx context.Context, userID string, channelID string) (interface{}, error) {
	vms, err := h.taskService.ListVMs(ctx)
	if err != nil {
		return h.buildErrorResponse(fmt.Sprintf("Failed to get status: %v", err)), nil
	}

	runningVMs := 0
	for _, vm := range vms {
		if vm.Status == "running" {
			runningVMs++
		}
	}

	return h.buildSuccessResponse(
		"🚀 *Aetherium Status*",
		[]map[string]interface{}{
			{
				"type": "section",
				"fields": []map[string]interface{}{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Total VMs:*\n%d", len(vms)),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Running VMs:*\n%d", runningVMs),
					},
				},
			},
		},
	), nil
}

// handleVMListCommand lists all VMs
func (h *CommandHandler) handleVMListCommand(ctx context.Context, userID string, channelID string) (interface{}, error) {
	vms, err := h.taskService.ListVMs(ctx)
	if err != nil {
		return h.buildErrorResponse(fmt.Sprintf("Failed to list VMs: %v", err)), nil
	}

	if len(vms) == 0 {
		return h.buildSimpleResponse("No VMs found"), nil
	}

	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "📋 Virtual Machines",
			},
		},
	}

	for _, vm := range vms {
		statusEmoji := "🟢"
		if vm.Status != "running" {
			statusEmoji = "🔴"
		}

		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"fields": []map[string]interface{}{
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Name:*\n%s", vm.Name),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Status:*\n%s %s", statusEmoji, vm.Status),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Resources:*\n%d vCPUs, %d MB RAM", vm.VCPUCount, vm.MemoryMB),
				},
				{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*ID:*\n`%s`", vm.ID),
				},
			},
		})
		blocks = append(blocks, map[string]interface{}{
			"type": "divider",
		})
	}

	return map[string]interface{}{
		"response_type": "in_channel",
		"blocks":        blocks,
	}, nil
}

// handleFlowCommand triggers a flow execution
func (h *CommandHandler) handleFlowCommand(ctx context.Context, text string, userID string, channelID string) (interface{}, error) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return h.buildErrorResponse("Usage: /aether flow <flow-id> [params...]"), nil
	}

	flowID := parts[0]
	params := make(map[string]interface{})

	// Parse parameters (key=value format)
	for i := 1; i < len(parts); i++ {
		kv := strings.SplitN(parts[i], "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = kv[1]
		}
	}

	execution, err := h.flowExecutor.ExecuteFlow(ctx, flowID, params, userID, channelID)
	if err != nil {
		return h.buildErrorResponse(fmt.Sprintf("Failed to execute flow: %v", err)), nil
	}

	return h.buildSuccessResponse(
		fmt.Sprintf("✅ Flow `%s` started", flowID),
		[]map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Execution ID:* `%s`\n*Status:* %s", execution.ID.String(), execution.Status),
				},
			},
		},
	), nil
}

// handleListFlowsCommand lists available flows
func (h *CommandHandler) handleListFlowsCommand(ctx context.Context, userID string, channelID string) (interface{}, error) {
	flows := h.flowExecutor.ListFlows()

	if len(flows) == 0 {
		return h.buildSimpleResponse("No flows available"), nil
	}

	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": "🔄 Available Flows",
			},
		},
	}

	for _, flow := range flows {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*%s* (`%s`)\n%s", flow.Name, flow.ID, flow.Description),
			},
			"accessory": map[string]interface{}{
				"type":      "button",
				"text":      map[string]string{"type": "plain_text", "text": "Run"},
				"action_id": "run_flow",
				"value":     flow.ID,
			},
		})

		// Add parameters info
		if len(flow.Parameters) > 0 {
			paramTexts := make([]string, 0, len(flow.Parameters))
			for _, param := range flow.Parameters {
				required := ""
				if param.Required {
					required = " (required)"
				}
				paramTexts = append(paramTexts, fmt.Sprintf("• `%s`%s: %s", param.Name, required, param.Description))
			}
			blocks = append(blocks, map[string]interface{}{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "*Parameters:*\n" + strings.Join(paramTexts, "\n"),
				},
			})
		}

		blocks = append(blocks, map[string]interface{}{
			"type": "divider",
		})
	}

	return map[string]interface{}{
		"response_type": "in_channel",
		"blocks":        blocks,
	}, nil
}

// handleHelpCommand shows help information
func (h *CommandHandler) handleHelpCommand(ctx context.Context, userID string, channelID string) (interface{}, error) {
	return h.buildHelpResponse(), nil
}

// Response builders

func (h *CommandHandler) buildSimpleResponse(text string) map[string]interface{} {
	return map[string]interface{}{
		"response_type": "ephemeral",
		"text":          text,
	}
}

func (h *CommandHandler) buildSuccessResponse(title string, blocks []map[string]interface{}) map[string]interface{} {
	allBlocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": title,
			},
		},
	}
	allBlocks = append(allBlocks, blocks...)

	return map[string]interface{}{
		"response_type": "in_channel",
		"blocks":        allBlocks,
	}
}

func (h *CommandHandler) buildErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"response_type": "ephemeral",
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("❌ *Error:* %s", message),
				},
			},
		},
	}
}

func (h *CommandHandler) buildHelpResponse() map[string]interface{} {
	return map[string]interface{}{
		"response_type": "ephemeral",
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type": "plain_text",
					"text": "🤖 Aetherium Help",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "*Available Commands:*",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "`/aether status` - Show cluster status\n" +
						"`/aether vms` - List all virtual machines\n" +
						"`/aether flows` - List available flows\n" +
						"`/aether flow <flow-id> [params]` - Run a flow\n" +
						"`/aether help` - Show this help message",
				},
			},
			{
				"type": "divider",
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "*Examples:*",
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": "`/aether flow quick-test command=node args=\"--version\"`\n" +
						"`/aether flow git-build repo_url=https://github.com/user/repo`",
				},
			},
		},
	}
}
