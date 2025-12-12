package container

import (
	"context"
	"fmt"
	"sync"

	"github.com/aetherium/aetherium/libs/common/pkg/config"
	"github.com/aetherium/aetherium/libs/common/pkg/events"
	"github.com/aetherium/aetherium/services/gateway/pkg/integrations"
	"github.com/aetherium/aetherium/libs/common/pkg/logging"
	"github.com/aetherium/aetherium/services/core/pkg/queue"
	"github.com/aetherium/aetherium/services/core/pkg/storage"
	"github.com/aetherium/aetherium/services/core/pkg/vmm"
)

// Container is a dependency injection container for managing component lifecycle
type Container struct {
	config *config.Config

	// Singletons
	taskQueue    queue.TaskQueue
	stateStore   storage.StateStore
	logger       logging.Logger
	vmOrch       vmm.VMOrchestrator
	eventBus     events.EventBus
	integrations map[string]integrations.Integration

	// Factories
	taskQueueFactory    TaskQueueFactory
	stateStoreFactory   StateStoreFactory
	loggerFactory       LoggerFactory
	vmOrchestratorFactory VMOrchestratorFactory
	eventBusFactory     EventBusFactory
	integrationRegistry *integrations.Registry

	mu sync.RWMutex
}

// New creates a new dependency injection container
func New(cfg *config.Config) *Container {
	return &Container{
		config:       cfg,
		integrations: make(map[string]integrations.Integration),
		integrationRegistry: integrations.NewRegistry(),
	}
}

// RegisterTaskQueueFactory registers a factory for task queue providers
func (c *Container) RegisterTaskQueueFactory(factory TaskQueueFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.taskQueueFactory = factory
}

// RegisterStateStoreFactory registers a factory for state store providers
func (c *Container) RegisterStateStoreFactory(factory StateStoreFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateStoreFactory = factory
}

// RegisterLoggerFactory registers a factory for logger providers
func (c *Container) RegisterLoggerFactory(factory LoggerFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggerFactory = factory
}

// RegisterVMOrchestratorFactory registers a factory for VMM orchestrator providers
func (c *Container) RegisterVMOrchestratorFactory(factory VMOrchestratorFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vmOrchestratorFactory = factory
}

// RegisterEventBusFactory registers a factory for event bus providers
func (c *Container) RegisterEventBusFactory(factory EventBusFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventBusFactory = factory
}

// RegisterIntegration registers an integration plugin
func (c *Container) RegisterIntegration(integration integrations.Integration) error {
	return c.integrationRegistry.Register(integration)
}

// Initialize initializes all components based on configuration
func (c *Container) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Initialize TaskQueue
	if err := c.initTaskQueue(ctx); err != nil {
		return fmt.Errorf("failed to initialize task queue: %w", err)
	}

	// Initialize StateStore
	if err := c.initStateStore(ctx); err != nil {
		return fmt.Errorf("failed to initialize state store: %w", err)
	}

	// Initialize Logger
	if err := c.initLogger(ctx); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize VMOrchestrator
	if err := c.initVMOrchestrator(ctx); err != nil {
		return fmt.Errorf("failed to initialize VM orchestrator: %w", err)
	}

	// Initialize EventBus
	if err := c.initEventBus(ctx); err != nil {
		return fmt.Errorf("failed to initialize event bus: %w", err)
	}

	// Initialize Integrations
	if err := c.initIntegrations(ctx); err != nil {
		return fmt.Errorf("failed to initialize integrations: %w", err)
	}

	return nil
}

func (c *Container) initTaskQueue(ctx context.Context) error {
	if c.taskQueueFactory == nil {
		return fmt.Errorf("task queue factory not registered")
	}

	provider := c.config.TaskQueue.Provider
	providerConfig := c.config.TaskQueue.Config

	tq, err := c.taskQueueFactory.Create(ctx, provider, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create task queue provider '%s': %w", provider, err)
	}

	c.taskQueue = tq
	return nil
}

func (c *Container) initStateStore(ctx context.Context) error {
	if c.stateStoreFactory == nil {
		return fmt.Errorf("state store factory not registered")
	}

	provider := c.config.Storage.Provider
	providerConfig := c.config.Storage.Config

	store, err := c.stateStoreFactory.Create(ctx, provider, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create state store provider '%s': %w", provider, err)
	}

	c.stateStore = store
	return nil
}

func (c *Container) initLogger(ctx context.Context) error {
	if c.loggerFactory == nil {
		return fmt.Errorf("logger factory not registered")
	}

	provider := c.config.Logging.Provider
	providerConfig := c.config.Logging.Config

	logger, err := c.loggerFactory.Create(ctx, provider, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create logger provider '%s': %w", provider, err)
	}

	c.logger = logger
	return nil
}

func (c *Container) initVMOrchestrator(ctx context.Context) error {
	if c.vmOrchestratorFactory == nil {
		return fmt.Errorf("VM orchestrator factory not registered")
	}

	provider := c.config.VMM.Provider
	providerConfig := c.config.VMM.Config

	orch, err := c.vmOrchestratorFactory.Create(ctx, provider, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create VM orchestrator provider '%s': %w", provider, err)
	}

	c.vmOrch = orch
	return nil
}

func (c *Container) initEventBus(ctx context.Context) error {
	if c.eventBusFactory == nil {
		return fmt.Errorf("event bus factory not registered")
	}

	provider := c.config.EventBus.Provider
	providerConfig := c.config.EventBus.Config

	bus, err := c.eventBusFactory.Create(ctx, provider, providerConfig)
	if err != nil {
		return fmt.Errorf("failed to create event bus provider '%s': %w", provider, err)
	}

	c.eventBus = bus
	return nil
}

func (c *Container) initIntegrations(ctx context.Context) error {
	for _, name := range c.config.Integrations.Enabled {
		integration, err := c.integrationRegistry.Get(name)
		if err != nil {
			return fmt.Errorf("integration '%s' not found: %w", name, err)
		}

		integrationConfig, err := c.config.GetIntegrationConfig(name)
		if err != nil {
			return fmt.Errorf("failed to get config for integration '%s': %w", name, err)
		}

		cfg := integrations.Config{Options: integrationConfig}
		if err := integration.Initialize(ctx, cfg); err != nil {
			return fmt.Errorf("failed to initialize integration '%s': %w", name, err)
		}

		c.integrations[name] = integration
	}

	return nil
}

// GetTaskQueue returns the task queue instance
func (c *Container) GetTaskQueue() queue.TaskQueue {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.taskQueue
}

// GetStateStore returns the state store instance
func (c *Container) GetStateStore() storage.StateStore {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stateStore
}

// GetLogger returns the logger instance
func (c *Container) GetLogger() logging.Logger {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.logger
}

// GetVMOrchestrator returns the VM orchestrator instance
func (c *Container) GetVMOrchestrator() vmm.VMOrchestrator {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.vmOrch
}

// GetEventBus returns the event bus instance
func (c *Container) GetEventBus() events.EventBus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.eventBus
}

// GetIntegration returns a specific integration instance
func (c *Container) GetIntegration(name string) (integrations.Integration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	integration, ok := c.integrations[name]
	if !ok {
		return nil, fmt.Errorf("integration '%s' not initialized", name)
	}

	return integration, nil
}

// Shutdown gracefully shuts down all components
func (c *Container) Shutdown(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors []error

	// Shutdown integrations
	for name, integration := range c.integrations {
		if err := integration.Close(); err != nil {
			errors = append(errors, fmt.Errorf("integration '%s': %w", name, err))
		}
	}

	// Shutdown other components if they implement Shutdown
	// This would require adding Shutdown methods to interfaces in future

	if len(errors) > 0 {
		return fmt.Errorf("shutdown errors: %v", errors)
	}

	return nil
}
