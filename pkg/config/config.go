package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	Queue    QueueConfig    `yaml:"queue"`
	VMM      VMMConfig      `yaml:"vmm"`
	Logging  LoggingConfig  `yaml:"logging"`
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
	Level  string `yaml:"level"` // "debug", "info", "warn", "error"
	Format string `yaml:"format"` // "json" or "text"
	Output string `yaml:"output"` // "stdout", "stderr", or file path
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
}
