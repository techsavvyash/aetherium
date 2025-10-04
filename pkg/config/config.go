package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Server       ServerConfig              `yaml:"server"`
	TaskQueue    ProviderConfig            `yaml:"task_queue"`
	Storage      ProviderConfig            `yaml:"storage"`
	Logging      ProviderConfig            `yaml:"logging"`
	VMM          ProviderConfig            `yaml:"vmm"`
	EventBus     ProviderConfig            `yaml:"event_bus"`
	Integrations IntegrationsConfig        `yaml:"integrations"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// ProviderConfig represents a pluggable component configuration
type ProviderConfig struct {
	Provider string                 `yaml:"provider"`
	Config   map[string]interface{} `yaml:"config"`
}

// IntegrationsConfig represents integrations configuration
type IntegrationsConfig struct {
	Enabled []string                      `yaml:"enabled"`
	Configs map[string]map[string]interface{} `yaml:",inline"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := expandEnvVars(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expanded), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.TaskQueue.Provider == "" {
		return fmt.Errorf("task queue provider is required")
	}

	if c.Storage.Provider == "" {
		return fmt.Errorf("storage provider is required")
	}

	if c.Logging.Provider == "" {
		return fmt.Errorf("logging provider is required")
	}

	if c.VMM.Provider == "" {
		return fmt.Errorf("vmm provider is required")
	}

	if c.EventBus.Provider == "" {
		return fmt.Errorf("event bus provider is required")
	}

	return nil
}

// expandEnvVars expands ${VAR} and $VAR patterns in the config
func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}

// GetString safely retrieves a string value from a config map
func GetString(config map[string]interface{}, key string) (string, error) {
	val, exists := config[key]
	if !exists {
		return "", fmt.Errorf("key %s not found", key)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %s is not a string", key)
	}

	return str, nil
}

// GetInt safely retrieves an int value from a config map
func GetInt(config map[string]interface{}, key string) (int, error) {
	val, exists := config[key]
	if !exists {
		return 0, fmt.Errorf("key %s not found", key)
	}

	// Handle both int and float64 (JSON unmarshaling)
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("key %s is not an integer", key)
	}
}

// GetBool safely retrieves a bool value from a config map
func GetBool(config map[string]interface{}, key string) (bool, error) {
	val, exists := config[key]
	if !exists {
		return false, fmt.Errorf("key %s not found", key)
	}

	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("key %s is not a boolean", key)
	}

	return b, nil
}

// GetStringOrDefault retrieves a string with a default value
func GetStringOrDefault(config map[string]interface{}, key, defaultVal string) string {
	val, err := GetString(config, key)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetIntOrDefault retrieves an int with a default value
func GetIntOrDefault(config map[string]interface{}, key string, defaultVal int) int {
	val, err := GetInt(config, key)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetBoolOrDefault retrieves a bool with a default value
func GetBoolOrDefault(config map[string]interface{}, key string, defaultVal bool) bool {
	val, err := GetBool(config, key)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetProviderConfig retrieves the configuration for a specific provider
func (c *Config) GetProviderConfig(providerType, providerName string) (map[string]interface{}, error) {
	var providerConfig ProviderConfig

	switch strings.ToLower(providerType) {
	case "task_queue", "queue":
		providerConfig = c.TaskQueue
	case "storage", "store":
		providerConfig = c.Storage
	case "logging", "logger", "log":
		providerConfig = c.Logging
	case "vmm", "vm":
		providerConfig = c.VMM
	case "event_bus", "events":
		providerConfig = c.EventBus
	default:
		return nil, fmt.Errorf("unknown provider type: %s", providerType)
	}

	if providerConfig.Provider != providerName {
		return nil, fmt.Errorf("provider mismatch: expected %s, got %s", providerName, providerConfig.Provider)
	}

	if providerConfig.Config == nil {
		return make(map[string]interface{}), nil
	}

	// Get provider-specific config
	if specificConfig, ok := providerConfig.Config[providerName].(map[string]interface{}); ok {
		return specificConfig, nil
	}

	return providerConfig.Config, nil
}

// GetIntegrationConfig retrieves configuration for a specific integration
func (c *Config) GetIntegrationConfig(name string) (map[string]interface{}, error) {
	// Check if integration is enabled
	enabled := false
	for _, enabled_name := range c.Integrations.Enabled {
		if enabled_name == name {
			enabled = true
			break
		}
	}

	if !enabled {
		return nil, fmt.Errorf("integration %s is not enabled", name)
	}

	if config, ok := c.Integrations.Configs[name]; ok {
		return config, nil
	}

	return make(map[string]interface{}), nil
}
