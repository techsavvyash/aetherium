package security

import (
	"context"
	"time"
)

// EncryptionService defines the interface for encryption operations
type EncryptionService interface {
	// Encrypt encrypts plaintext using the specified key
	Encrypt(ctx context.Context, plaintext string, keyID string) (*EncryptResult, error)

	// Decrypt decrypts ciphertext using the specified key
	Decrypt(ctx context.Context, ciphertext string, keyID string) (string, error)

	// RotateKey triggers a key rotation
	RotateKey(ctx context.Context, keyID string) error

	// GetKeyInfo returns metadata about a key (not the key itself)
	GetKeyInfo(ctx context.Context, keyID string) (*KeyInfo, error)

	// Health checks if the encryption service is available
	Health(ctx context.Context) error

	// Close closes any open connections
	Close() error
}

// EncryptResult contains the result of an encryption operation
type EncryptResult struct {
	Ciphertext string // Base64-encoded ciphertext
	KeyID      string // Key identifier used for encryption
	Version    int    // Key version used for encryption
}

// KeyInfo contains metadata about an encryption key
type KeyInfo struct {
	Name            string
	Type            string
	Version         int
	LatestVersion   int
	MinDecryptVers  int
	Created         time.Time
	Updated         time.Time
	DeletionAllowed bool
	Derived         bool
	Exportable      bool
}
