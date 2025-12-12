package factories

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
	"github.com/aetherium/aetherium/libs/common/pkg/logging"
	"github.com/aetherium/aetherium/libs/common/pkg/logging/stdout"
)

// DefaultLoggerFactory creates Logger instances
type DefaultLoggerFactory struct{}

// NewLoggerFactory creates a new logger factory
func NewLoggerFactory() *DefaultLoggerFactory {
	return &DefaultLoggerFactory{}
}

// Create creates a Logger based on the provider name
func (f *DefaultLoggerFactory) Create(ctx context.Context, provider string, cfg map[string]interface{}) (logging.Logger, error) {
	switch provider {
	case "stdout":
		colorize := config.GetBoolOrDefault(cfg, "colorize", true)
		return stdout.NewStdoutLogger(colorize), nil
	case "loki":
		// TODO: Implement Loki logger
		return nil, fmt.Errorf("loki logger not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported logger provider: %s", provider)
	}
}

// SupportedProviders returns list of supported providers
func (f *DefaultLoggerFactory) SupportedProviders() []string {
	return []string{"stdout", "loki"}
}
