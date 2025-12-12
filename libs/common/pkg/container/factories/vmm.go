package factories

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/services/core/pkg/vmm"
	"github.com/aetherium/aetherium/services/core/pkg/vmm/docker"
	"github.com/aetherium/aetherium/services/core/pkg/vmm/firecracker"
)

// DefaultVMMFactory creates VMOrchestrator instances
type DefaultVMMFactory struct{}

// NewVMMFactory creates a new VMM factory
func NewVMMFactory() *DefaultVMMFactory {
	return &DefaultVMMFactory{}
}

// Create creates a VMOrchestrator based on the provider name
func (f *DefaultVMMFactory) Create(ctx context.Context, provider string, config map[string]interface{}) (vmm.VMOrchestrator, error) {
	switch provider {
	case "docker":
		return docker.NewDockerOrchestrator(config)
	case "firecracker":
		return firecracker.NewFirecrackerOrchestrator(config)
	default:
		return nil, fmt.Errorf("unsupported VMM provider: %s", provider)
	}
}

// SupportedProviders returns list of supported providers
func (f *DefaultVMMFactory) SupportedProviders() []string {
	return []string{"docker", "firecracker"}
}
