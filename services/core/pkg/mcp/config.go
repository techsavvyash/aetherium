// Package mcp provides utilities for generating Claude Code MCP server configurations.
package mcp

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/aetherium/aetherium/services/core/pkg/storage"
)

// ClaudeSettings represents the ~/.claude/settings.json structure
type ClaudeSettings struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry is the format Claude Code expects for MCP server configuration
type MCPServerEntry struct {
	// For stdio type servers
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`

	// For http type servers
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`

	// Common - environment variables for the MCP server
	Env map[string]string `json:"env,omitempty"`
}

// envVarPattern matches ${VAR_NAME} patterns for environment variable expansion
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// GenerateClaudeSettings converts environment MCP configs to Claude Code settings format
func GenerateClaudeSettings(servers []storage.MCPServerConfig, envVars map[string]string) *ClaudeSettings {
	settings := &ClaudeSettings{
		MCPServers: make(map[string]MCPServerEntry),
	}

	for _, server := range servers {
		entry := MCPServerEntry{}

		switch server.Type {
		case storage.MCPServerTypeStdio:
			entry.Command = server.Command
			entry.Args = server.Args
		case storage.MCPServerTypeHTTP:
			entry.URL = expandEnvVars(server.URL, envVars)
			if len(server.Headers) > 0 {
				entry.Headers = make(map[string]string)
				for k, v := range server.Headers {
					entry.Headers[k] = expandEnvVars(v, envVars)
				}
			}
		}

		// Copy and expand env vars
		if len(server.Env) > 0 {
			entry.Env = make(map[string]string)
			for k, v := range server.Env {
				entry.Env[k] = expandEnvVars(v, envVars)
			}
		}

		settings.MCPServers[server.Name] = entry
	}

	return settings
}

// expandEnvVars replaces ${VAR} patterns with values from envVars
func expandEnvVars(s string, envVars map[string]string) string {
	if envVars == nil {
		return s
	}

	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if value, ok := envVars[varName]; ok {
			return value
		}
		// Keep original if not found
		return match
	})
}

// ToJSON serializes the settings to JSON for writing to ~/.claude/settings.json
func (s *ClaudeSettings) ToJSON() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// Presets provides common MCP server configurations
var Presets = map[string]storage.MCPServerConfig{
	"playwright": {
		Name:    "playwright",
		Type:    storage.MCPServerTypeStdio,
		Command: "npx",
		Args:    []string{"@playwright/mcp@latest"},
	},
	"filesystem": {
		Name:    "filesystem",
		Type:    storage.MCPServerTypeStdio,
		Command: "npx",
		Args:    []string{"-y", "@anthropic/mcp-filesystem", "/workspace"},
	},
	"git": {
		Name:    "git",
		Type:    storage.MCPServerTypeStdio,
		Command: "npx",
		Args:    []string{"-y", "@anthropic/mcp-git"},
	},
}

// GetPreset returns a preset MCP server configuration by name
func GetPreset(name string) (storage.MCPServerConfig, bool) {
	preset, ok := Presets[name]
	return preset, ok
}
