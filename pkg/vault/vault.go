package vault

import (
	"context"
	"fmt"
	"log"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// Client wraps the Vault API client with helper methods
type Client struct {
	client *vault.Client
	mount  string // KV mount point (default: "secret")
}

// Config holds Vault client configuration
type Config struct {
	Address string // Vault server address (e.g., "http://localhost:8200")
	Token   string // Vault token for authentication
	Mount   string // KV secrets mount point (default: "secret")
}

// NewClient creates a new Vault client
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Address == "" {
		cfg.Address = "http://localhost:8200"
	}
	if cfg.Mount == "" {
		cfg.Mount = "secret"
	}

	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = cfg.Address

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	if cfg.Token != "" {
		client.SetToken(cfg.Token)
	}

	return &Client{
		client: client,
		mount:  cfg.Mount,
	}, nil
}

// GetSecret retrieves a secret from Vault KV v2
func (c *Client) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := c.client.KVv2(c.mount).Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at %s", path)
	}

	return secret.Data, nil
}

// GetSecretString retrieves a single string value from a secret
func (c *Client) GetSecretString(ctx context.Context, path, key string) (string, error) {
	data, err := c.GetSecret(ctx, path)
	if err != nil {
		return "", err
	}

	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s", key, path)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value at %s/%s is not a string", path, key)
	}

	return strValue, nil
}

// PutSecret stores a secret in Vault KV v2
func (c *Client) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	_, err := c.client.KVv2(c.mount).Put(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to write secret to %s: %w", path, err)
	}

	return nil
}

// DeleteSecret removes a secret from Vault
func (c *Client) DeleteSecret(ctx context.Context, path string) error {
	err := c.client.KVv2(c.mount).Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %w", path, err)
	}

	return nil
}

// Health checks if Vault is accessible and healthy
func (c *Client) Health(ctx context.Context) error {
	health, err := c.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	return nil
}

// SecretStore provides high-level secret management for Aetherium
type SecretStore struct {
	client *Client
}

// NewSecretStore creates a new secret store
func NewSecretStore(cfg *Config) (*SecretStore, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	// Wait for Vault to be ready (with retries)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for i := 0; i < 10; i++ {
		if err := client.Health(ctx); err == nil {
			break
		}
		log.Printf("Waiting for Vault to be ready... (attempt %d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	return &SecretStore{client: client}, nil
}

// GetDatabaseCredentials retrieves database credentials
func (s *SecretStore) GetDatabaseCredentials(ctx context.Context) (host, user, password, database string, err error) {
	data, err := s.client.GetSecret(ctx, "database/postgres")
	if err != nil {
		return "", "", "", "", err
	}

	host, _ = data["host"].(string)
	user, _ = data["user"].(string)
	password, _ = data["password"].(string)
	database, _ = data["database"].(string)

	if host == "" || user == "" || password == "" || database == "" {
		return "", "", "", "", fmt.Errorf("incomplete database credentials in vault")
	}

	return host, user, password, database, nil
}

// GetRedisCredentials retrieves Redis connection info
func (s *SecretStore) GetRedisCredentials(ctx context.Context) (addr, password string, err error) {
	data, err := s.client.GetSecret(ctx, "redis/config")
	if err != nil {
		return "", "", err
	}

	addr, _ = data["addr"].(string)
	password, _ = data["password"].(string)

	if addr == "" {
		return "", "", fmt.Errorf("redis address not found in vault")
	}

	return addr, password, nil
}

// GetIntegrationToken retrieves an integration API token
func (s *SecretStore) GetIntegrationToken(ctx context.Context, integration string) (string, error) {
	path := fmt.Sprintf("integrations/%s", integration)
	return s.client.GetSecretString(ctx, path, "token")
}

// GetIntegrationSecret retrieves an integration secret
func (s *SecretStore) GetIntegrationSecret(ctx context.Context, integration string) (map[string]interface{}, error) {
	path := fmt.Sprintf("integrations/%s", integration)
	return s.client.GetSecret(ctx, path)
}

// SetDatabaseCredentials stores database credentials
func (s *SecretStore) SetDatabaseCredentials(ctx context.Context, host, user, password, database string) error {
	return s.client.PutSecret(ctx, "database/postgres", map[string]interface{}{
		"host":     host,
		"user":     user,
		"password": password,
		"database": database,
	})
}

// SetRedisCredentials stores Redis credentials
func (s *SecretStore) SetRedisCredentials(ctx context.Context, addr, password string) error {
	return s.client.PutSecret(ctx, "redis/config", map[string]interface{}{
		"addr":     addr,
		"password": password,
	})
}

// SetIntegrationToken stores an integration token
func (s *SecretStore) SetIntegrationToken(ctx context.Context, integration, token string) error {
	return s.client.PutSecret(ctx, fmt.Sprintf("integrations/%s", integration), map[string]interface{}{
		"token": token,
	})
}

// SetIntegrationSecret stores integration secrets
func (s *SecretStore) SetIntegrationSecret(ctx context.Context, integration string, data map[string]interface{}) error {
	path := fmt.Sprintf("integrations/%s", integration)
	return s.client.PutSecret(ctx, path, data)
}
