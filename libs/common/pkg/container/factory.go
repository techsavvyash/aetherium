package container

import (
	"context"

	"github.com/aetherium/aetherium/libs/common/pkg/events"
	"github.com/aetherium/aetherium/libs/common/pkg/logging"
	"github.com/aetherium/aetherium/services/core/pkg/queue"
	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/aetherium/aetherium/services/core/pkg/vmm"
)

// TaskQueueFactory creates TaskQueue instances based on provider name
type TaskQueueFactory interface {
	Create(ctx context.Context, provider string, config map[string]interface{}) (queue.TaskQueue, error)
	SupportedProviders() []string
}

// StateStoreFactory creates StateStore instances based on provider name
type StateStoreFactory interface {
	Create(ctx context.Context, provider string, config map[string]interface{}) (storage.StateStore, error)
	SupportedProviders() []string
}

// LoggerFactory creates Logger instances based on provider name
type LoggerFactory interface {
	Create(ctx context.Context, provider string, config map[string]interface{}) (logging.Logger, error)
	SupportedProviders() []string
}

// VMOrchestratorFactory creates VMOrchestrator instances based on provider name
type VMOrchestratorFactory interface {
	Create(ctx context.Context, provider string, config map[string]interface{}) (vmm.VMOrchestrator, error)
	SupportedProviders() []string
}

// EventBusFactory creates EventBus instances based on provider name
type EventBusFactory interface {
	Create(ctx context.Context, provider string, config map[string]interface{}) (events.EventBus, error)
	SupportedProviders() []string
}
