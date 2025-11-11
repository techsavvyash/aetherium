package discord

import (
	"context"
	"fmt"
	"strings"

	"github.com/aetherium/aetherium/pkg/flows"
	"github.com/aetherium/aetherium/pkg/service"
)

// CommandHandler handles Discord slash commands with access to services
type CommandHandler struct {
	discord      *DiscordIntegration
	taskService  *service.TaskService
	flowExecutor *flows.FlowExecutor
}

// NewCommandHandler creates a new Discord command handler
func NewCommandHandler(discord *DiscordIntegration, taskService *service.TaskService, flowExecutor *flows.FlowExecutor) *CommandHandler {
	return &CommandHandler{
		discord:      discord,
		taskService:  taskService,
		flowExecutor: flowExecutor,
	}
}

// HandleCommand processes a Discord slash command interaction
func (h *CommandHandler) HandleCommand(ctx context.Context, interaction map[string]interface{}) (map[string]interface{}, error) {
	data := interaction["data"].(map[string]interface{})
	commandName := data["name"].(string)

	// Get subcommand if present
	var subcommand string
	var options map[string]interface{}

	if opts, ok := data["options"].([]interface{}); ok && len(opts) > 0 {
		firstOption := opts[0].(map[string]interface{})
		subcommand = firstOption["name"].(string)

		// Parse subcommand options
		options = make(map[string]interface{})
		if subOpts, ok := firstOption["options"].([]interface{}); ok {
			for _, opt := range subOpts {
				optMap := opt.(map[string]interface{})
				options[optMap["name"].(string)] = optMap["value"]
			}
		}
	}

	switch commandName {
	case "aether":
		return h.handleAetherCommand(ctx, subcommand, options)
	default:
		return h.buildErrorResponse(fmt.Sprintf("Unknown command: %s", commandName)), nil
	}
}

// handleAetherCommand handles the /aether command with subcommands
func (h *CommandHandler) handleAetherCommand(ctx context.Context, subcommand string, options map[string]interface{}) (map[string]interface{}, error) {
	switch subcommand {
	case "status":
		return h.handleStatusCommand(ctx)
	case "vms":
		return h.handleVMListCommand(ctx)
	case "flows":
		return h.handleListFlowsCommand(ctx)
	case "flow":
		return h.handleFlowCommand(ctx, options)
	default:
		return h.buildErrorResponse(fmt.Sprintf("Unknown subcommand: %s", subcommand)), nil
	}
}

// handleStatusCommand shows cluster status
func (h *CommandHandler) handleStatusCommand(ctx context.Context) (map[string]interface{}, error) {
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

	embed := map[string]interface{}{
		"title": "🚀 Aetherium Status",
		"color": 0x00ff00, // Green
		"fields": []map[string]interface{}{
			{
				"name":   "Total VMs",
				"value":  fmt.Sprintf("%d", len(vms)),
				"inline": true,
			},
			{
				"name":   "Running VMs",
				"value":  fmt.Sprintf("%d", runningVMs),
				"inline": true,
			},
		},
	}

	return h.buildEmbedResponse(embed), nil
}

// handleVMListCommand lists all VMs
func (h *CommandHandler) handleVMListCommand(ctx context.Context) (map[string]interface{}, error) {
	vms, err := h.taskService.ListVMs(ctx)
	if err != nil {
		return h.buildErrorResponse(fmt.Sprintf("Failed to list VMs: %v", err)), nil
	}

	if len(vms) == 0 {
		return h.buildSimpleResponse("No VMs found"), nil
	}

	fields := make([]map[string]interface{}, 0)

	for _, vm := range vms {
		statusEmoji := "🟢"
		if vm.Status != "running" {
			statusEmoji = "🔴"
		}

		fields = append(fields, map[string]interface{}{
			"name": fmt.Sprintf("%s %s", statusEmoji, vm.Name),
			"value": fmt.Sprintf("**ID:** `%s`\n**Resources:** %d vCPUs, %d MB RAM\n**Status:** %s",
				vm.ID, vm.VCPUCount, vm.MemoryMB, vm.Status),
			"inline": false,
		})
	}

	embed := map[string]interface{}{
		"title":  "📋 Virtual Machines",
		"color":  0x0099ff, // Blue
		"fields": fields,
	}

	return h.buildEmbedResponse(embed), nil
}

// handleListFlowsCommand lists available flows
func (h *CommandHandler) handleListFlowsCommand(ctx context.Context) (map[string]interface{}, error) {
	flows := h.flowExecutor.ListFlows()

	if len(flows) == 0 {
		return h.buildSimpleResponse("No flows available"), nil
	}

	fields := make([]map[string]interface{}, 0)

	for _, flow := range flows {
		paramsList := ""
		if len(flow.Parameters) > 0 {
			params := make([]string, 0, len(flow.Parameters))
			for _, param := range flow.Parameters {
				required := ""
				if param.Required {
					required = " (required)"
				}
				params = append(params, fmt.Sprintf("• `%s`%s: %s", param.Name, required, param.Description))
			}
			paramsList = "\n**Parameters:**\n" + strings.Join(params, "\n")
		}

		fields = append(fields, map[string]interface{}{
			"name":   fmt.Sprintf("%s (`%s`)", flow.Name, flow.ID),
			"value":  flow.Description + paramsList,
			"inline": false,
		})
	}

	embed := map[string]interface{}{
		"title":       "🔄 Available Flows",
		"description": "Use `/aether flow <flow-id> params:<key=value>` to run a flow",
		"color":       0xff9900, // Orange
		"fields":      fields,
	}

	return h.buildEmbedResponse(embed), nil
}

// handleFlowCommand triggers a flow execution
func (h *CommandHandler) handleFlowCommand(ctx context.Context, options map[string]interface{}) (map[string]interface{}, error) {
	flowID, ok := options["flow_id"].(string)
	if !ok || flowID == "" {
		return h.buildErrorResponse("flow_id is required"), nil
	}

	params := make(map[string]interface{})

	// Parse parameters (key=value format)
	if paramsStr, ok := options["params"].(string); ok && paramsStr != "" {
		parts := strings.Fields(paramsStr)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				params[kv[0]] = kv[1]
			}
		}
	}

	execution, err := h.flowExecutor.ExecuteFlow(ctx, flowID, params, "discord-user", "discord-channel")
	if err != nil {
		return h.buildErrorResponse(fmt.Sprintf("Failed to execute flow: %v", err)), nil
	}

	embed := map[string]interface{}{
		"title":       fmt.Sprintf("✅ Flow `%s` started", flowID),
		"color":       0x00ff00, // Green
		"description": fmt.Sprintf("**Execution ID:** `%s`\n**Status:** %s", execution.ID.String(), execution.Status),
	}

	return h.buildEmbedResponse(embed), nil
}

// Response builders

func (h *CommandHandler) buildSimpleResponse(content string) map[string]interface{} {
	return map[string]interface{}{
		"type": ResponseTypeChannelMessageWithSource,
		"data": map[string]interface{}{
			"content": content,
		},
	}
}

func (h *CommandHandler) buildEmbedResponse(embed map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"type": ResponseTypeChannelMessageWithSource,
		"data": map[string]interface{}{
			"embeds": []map[string]interface{}{embed},
		},
	}
}

func (h *CommandHandler) buildErrorResponse(message string) map[string]interface{} {
	return map[string]interface{}{
		"type": ResponseTypeChannelMessageWithSource,
		"data": map[string]interface{}{
			"embeds": []map[string]interface{}{
				{
					"title":       "❌ Error",
					"description": message,
					"color":       0xff0000, // Red
				},
			},
		},
	}
}
