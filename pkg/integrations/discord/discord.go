package discord

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aetherium/aetherium/pkg/integrations"
	"github.com/aetherium/aetherium/pkg/types"
)

// DiscordIntegration implements the Integration interface for Discord
type DiscordIntegration struct {
	config *Config
	client *http.Client
}

// Config holds Discord integration configuration
type Config struct {
	BotToken      string // Discord Bot Token
	ApplicationID string // Discord Application ID
	PublicKey     string // Discord Application Public Key (for signature verification)
	BaseURL       string // Discord API base URL
}

// InteractionType represents Discord interaction types
type InteractionType int

const (
	InteractionTypePing               InteractionType = 1
	InteractionTypeApplicationCommand InteractionType = 2
	InteractionTypeMessageComponent   InteractionType = 3
	InteractionTypeModalSubmit        InteractionType = 5
)

// InteractionResponseType represents Discord interaction response types
type InteractionResponseType int

const (
	ResponseTypePong                     InteractionResponseType = 1
	ResponseTypeChannelMessageWithSource InteractionResponseType = 4
	ResponseTypeDeferredChannelMessage   InteractionResponseType = 5
	ResponseTypeDeferredUpdateMessage    InteractionResponseType = 6
	ResponseTypeUpdateMessage            InteractionResponseType = 7
)

// NewDiscordIntegration creates a new Discord integration
func NewDiscordIntegration() *DiscordIntegration {
	return &DiscordIntegration{
		client: &http.Client{},
	}
}

// Name returns the unique name of the integration
func (d *DiscordIntegration) Name() string {
	return "discord"
}

// Initialize initializes the integration with configuration
func (d *DiscordIntegration) Initialize(ctx context.Context, config integrations.Config) error {
	d.config = &Config{}

	if token, ok := config.Options["bot_token"].(string); ok {
		d.config.BotToken = token
	}

	if appID, ok := config.Options["application_id"].(string); ok {
		d.config.ApplicationID = appID
	}

	if publicKey, ok := config.Options["public_key"].(string); ok {
		d.config.PublicKey = publicKey
	}

	if baseURL, ok := config.Options["base_url"].(string); ok {
		d.config.BaseURL = baseURL
	} else {
		d.config.BaseURL = "https://discord.com/api/v10"
	}

	if d.config.BotToken == "" {
		return fmt.Errorf("discord bot token is required")
	}

	if d.config.ApplicationID == "" {
		return fmt.Errorf("discord application_id is required")
	}

	if d.config.PublicKey == "" {
		return fmt.Errorf("discord public_key is required")
	}

	return nil
}

// HandleEvent processes an event from the event bus
func (d *DiscordIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
	switch event.Type {
	case "task.completed":
		return d.handleTaskCompleted(ctx, event)
	case "task.failed":
		return d.handleTaskFailed(ctx, event)
	case "vm.created":
		return d.handleVMCreated(ctx, event)
	default:
		return nil
	}
}

// handleTaskCompleted handles task completion events
func (d *DiscordIntegration) handleTaskCompleted(ctx context.Context, event *types.Event) error {
	taskID, _ := event.Data["task_id"].(string)
	channelID, _ := event.Data["discord_channel"].(string)

	if channelID == "" {
		return nil
	}

	message := fmt.Sprintf("✅ Task `%s` completed successfully!", taskID)
	return d.sendMessage(ctx, channelID, message, nil)
}

// handleTaskFailed handles task failure events
func (d *DiscordIntegration) handleTaskFailed(ctx context.Context, event *types.Event) error {
	taskID, _ := event.Data["task_id"].(string)
	errorMsg, _ := event.Data["error"].(string)
	channelID, _ := event.Data["discord_channel"].(string)

	if channelID == "" {
		return nil
	}

	message := fmt.Sprintf("❌ Task `%s` failed: %s", taskID, errorMsg)
	return d.sendMessage(ctx, channelID, message, nil)
}

// handleVMCreated handles VM creation events
func (d *DiscordIntegration) handleVMCreated(ctx context.Context, event *types.Event) error {
	vmID, _ := event.Data["vm_id"].(string)
	vmName, _ := event.Data["vm_name"].(string)
	channelID, _ := event.Data["discord_channel"].(string)

	if channelID == "" {
		return nil
	}

	message := fmt.Sprintf("🚀 New VM created: `%s` (ID: `%s`)", vmName, vmID)
	return d.sendMessage(ctx, channelID, message, nil)
}

// SendNotification sends a notification via this integration
func (d *DiscordIntegration) SendNotification(ctx context.Context, notification *types.Notification) error {
	channelID := notification.Target

	var embeds []map[string]interface{}
	if notification.Data != nil {
		if e, ok := notification.Data["embeds"].([]map[string]interface{}); ok {
			embeds = e
		}
	}

	return d.sendMessage(ctx, channelID, notification.Message, embeds)
}

// CreateArtifact creates an output artifact (not applicable for Discord)
func (d *DiscordIntegration) CreateArtifact(ctx context.Context, artifact *types.Artifact) error {
	return fmt.Errorf("artifact creation not supported for discord")
}

// sendMessage sends a message to a Discord channel
func (d *DiscordIntegration) sendMessage(ctx context.Context, channelID string, content string, embeds []map[string]interface{}) error {
	payload := map[string]interface{}{
		"content": content,
	}

	if len(embeds) > 0 {
		payload["embeds"] = embeds
	}

	url := fmt.Sprintf("%s/channels/%s/messages", d.config.BaseURL, channelID)
	return d.makeRequest(ctx, "POST", url, payload, nil)
}

// SendEmbed sends a rich embed message
func (d *DiscordIntegration) SendEmbed(ctx context.Context, channelID string, embed map[string]interface{}) error {
	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}

	url := fmt.Sprintf("%s/channels/%s/messages", d.config.BaseURL, channelID)
	return d.makeRequest(ctx, "POST", url, payload, nil)
}

// makeRequest makes an authenticated request to Discord API
func (d *DiscordIntegration) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bot %s", d.config.BotToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Health returns the health status of the integration
func (d *DiscordIntegration) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/users/@me", d.config.BaseURL)
	return d.makeRequest(ctx, "GET", url, nil, nil)
}

// Close closes the integration connection
func (d *DiscordIntegration) Close() error {
	return nil
}

// VerifyInteraction verifies a Discord interaction signature
func (d *DiscordIntegration) VerifyInteraction(timestamp, signature string, body []byte) bool {
	publicKeyBytes, err := hex.DecodeString(d.config.PublicKey)
	if err != nil {
		return false
	}

	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	message := append([]byte(timestamp), body...)
	return ed25519.Verify(publicKeyBytes, message, signatureBytes)
}

// RegisterCommands registers slash commands with Discord
func (d *DiscordIntegration) RegisterCommands(ctx context.Context) error {
	commands := []map[string]interface{}{
		{
			"name":        "aether",
			"description": "Interact with Aetherium platform",
			"options": []map[string]interface{}{
				{
					"name":        "status",
					"description": "Show cluster status",
					"type":        1, // SUB_COMMAND
				},
				{
					"name":        "vms",
					"description": "List all virtual machines",
					"type":        1, // SUB_COMMAND
				},
				{
					"name":        "flows",
					"description": "List available flows",
					"type":        1, // SUB_COMMAND
				},
				{
					"name":        "flow",
					"description": "Run a flow",
					"type":        1, // SUB_COMMAND
					"options": []map[string]interface{}{
						{
							"name":        "flow_id",
							"description": "Flow ID to execute",
							"type":        3, // STRING
							"required":    true,
						},
						{
							"name":        "params",
							"description": "Flow parameters (key=value format)",
							"type":        3, // STRING
							"required":    false,
						},
					},
				},
			},
		},
	}

	url := fmt.Sprintf("%s/applications/%s/commands", d.config.BaseURL, d.config.ApplicationID)

	for _, command := range commands {
		if err := d.makeRequest(ctx, "POST", url, command, nil); err != nil {
			return fmt.Errorf("failed to register command %s: %w", command["name"], err)
		}
	}

	return nil
}

// HandleInteraction handles a Discord interaction (slash command or component)
// Returns a response that should be sent back to Discord
func (d *DiscordIntegration) HandleInteraction(ctx context.Context, interaction map[string]interface{}) (map[string]interface{}, error) {
	interactionType := int(interaction["type"].(float64))

	// Handle ping (for verification)
	if interactionType == int(InteractionTypePing) {
		return map[string]interface{}{
			"type": ResponseTypePong,
		}, nil
	}

	// Interactions are handled by the command handler
	return nil, nil
}
