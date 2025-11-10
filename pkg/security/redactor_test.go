package security

import (
	"strings"
	"testing"
)

func TestRedactor_Redact(t *testing.T) {
	tests := []struct {
		name     string
		secrets  map[string]string
		input    string
		expected string
	}{
		{
			name: "GitHub token redaction",
			secrets: map[string]string{
				"GITHUB_TOKEN": "ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			},
			input:    "Cloning with token: ghp_1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "Cloning with token: [REDACTED]",
		},
		{
			name: "Claude API key redaction",
			secrets: map[string]string{
				"CLAUDE_API_KEY": "sk-ant-api03-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567AB890CD123EF456GH789IJ012KL345MN",
			},
			input:    "Using API key: sk-ant-api03-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz567AB890CD123EF456GH789IJ012KL345MN",
			expected: "Using API key: [REDACTED]",
		},
		{
			name: "Multiple secrets",
			secrets: map[string]string{
				"TOKEN1": "secret123",
				"TOKEN2": "password456",
			},
			input:    "Token is secret123 and password is password456",
			expected: "Token is [REDACTED] and password is [REDACTED]",
		},
		{
			name:     "Empty input",
			secrets:  map[string]string{"TOKEN": "secret"},
			input:    "",
			expected: "",
		},
		{
			name:     "No secrets to redact",
			secrets:  map[string]string{},
			input:    "Normal output with no secrets",
			expected: "Normal output with no secrets",
		},
		{
			name: "Pattern-based GitHub token detection",
			secrets: map[string]string{
				"OTHER_TOKEN": "unrelated",
			},
			input:    "Error: authentication failed with ghp_AbCdEfGhIjKlMnOpQrStUvWxYz0123456789",
			expected: "Error: authentication failed with [REDACTED]",
		},
		{
			name: "AWS key detection",
			secrets: map[string]string{},
			input:    "AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE",
			expected: "AWS_ACCESS_KEY_ID=[REDACTED]",
		},
		{
			name: "Bearer token in header",
			secrets: map[string]string{},
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			expected: "Authorization: [REDACTED]",
		},
		{
			name: "Private key redaction",
			secrets: map[string]string{},
			input: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF6xWT7z0q8dR6UYdaX0D/gfbFwIu
-----END RSA PRIVATE KEY-----`,
			expected: "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redactor := NewRedactor(tt.secrets)
			result := redactor.Redact(tt.input)
			if result != tt.expected {
				t.Errorf("Redact() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRedactor_RedactMap(t *testing.T) {
	secrets := map[string]string{
		"SECRET": "confidential123",
	}
	redactor := NewRedactor(secrets)

	input := map[string]string{
		"public":  "normal value",
		"private": "confidential123 leaked",
		"empty":   "",
	}

	result := redactor.RedactMap(input)

	if result["public"] != "normal value" {
		t.Errorf("Public value should not be redacted")
	}
	if result["private"] != "[REDACTED] leaked" {
		t.Errorf("Private value should be redacted, got: %s", result["private"])
	}
	if result["empty"] != "" {
		t.Errorf("Empty value should remain empty")
	}
}

func TestRedactor_RedactSlice(t *testing.T) {
	secrets := map[string]string{
		"TOKEN": "mytoken123",
	}
	redactor := NewRedactor(secrets)

	input := []string{
		"normal string",
		"contains mytoken123 here",
		"",
	}

	result := redactor.RedactSlice(input)

	if result[0] != "normal string" {
		t.Errorf("Normal string should not be redacted")
	}
	if result[1] != "contains [REDACTED] here" {
		t.Errorf("Token should be redacted, got: %s", result[1])
	}
	if result[2] != "" {
		t.Errorf("Empty string should remain empty")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	tests := []struct {
		key       string
		sensitive bool
	}{
		{"PASSWORD", true},
		{"password", true},
		{"DB_PASSWORD", true},
		{"api_key", true},
		{"API_KEY", true},
		{"GITHUB_TOKEN", true},
		{"secret", true},
		{"MY_SECRET_KEY", true},
		{"authorization", true},
		{"bearer_token", true},
		{"username", false},
		{"email", false},
		{"public_key", false}, // Debatable, but public keys are not secret
		{"NORMAL_VAR", false},
		{"count", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := IsSensitiveKey(tt.key)
			if result != tt.sensitive {
				t.Errorf("IsSensitiveKey(%s) = %v, want %v", tt.key, result, tt.sensitive)
			}
		})
	}
}

func TestRedactSensitiveEnv(t *testing.T) {
	env := map[string]string{
		"USERNAME":      "john",
		"PASSWORD":      "secret123",
		"API_KEY":       "key123",
		"GITHUB_TOKEN":  "ghp_token",
		"DATABASE_HOST": "localhost",
		"PORT":          "5432",
	}

	result := RedactSensitiveEnv(env)

	// Non-sensitive should be preserved
	if result["USERNAME"] != "john" {
		t.Errorf("USERNAME should not be redacted")
	}
	if result["DATABASE_HOST"] != "localhost" {
		t.Errorf("DATABASE_HOST should not be redacted")
	}
	if result["PORT"] != "5432" {
		t.Errorf("PORT should not be redacted")
	}

	// Sensitive should be redacted
	if result["PASSWORD"] != "[REDACTED]" {
		t.Errorf("PASSWORD should be redacted")
	}
	if result["API_KEY"] != "[REDACTED]" {
		t.Errorf("API_KEY should be redacted")
	}
	if result["GITHUB_TOKEN"] != "[REDACTED]" {
		t.Errorf("GITHUB_TOKEN should be redacted")
	}
}

func TestNewRedactorWithValues(t *testing.T) {
	values := []string{
		"secret1",
		"secret2",
		"", // Should be filtered out
		"ab", // Too short, should be filtered out
	}

	redactor := NewRedactorWithValues(values)

	if len(redactor.secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(redactor.secrets))
	}

	input := "Text with secret1 and secret2 and ab"
	result := redactor.Redact(input)
	expected := "Text with [REDACTED] and [REDACTED] and ab"

	if result != expected {
		t.Errorf("Redact() = %v, want %v", result, expected)
	}
}

func TestRedactor_URLEncodedSecrets(t *testing.T) {
	secrets := map[string]string{
		"TOKEN": "my secret token",
	}
	redactor := NewRedactor(secrets)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal secret",
			input:    "Token is: my secret token",
			expected: "Token is: [REDACTED]",
		},
		{
			name:     "URL encoded secret",
			input:    "Token is: my%20secret%20token",
			expected: "Token is: [REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.Redact(tt.input)
			if result != tt.expected {
				t.Errorf("Redact() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func BenchmarkRedactor_Redact(b *testing.B) {
	secrets := map[string]string{
		"TOKEN1": "secret123",
		"TOKEN2": "password456",
		"TOKEN3": "apikey789",
	}
	redactor := NewRedactor(secrets)
	input := strings.Repeat("Normal log output with secret123 and password456 mixed in. ", 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = redactor.Redact(input)
	}
}
