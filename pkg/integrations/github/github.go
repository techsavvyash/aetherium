package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aetherium/aetherium/pkg/integrations"
	"github.com/aetherium/aetherium/pkg/types"
)

// GitHubIntegration implements the Integration interface for GitHub
type GitHubIntegration struct {
	config *Config
	client *http.Client
}

// Config holds GitHub integration configuration
type Config struct {
	Token         string // Personal access token or GitHub App token
	WebhookSecret string // Secret for webhook validation
	BaseURL       string // GitHub API base URL (default: https://api.github.com)
}

// NewGitHubIntegration creates a new GitHub integration
func NewGitHubIntegration() *GitHubIntegration {
	return &GitHubIntegration{
		client: &http.Client{},
	}
}

// Name returns the unique name of the integration
func (g *GitHubIntegration) Name() string {
	return "github"
}

// Initialize initializes the integration with configuration
func (g *GitHubIntegration) Initialize(ctx context.Context, config integrations.Config) error {
	g.config = &Config{}

	if token, ok := config.Options["token"].(string); ok {
		g.config.Token = token
	}

	if secret, ok := config.Options["webhook_secret"].(string); ok {
		g.config.WebhookSecret = secret
	}

	if baseURL, ok := config.Options["base_url"].(string); ok {
		g.config.BaseURL = baseURL
	} else {
		g.config.BaseURL = "https://api.github.com"
	}

	if g.config.Token == "" {
		return fmt.Errorf("github token is required")
	}

	return nil
}

// HandleEvent processes an event from the event bus
func (g *GitHubIntegration) HandleEvent(ctx context.Context, event *types.Event) error {
	switch event.Type {
	case "task.completed":
		return g.handleTaskCompleted(ctx, event)
	case "task.failed":
		return g.handleTaskFailed(ctx, event)
	default:
		// Ignore other events
		return nil
	}
}

// handleTaskCompleted handles task completion events
func (g *GitHubIntegration) handleTaskCompleted(ctx context.Context, event *types.Event) error {
	// Extract task data
	taskID, _ := event.Data["task_id"].(string)
	prURL, _ := event.Data["pull_request"].(string)

	if prURL == "" {
		return nil // No PR associated
	}

	// Post success comment
	comment := fmt.Sprintf("✅ Task %s completed successfully!", taskID)
	return g.postComment(ctx, prURL, comment)
}

// handleTaskFailed handles task failure events
func (g *GitHubIntegration) handleTaskFailed(ctx context.Context, event *types.Event) error {
	taskID, _ := event.Data["task_id"].(string)
	errorMsg, _ := event.Data["error"].(string)
	prURL, _ := event.Data["pull_request"].(string)

	if prURL == "" {
		return nil
	}

	comment := fmt.Sprintf("❌ Task %s failed: %s", taskID, errorMsg)
	return g.postComment(ctx, prURL, comment)
}

// SendNotification sends a notification via this integration
func (g *GitHubIntegration) SendNotification(ctx context.Context, notification *types.Notification) error {
	switch notification.Type {
	case "pr_comment":
		prURL, _ := notification.Data["pr_url"].(string)
		return g.postComment(ctx, prURL, notification.Message)
	case "issue_comment":
		issueURL, _ := notification.Data["issue_url"].(string)
		return g.postComment(ctx, issueURL, notification.Message)
	default:
		return fmt.Errorf("unknown notification type: %s", notification.Type)
	}
}

// CreateArtifact creates an output artifact (e.g., PR, issue, message)
func (g *GitHubIntegration) CreateArtifact(ctx context.Context, artifact *types.Artifact) error {
	switch artifact.Type {
	case "pull_request":
		return g.createPullRequest(ctx, artifact.Content)
	case "issue":
		return g.createIssue(ctx, artifact.Content)
	default:
		return fmt.Errorf("unknown artifact type: %s", artifact.Type)
	}
}

// createPullRequest creates a new pull request
func (g *GitHubIntegration) createPullRequest(ctx context.Context, data map[string]interface{}) error {
	owner, _ := data["owner"].(string)
	repo, _ := data["repo"].(string)
	title, _ := data["title"].(string)
	body, _ := data["body"].(string)
	head, _ := data["head"].(string) // branch to merge from
	base, _ := data["base"].(string) // branch to merge into

	if owner == "" || repo == "" || title == "" || head == "" || base == "" {
		return fmt.Errorf("missing required fields for PR creation")
	}

	prData := map[string]interface{}{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls", g.config.BaseURL, owner, repo)
	return g.makeRequest(ctx, "POST", url, prData, nil)
}

// createIssue creates a new issue
func (g *GitHubIntegration) createIssue(ctx context.Context, data map[string]interface{}) error {
	owner, _ := data["owner"].(string)
	repo, _ := data["repo"].(string)
	title, _ := data["title"].(string)
	body, _ := data["body"].(string)

	if owner == "" || repo == "" || title == "" {
		return fmt.Errorf("missing required fields for issue creation")
	}

	issueData := map[string]interface{}{
		"title": title,
		"body":  body,
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.config.BaseURL, owner, repo)
	return g.makeRequest(ctx, "POST", url, issueData, nil)
}

// postComment posts a comment on a PR or issue
func (g *GitHubIntegration) postComment(ctx context.Context, issueURL string, comment string) error {
	// Extract owner, repo, and issue number from URL
	// URL format: https://github.com/owner/repo/pull/123 or https://github.com/owner/repo/issues/123
	parts := strings.Split(strings.TrimPrefix(issueURL, "https://github.com/"), "/")
	if len(parts) < 4 {
		return fmt.Errorf("invalid issue/PR URL")
	}

	owner := parts[0]
	repo := parts[1]
	number := parts[3]

	commentData := map[string]interface{}{
		"body": comment,
	}

	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s/comments", g.config.BaseURL, owner, repo, number)
	return g.makeRequest(ctx, "POST", url, commentData, nil)
}

// makeRequest makes an authenticated request to GitHub API
func (g *GitHubIntegration) makeRequest(ctx context.Context, method, url string, body interface{}, result interface{}) error {
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

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.config.Token))
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Health returns the health status of the integration
func (g *GitHubIntegration) Health(ctx context.Context) error {
	// Test API access by getting user info
	url := fmt.Sprintf("%s/user", g.config.BaseURL)
	return g.makeRequest(ctx, "GET", url, nil, nil)
}

// Close closes the integration connection
func (g *GitHubIntegration) Close() error {
	// No persistent connections to close
	return nil
}

// VerifyWebhookSignature verifies a GitHub webhook signature
func (g *GitHubIntegration) VerifyWebhookSignature(payload []byte, signature string) bool {
	// TODO: Implement HMAC verification
	// For now, just check if secret is configured
	return g.config.WebhookSecret != ""
}

// ParseWebhookPayload parses a GitHub webhook payload
func (g *GitHubIntegration) ParseWebhookPayload(eventType string, payload []byte) (*integrations.WebhookPayload, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}

	return &integrations.WebhookPayload{
		Integration: "github",
		Type:        eventType,
		Data:        data,
	}, nil
}
