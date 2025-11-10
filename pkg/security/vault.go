package security

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	vault "github.com/hashicorp/vault/api"
)

// VaultEncryptionService implements EncryptionService using HashiCorp Vault
type VaultEncryptionService struct {
	client     *vault.Client
	mountPath  string // Transit mount path, default: "transit"
	defaultKey string // Default key name
}

// VaultConfig holds Vault configuration
type VaultConfig struct {
	Address    string // Vault server address (e.g., "http://localhost:8200")
	Token      string // Vault authentication token
	MountPath  string // Transit engine mount path (default: "transit")
	DefaultKey string // Default encryption key name (default: "aetherium")
}

// NewVaultEncryptionService creates a new Vault-based encryption service
func NewVaultEncryptionService(config VaultConfig) (*VaultEncryptionService, error) {
	// Set defaults
	if config.MountPath == "" {
		config.MountPath = "transit"
	}
	if config.DefaultKey == "" {
		config.DefaultKey = "aetherium"
	}

	// Create Vault client
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set token
	client.SetToken(config.Token)

	service := &VaultEncryptionService{
		client:     client,
		mountPath:  config.MountPath,
		defaultKey: config.DefaultKey,
	}

	// Verify connection
	if err := service.Health(context.Background()); err != nil {
		return nil, fmt.Errorf("vault health check failed: %w", err)
	}

	return service, nil
}

// Encrypt encrypts plaintext using Vault's transit engine
func (v *VaultEncryptionService) Encrypt(ctx context.Context, plaintext string, keyID string) (*EncryptResult, error) {
	if keyID == "" {
		keyID = v.defaultKey
	}

	// Base64 encode the plaintext
	encoded := base64.StdEncoding.EncodeToString([]byte(plaintext))

	// Encrypt using Vault transit
	path := fmt.Sprintf("%s/encrypt/%s", v.mountPath, keyID)
	data := map[string]interface{}{
		"plaintext": encoded,
	}

	secret, err := v.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("empty response from vault")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid ciphertext in response")
	}

	// Extract key version if available
	version := 1
	if v, ok := secret.Data["key_version"].(int); ok {
		version = v
	}

	return &EncryptResult{
		Ciphertext: ciphertext,
		KeyID:      keyID,
		Version:    version,
	}, nil
}

// Decrypt decrypts ciphertext using Vault's transit engine
func (v *VaultEncryptionService) Decrypt(ctx context.Context, ciphertext string, keyID string) (string, error) {
	if keyID == "" {
		keyID = v.defaultKey
	}

	// Decrypt using Vault transit
	path := fmt.Sprintf("%s/decrypt/%s", v.mountPath, keyID)
	data := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	secret, err := v.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("empty response from vault")
	}

	encoded, ok := secret.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("invalid plaintext in response")
	}

	// Base64 decode the plaintext
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode plaintext: %w", err)
	}

	return string(decoded), nil
}

// RotateKey rotates the encryption key
func (v *VaultEncryptionService) RotateKey(ctx context.Context, keyID string) error {
	if keyID == "" {
		keyID = v.defaultKey
	}

	path := fmt.Sprintf("%s/keys/%s/rotate", v.mountPath, keyID)

	_, err := v.client.Logical().WriteWithContext(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to rotate key: %w", err)
	}

	return nil
}

// GetKeyInfo retrieves metadata about an encryption key
func (v *VaultEncryptionService) GetKeyInfo(ctx context.Context, keyID string) (*KeyInfo, error) {
	if keyID == "" {
		keyID = v.defaultKey
	}

	path := fmt.Sprintf("%s/keys/%s", v.mountPath, keyID)

	secret, err := v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key info: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}

	info := &KeyInfo{
		Name: keyID,
	}

	// Extract key information
	if v, ok := secret.Data["type"].(string); ok {
		info.Type = v
	}
	if v, ok := secret.Data["latest_version"].(int); ok {
		info.LatestVersion = v
		info.Version = v
	}
	if v, ok := secret.Data["min_decryption_version"].(int); ok {
		info.MinDecryptVers = v
	}
	if v, ok := secret.Data["deletion_allowed"].(bool); ok {
		info.DeletionAllowed = v
	}
	if v, ok := secret.Data["derived"].(bool); ok {
		info.Derived = v
	}
	if v, ok := secret.Data["exportable"].(bool); ok {
		info.Exportable = v
	}

	// Parse creation time if available
	if v, ok := secret.Data["creation_time"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			info.Created = t
		}
	}

	return info, nil
}

// Health checks if Vault is available and accessible
func (v *VaultEncryptionService) Health(ctx context.Context) error {
	health, err := v.client.Sys().HealthWithContext(ctx)
	if err != nil {
		return fmt.Errorf("vault health check failed: %w", err)
	}

	if health.Sealed {
		return fmt.Errorf("vault is sealed")
	}

	if !health.Initialized {
		return fmt.Errorf("vault is not initialized")
	}

	// Try to read the key to verify access
	path := fmt.Sprintf("%s/keys/%s", v.mountPath, v.defaultKey)
	_, err = v.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return fmt.Errorf("cannot access encryption key: %w", err)
	}

	return nil
}

// Close closes the Vault client connection
func (v *VaultEncryptionService) Close() error {
	// Vault client doesn't require explicit cleanup
	return nil
}
