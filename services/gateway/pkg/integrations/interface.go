package integrations

import (
	"context"

	"github.com/aetherium/aetherium/libs/types/pkg/domain"
)

// Integration defines the interface for integration implementations
type Integration interface {
	// Name returns the unique name of the integration
	Name() string

	// Initialize initializes the integration with configuration
	Initialize(ctx context.Context, config Config) error

	// HandleEvent processes an event from the event bus
	HandleEvent(ctx context.Context, event *types.Event) error

	// SendNotification sends a notification via this integration
	SendNotification(ctx context.Context, notification *types.Notification) error

	// CreateArtifact creates an output artifact (e.g., PR, issue, message)
	CreateArtifact(ctx context.Context, artifact *types.Artifact) error

	// Health returns the health status of the integration
	Health(ctx context.Context) error

	// Close closes the integration connection
	Close() error
}

// Config represents integration configuration
type Config struct {
	// Integration-specific configuration
	Options map[string]interface{}
}

// WebhookPayload represents a webhook payload from an integration
type WebhookPayload struct {
	Integration string                 `json:"integration"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Signature   string                 `json:"signature,omitempty"`
}
