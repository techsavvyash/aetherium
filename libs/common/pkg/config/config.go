package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Redis        RedisConfig        `yaml:"redis"`
	Queue        QueueConfig        `yaml:"queue"`
	TaskQueue    TaskQueueConfig    `yaml:"task_queue"`
	Storage      StorageConfig      `yaml:"storage"`
	VMM          VMMConfig          `yaml:"vmm"`
	Network      NetworkConfig      `yaml:"network"`
	Logging      LoggingConfig      `yaml:"logging"`
	EventBus     EventBusConfig     `yaml:"event_bus"`
	Integrations IntegrationsConfig `yaml:"integrations"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // "development" or "production"
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	Database     string `yaml:"database"`
	SSLMode      string `yaml:"sslmode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	Concurrency int            `yaml:"concurrency"`
	Queues      map[string]int `yaml:"queues"`
}

// VMMConfig holds VMM configuration
type VMMConfig struct {
	DefaultOrchestrator string            `yaml:"default_orchestrator"` // "firecracker" or "docker"
	Firecracker         FirecrackerConfig `yaml:"firecracker"`
	Docker              DockerConfig      `yaml:"docker"`
}

// FirecrackerConfig holds Firecracker-specific configuration
type FirecrackerConfig struct {
	KernelPath      string `yaml:"kernel_path"`
	RootFSTemplate  string `yaml:"rootfs_template"`
	SocketDir       string `yaml:"socket_dir"`
	DefaultVCPU     int    `yaml:"default_vcpu"`
	DefaultMemoryMB int    `yaml:"default_memory_mb"`
}

// DockerConfig holds Docker-specific configuration
type DockerConfig struct {
	Network string `yaml:"network"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Provider string            `yaml:"provider"` // "stdout", "loki"
	Level    string            `yaml:"level"`    // "debug", "info", "warn", "error"
	Format   string            `yaml:"format"`   // "json" or "text"
	Output   string            `yaml:"output"`   // "stdout", "stderr", or file path
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
	Loki     LokiConfig        `yaml:"loki"`     // Loki-specific config
}

// TaskQueueConfig holds task queue configuration
type TaskQueueConfig struct {
	Provider string                 `yaml:"provider"` // "asynq", "memory"
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Provider string                 `yaml:"provider"` // "postgres", "memory"
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
}

// EventBusConfig holds event bus configuration
type EventBusConfig struct {
	Provider string                 `yaml:"provider"` // "redis", "memory"
	Config   map[string]interface{} `yaml:"config"`   // Provider-specific config
}

// IntegrationsConfig holds integrations configuration
type IntegrationsConfig struct {
	Enabled []string                     `yaml:"enabled"`      // List of enabled integrations
	Config  map[string]map[string]interface{} `yaml:"config"` // Per-integration config
}

// LokiConfig holds Loki logging configuration
type LokiConfig struct {
	URL           string            `yaml:"url"`
	BatchSize     int               `yaml:"batch_size"`
	BatchInterval string            `yaml:"batch_interval"` // e.g., "5s"
	Labels        map[string]string `yaml:"labels"`
}

// NetworkConfig holds network configuration
type NetworkConfig struct {
	BridgeName string      `yaml:"bridge_name"`
	BridgeIP   string      `yaml:"bridge_ip"`
	SubnetCIDR string      `yaml:"subnet_cidr"`
	TAPPrefix  string      `yaml:"tap_prefix"`
	EnableNAT  bool        `yaml:"enable_nat"`
	Proxy      ProxyConfig `yaml:"proxy"`
}

// ProxyConfig holds proxy configuration
type ProxyConfig struct {
	Enabled        bool        `yaml:"enabled"`
	Provider       string      `yaml:"provider"`        // "squid" or "none"
	Transparent    bool        `yaml:"transparent"`
	Port           int         `yaml:"port"`
	WhitelistMode  string      `yaml:"whitelist_mode"`  // "enforce", "monitor", "disabled"
	DefaultDomains []string    `yaml:"default_domains"`
	Squid          SquidConfig `yaml:"squid"`
	RedirectHTTP   bool        `yaml:"redirect_http"`
	RedirectHTTPS  bool        `yaml:"redirect_https"`
}

// SquidConfig holds Squid-specific configuration
type SquidConfig struct {
	ConfigPath  string `yaml:"config_path"`
	CacheDir    string `yaml:"cache_dir"`
	CacheSizeMB int    `yaml:"cache_size_mb"`
	AccessLog   string `yaml:"access_log"`
	CacheLog    string `yaml:"cache_log"`
}

// Load loads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply environment variable overrides
	config.applyEnvOverrides()

	// Set defaults
	config.setDefaults()

	return &config, nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		c.Database.Host = dbHost
	}
	if dbPass := os.Getenv("DB_PASSWORD"); dbPass != "" {
		c.Database.Password = dbPass
	}
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		c.Redis.Addr = redisAddr
	}
	if redisPass := os.Getenv("REDIS_PASSWORD"); redisPass != "" {
		c.Redis.Password = redisPass
	}
}

// setDefaults sets default values if not specified
func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "development"
	}

	if c.Database.Host == "" {
		c.Database.Host = "localhost"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}

	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}

	if c.Queue.Concurrency == 0 {
		c.Queue.Concurrency = 10
	}
	if c.Queue.Queues == nil {
		c.Queue.Queues = map[string]int{
			"critical": 6,
			"high":     5,
			"default":  3,
			"low":      1,
		}
	}

	if c.VMM.DefaultOrchestrator == "" {
		c.VMM.DefaultOrchestrator = "firecracker"
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "text"
	}
	if c.Logging.Output == "" {
		c.Logging.Output = "stdout"
	}

	// Network defaults
	if c.Network.BridgeName == "" {
		c.Network.BridgeName = "aetherium0"
	}
	if c.Network.BridgeIP == "" {
		c.Network.BridgeIP = "172.16.0.1/24"
	}
	if c.Network.SubnetCIDR == "" {
		c.Network.SubnetCIDR = "172.16.0.0/24"
	}
	if c.Network.TAPPrefix == "" {
		c.Network.TAPPrefix = "aether-"
	}
	// EnableNAT defaults to false if not specified (zero value)

	// Proxy defaults
	if c.Network.Proxy.Provider == "" {
		c.Network.Proxy.Provider = "squid"
	}
	if c.Network.Proxy.Port == 0 {
		c.Network.Proxy.Port = 3128
	}
	if c.Network.Proxy.WhitelistMode == "" {
		c.Network.Proxy.WhitelistMode = "enforce"
	}
	if len(c.Network.Proxy.DefaultDomains) == 0 {
		c.Network.Proxy.DefaultDomains = []string{
			// Package managers
			"registry.npmjs.org",
			"pypi.org",
			"files.pythonhosted.org",
			"rubygems.org",
			"repo.maven.apache.org",
			"proxy.golang.org",
			"crates.io",
			// Version control
			"github.com",
			"githubusercontent.com",
			"gitlab.com",
			"bitbucket.org",
			// Tool installers
			"nodejs.org",
			"python.org",
			"go.dev",
			"rust-lang.org",
			"mise.jdx.dev",
		}
	}

	// Squid defaults
	if c.Network.Proxy.Squid.ConfigPath == "" {
		c.Network.Proxy.Squid.ConfigPath = "/etc/squid/aetherium.conf"
	}
	if c.Network.Proxy.Squid.CacheDir == "" {
		c.Network.Proxy.Squid.CacheDir = "/var/spool/squid-aetherium"
	}
	if c.Network.Proxy.Squid.CacheSizeMB == 0 {
		c.Network.Proxy.Squid.CacheSizeMB = 1024
	}
	if c.Network.Proxy.Squid.AccessLog == "" {
		c.Network.Proxy.Squid.AccessLog = "/var/log/squid/aetherium-access.log"
	}
	if c.Network.Proxy.Squid.CacheLog == "" {
		c.Network.Proxy.Squid.CacheLog = "/var/log/squid/aetherium-cache.log"
	}

	// Logging provider defaults
	if c.Logging.Provider == "" {
		c.Logging.Provider = "stdout"
	}

	// TaskQueue provider defaults
	if c.TaskQueue.Provider == "" {
		c.TaskQueue.Provider = "asynq"
	}
	if c.TaskQueue.Config == nil {
		c.TaskQueue.Config = make(map[string]interface{})
	}

	// Storage provider defaults
	if c.Storage.Provider == "" {
		c.Storage.Provider = "postgres"
	}
	if c.Storage.Config == nil {
		c.Storage.Config = make(map[string]interface{})
	}

	// EventBus provider defaults
	if c.EventBus.Provider == "" {
		c.EventBus.Provider = "redis"
	}
	if c.EventBus.Config == nil {
		c.EventBus.Config = make(map[string]interface{})
	}

	// Integrations defaults
	if c.Integrations.Enabled == nil {
		c.Integrations.Enabled = []string{}
	}
	if c.Integrations.Config == nil {
		c.Integrations.Config = make(map[string]map[string]interface{})
	}
}

// GetIntegrationConfig returns the configuration for a specific integration
func (c *Config) GetIntegrationConfig(name string) (map[string]interface{}, error) {
	cfg, ok := c.Integrations.Config[name]
	if !ok {
		return make(map[string]interface{}), nil // Return empty config if not found
	}
	return cfg, nil
}

// GetIntOrDefault returns an integer value from config with a default fallback
func GetIntOrDefault(cfg map[string]interface{}, key string, defaultVal int) int {
	if cfg == nil {
		return defaultVal
	}
	val, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	if intVal, ok := val.(int); ok {
		return intVal
	}
	// Try to convert from float64 (common in YAML)
	if floatVal, ok := val.(float64); ok {
		return int(floatVal)
	}
	return defaultVal
}

// GetBoolOrDefault returns a boolean value from config with a default fallback
func GetBoolOrDefault(cfg map[string]interface{}, key string, defaultVal bool) bool {
	if cfg == nil {
		return defaultVal
	}
	val, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	if boolVal, ok := val.(bool); ok {
		return boolVal
	}
	return defaultVal
}

// GetStringOrDefault returns a string value from config with a default fallback
func GetStringOrDefault(cfg map[string]interface{}, key string, defaultVal string) string {
	if cfg == nil {
		return defaultVal
	}
	val, ok := cfg[key]
	if !ok {
		return defaultVal
	}
	if strVal, ok := val.(string); ok {
		return strVal
	}
	return defaultVal
}
