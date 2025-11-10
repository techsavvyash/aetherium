package security

import (
	"regexp"
	"strings"
)

// Redactor handles redaction of sensitive information from text output
type Redactor struct {
	secrets  []string
	patterns []*regexp.Regexp
}

// Common secret patterns
var defaultPatterns = []*regexp.Regexp{
	// GitHub tokens (classic, fine-grained, app)
	regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`gho_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghu_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghs_[a-zA-Z0-9]{36}`),
	regexp.MustCompile(`ghr_[a-zA-Z0-9]{36}`),

	// AWS Access Keys
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	regexp.MustCompile(`(?i)aws_secret_access_key\s*=\s*[A-Za-z0-9/+=]{40}`),

	// Anthropic/Claude API Keys
	regexp.MustCompile(`sk-ant-api03-[a-zA-Z0-9_-]{95}`),
	regexp.MustCompile(`sk-ant-[a-zA-Z0-9_-]+`),

	// OpenAI API Keys
	regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`),

	// Generic API keys
	regexp.MustCompile(`(?i)api[_-]?key["\s:=]+[a-zA-Z0-9_\-]{20,}`),

	// Bearer tokens in headers
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`),
	regexp.MustCompile(`(?i)authorization:\s*bearer\s+[a-zA-Z0-9_\-\.]{20,}`),

	// Private keys
	regexp.MustCompile(`-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----[\s\S]+?-----END\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`),

	// JWT tokens
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]+\.eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`),

	// Generic passwords
	regexp.MustCompile(`(?i)password["\s:=]+[^\s"']{8,}`),

	// Slack tokens
	regexp.MustCompile(`xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{24,}`),

	// Stripe keys
	regexp.MustCompile(`sk_live_[a-zA-Z0-9]{24,}`),
	regexp.MustCompile(`rk_live_[a-zA-Z0-9]{24,}`),
}

// NewRedactor creates a new redactor with specific secrets to redact
func NewRedactor(secrets map[string]string) *Redactor {
	secretValues := make([]string, 0, len(secrets))
	for _, value := range secrets {
		if value != "" && len(value) > 3 { // Only redact non-trivial values
			secretValues = append(secretValues, value)
		}
	}

	return &Redactor{
		secrets:  secretValues,
		patterns: defaultPatterns,
	}
}

// NewRedactorWithValues creates a redactor with secret values (no keys needed)
func NewRedactorWithValues(secretValues []string) *Redactor {
	filtered := make([]string, 0, len(secretValues))
	for _, value := range secretValues {
		if value != "" && len(value) > 3 {
			filtered = append(filtered, value)
		}
	}

	return &Redactor{
		secrets:  filtered,
		patterns: defaultPatterns,
	}
}

// Redact redacts all known secrets and common patterns from the input text
func (r *Redactor) Redact(text string) string {
	if text == "" {
		return text
	}

	redacted := text

	// First, redact exact secret values
	for _, secret := range r.secrets {
		if secret == "" {
			continue
		}
		// Use case-sensitive replacement for exact matches
		redacted = strings.ReplaceAll(redacted, secret, "[REDACTED]")

		// Also try URL-encoded version
		urlEncoded := strings.ReplaceAll(secret, " ", "%20")
		if urlEncoded != secret {
			redacted = strings.ReplaceAll(redacted, urlEncoded, "[REDACTED]")
		}
	}

	// Then, redact common secret patterns
	for _, pattern := range r.patterns {
		redacted = pattern.ReplaceAllString(redacted, "[REDACTED]")
	}

	return redacted
}

// RedactMap redacts secrets from a map of strings
func (r *Redactor) RedactMap(data map[string]string) map[string]string {
	if data == nil {
		return nil
	}

	redacted := make(map[string]string, len(data))
	for k, v := range data {
		redacted[k] = r.Redact(v)
	}
	return redacted
}

// RedactSlice redacts secrets from a slice of strings
func (r *Redactor) RedactSlice(data []string) []string {
	if data == nil {
		return nil
	}

	redacted := make([]string, len(data))
	for i, v := range data {
		redacted[i] = r.Redact(v)
	}
	return redacted
}

// IsSensitiveKey checks if a key name suggests it contains sensitive data
func IsSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitivePatterns := []string{
		"password", "passwd", "pwd",
		"secret", "token", "key", "api_key", "apikey",
		"auth", "authorization", "bearer",
		"credential", "cred",
		"private", "priv",
		"salt", "hash",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// RedactSensitiveEnv filters out environment variables with sensitive names
func RedactSensitiveEnv(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}

	redacted := make(map[string]string, len(env))
	for k, v := range env {
		if IsSensitiveKey(k) {
			redacted[k] = "[REDACTED]"
		} else {
			redacted[k] = v
		}
	}
	return redacted
}
