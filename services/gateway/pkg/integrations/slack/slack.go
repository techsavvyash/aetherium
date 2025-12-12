package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aetherium/aetherium/services/gateway/pkg/integrations"
	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// SlackIntegration implements the Integration interface for Slack
type SlackIntegration struct {
	config *Config
	client *http.Client
}

// Config holds Slack integration configuration
type Config struct {
	BotToken      string // Slack Bot User OAuth Token
	SigningSecret string // Slack signing secret for webhook validation
	BaseURL       string // Slack API base URL (default: https://slack.com/api)
}

// NewSlackIntegration creates a new Slack integration
func NewSlackIntegration() *SlackIntegration {
	return &SlackIntegration{
		client: &http.Client{},
	}
}

// Name returns the unique name of the integration
func (s *SlackIntegration) Name() string {
	return "slack"
}

// Initialize initializes the integration with configuration
func (s *SlackIntegration) Initialize(ctx context.Context, config integrations.Config) error {
	s.config = &Config{}

	if token, ok := config.Options["bot_token"].(string); ok {
		s.config.BotToken = token
	}

	if secret, ok := config.Options["signing_secret"].(string); ok {
		s.config.SigningSecret = secret
	}

	if baseURL, ok := config.Options["base_url"].(string); ok {
		s.config.BaseURL = baseURL
	} else {
		s.config.BaseURL = "https://slack.com/api"
	}

	if s.config.BotToken == "" {
		return fmt.Errorf("slack bot token is required")
	}

	return nil
}

// HandleEvent processes an event from the event bus
func (s *SlackIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
	switch event.Type {
	case "task.completed":
		return s.handleTaskCompleted(ctx, event)
	case "task.failed":
		return s.handleTaskFailed(ctx, event)
	case "vm.created":
		return s.handleVMCreated(ctx, event)
	default:
		// Ignore other events
		return nil
	}
}

// handleTaskCompleted handles task completion events
func (s *SlackIntegration) handleTaskCompleted(ctx context.Context, event *types.Event) error {
	taskID, _ := event.Data["task_id"].(string)
	channel, _ := event.Data["slack_channel"].(string)

	if channel == "" {
		return nil // No channel specified
	}

	message := fmt.Sprintf("✅ Task `%s` completed successfully!", taskID)
	return s.sendMessage(ctx, channel, message, nil)
}

// handleTaskFailed handles task failure events
func (s *SlackIntegration) handleTaskFailed(ctx context.Context, event *types.Event) error {
	taskID, _ := event.Data["task_id"].(string)
	errorMsg, _ := event.Data["error"].(string)
	channel, _ := event.Data["slack_channel"].(string)

	if channel == "" {
		return nil
	}

	message := fmt.Sprintf("❌ Task `%s` failed: %s", taskID, errorMsg)
	return s.sendMessage(ctx, channel, message, nil)
}

// handleVMCreated handles VM creation events
func (s *SlackIntegration) handleVMCreated(ctx context.Context, event *types.Event) error {
	vmID, _ := event.Data["vm_id"].(string)
	vmName, _ := event.Data["vm_name"].(string)
	channel, _ := event.Data["slack_channel"].(string)

	if channel == "" {
		return nil
	}

	message := fmt.Sprintf("🚀 New VM created: `%s` (ID: `%s`)", vmName, vmID)
	return s.sendMessage(ctx, channel, message, nil)
}

// SendNotification sends a notification via this integration
func (s *SlackIntegration) SendNotification(ctx context.Context, notification *types.Notification) error {
	channel := notification.Target

	// Build blocks for rich message
	var blocks []map[string]interface{}
	if notification.Data != nil {
		if b, ok := notification.Data["blocks"].([]map[string]interface{}); ok {
			blocks = b
		}
	}

	return s.sendMessage(ctx, channel, notification.Message, blocks)
}

// CreateArtifact creates an output artifact (not applicable for Slack)
func (s *SlackIntegration) CreateArtifact(ctx context.Context, artifact *types.Artifact) error {
	return fmt.Errorf("artifact creation not supported for slack")
}

// sendMessage sends a message to a Slack channel
func (s *SlackIntegration) sendMessage(ctx context.Context, channel string, text string, blocks []map[string]interface{}) error {
	payload := map[string]interface{}{
		"channel": channel,
		"text":    text,
	}

	if len(blocks) > 0 {
		payload["blocks"] = blocks
	}

	url := fmt.Sprintf("%s/chat.postMessage", s.config.BaseURL)
	return s.makeRequest(ctx, "POST", url, payload, nil)
}

// SendInteractiveMessage sends an interactive message with buttons/actions
func (s *SlackIntegration) SendInteractiveMessage(ctx context.Context, channel string, text string, actions []Action) error {
	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": text,
			},
		},
	}

	if len(actions) > 0 {
		elements := make([]map[string]interface{}, 0, len(actions))
		for _, action := range actions {
			elements = append(elements, map[string]interface{}{
				"type":      "button",
				"text":      map[string]string{"type": "plain_text", "text": action.Text},
				"action_id": action.ActionID,
				"value":     action.Value,
			})
		}

		blocks = append(blocks, map[string]interface{}{
			"type":     "actions",
			"elements": elements,
		})
	}

	return s.sendMessage(ctx, channel, text, blocks)
}

// Action represents a Slack interactive button/action
type Action struct {
	Text     string
	ActionID string
	Value    string
}

// makeRequest makes an authenticated request to Slack API
func (s *SlackIntegration) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
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

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.BotToken))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Check Slack's ok field
	var slackResp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &slackResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !slackResp.OK {
		return fmt.Errorf("slack API error: %s", slackResp.Error)
	}

	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Health returns the health status of the integration
func (s *SlackIntegration) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/auth.test", s.config.BaseURL)
	return s.makeRequest(ctx, "POST", url, nil, nil)
}

// Close closes the integration connection
func (s *SlackIntegration) Close() error {
	// No persistent connections to close
	return nil
}

// VerifySlackRequest verifies a Slack request signature
func (s *SlackIntegration) VerifySlackRequest(timestamp, signature string, body []byte) bool {
	// TODO: Implement HMAC verification using signing secret
	// For now, just check if secret is configured
	return s.config.SigningSecret != ""
}

// HandleSlashCommand handles a Slack slash command
func (s *SlackIntegration) HandleSlashCommand(ctx context.Context, command string, text string, userID string, channelID string) (string, error) {
	switch command {
	case "/aetherium-status":
		return "Aetherium is running! 🚀", nil
	case "/aetherium-vm-list":
		// TODO: Query VMs and return list
		return "VM list functionality coming soon!", nil
	case "/aetherium-task":
		// TODO: Create task from slash command
		return fmt.Sprintf("Creating task: %s", text), nil
	default:
		return fmt.Sprintf("Unknown command: %s", command), nil
	}
}
