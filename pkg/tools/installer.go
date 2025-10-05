package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aetherium/aetherium/pkg/vmm"
)

// Installer handles tool installation in VMs
type Installer struct {
	orchestrator vmm.VMOrchestrator
}

// NewInstaller creates a new tool installer
func NewInstaller(orchestrator vmm.VMOrchestrator) *Installer {
	return &Installer{
		orchestrator: orchestrator,
	}
}

// InstallTools installs a list of tools in a VM
func (i *Installer) InstallTools(ctx context.Context, vmID string, tools []string, versions map[string]string) error {
	if len(tools) == 0 {
		return nil
	}

	log.Printf("Installing tools in VM %s: %v", vmID, tools)

	for _, tool := range tools {
		version := versions[tool]
		if version == "" {
			version = "latest"
		}

		log.Printf("Installing %s@%s...", tool, version)

		if err := i.installTool(ctx, vmID, tool, version); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}

		log.Printf("✓ %s@%s installed successfully", tool, version)
	}

	return nil
}

// installTool installs a specific tool
func (i *Installer) installTool(ctx context.Context, vmID string, tool string, version string) error {
	script, err := getInstallScript(tool, version)
	if err != nil {
		return err
	}

	// Execute installation script
	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", script},
	}

	result, err := i.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return fmt.Errorf("failed to execute install script: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("install script failed: %s\n%s", result.Stderr, result.Stdout)
	}

	return nil
}

// VerifyTools checks if tools are installed
func (i *Installer) VerifyTools(ctx context.Context, vmID string, tools []string) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, tool := range tools {
		installed, err := i.isToolInstalled(ctx, vmID, tool)
		if err != nil {
			log.Printf("Warning: Failed to verify %s: %v", tool, err)
			results[tool] = false
			continue
		}
		results[tool] = installed
	}

	return results, nil
}

// isToolInstalled checks if a tool is installed
func (i *Installer) isToolInstalled(ctx context.Context, vmID string, tool string) (bool, error) {
	checkCmd := getVerifyCommand(tool)

	cmd := &vmm.Command{
		Cmd:  "bash",
		Args: []string{"-c", checkCmd},
	}

	result, err := i.orchestrator.ExecuteCommand(ctx, vmID, cmd)
	if err != nil {
		return false, err
	}

	return result.ExitCode == 0, nil
}

// getInstallScript returns the installation script for a tool
func getInstallScript(tool string, version string) (string, error) {
	// Normalize tool name
	tool = strings.ToLower(strings.TrimSpace(tool))

	switch tool {
	case "nodejs", "node":
		return getNodeJSInstallScript(version), nil
	case "bun":
		return getBunInstallScript(version), nil
	case "claude-code", "claudecode":
		return getClaudeCodeInstallScript(version), nil
	case "go", "golang":
		return getGoInstallScript(version), nil
	case "python", "python3":
		return getPythonInstallScript(version), nil
	case "rust", "cargo":
		return getRustInstallScript(version), nil
	case "git":
		return getGitInstallScript(), nil
	case "docker":
		return getDockerInstallScript(), nil
	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

// getVerifyCommand returns the command to verify if a tool is installed
func getVerifyCommand(tool string) string {
	tool = strings.ToLower(strings.TrimSpace(tool))

	switch tool {
	case "nodejs", "node":
		return "which node && node --version"
	case "bun":
		return "which bun && bun --version"
	case "claude-code", "claudecode":
		return "which claude-code && claude-code --version"
	case "go", "golang":
		return "which go && go version"
	case "python", "python3":
		return "which python3 && python3 --version"
	case "rust", "cargo":
		return "which cargo && cargo --version"
	case "git":
		return "which git && git --version"
	case "docker":
		return "which docker && docker --version"
	default:
		return fmt.Sprintf("which %s", tool)
	}
}

// Tool-specific installation scripts

func getNodeJSInstallScript(version string) string {
	if version == "" || version == "latest" {
		version = "20"
	}
	return fmt.Sprintf(`
set -e
export DEBIAN_FRONTEND=noninteractive

# Install Node.js via NodeSource
curl -fsSL https://deb.nodesource.com/setup_%s.x | bash -
apt-get install -y nodejs

# Verify installation
node --version
npm --version
`, version)
}

func getBunInstallScript(version string) string {
	return `
set -e

# Install unzip if not present
apt-get update && apt-get install -y unzip curl

# Install Bun
curl -fsSL https://bun.sh/install | bash

# Add to PATH
export BUN_INSTALL="$HOME/.bun"
export PATH="$BUN_INSTALL/bin:$PATH"

# Make it permanent
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc

# Verify installation
~/.bun/bin/bun --version
`
}

func getClaudeCodeInstallScript(version string) string {
	return `
set -e

# Ensure npm is available (Claude Code is installed via npm)
if ! command -v npm &> /dev/null; then
    echo "Error: npm is required to install claude-code"
    exit 1
fi

# Install Claude Code CLI globally
npm install -g claude-code

# Verify installation
claude-code --version

echo "Claude Code installed successfully"
`
}

func getGoInstallScript(version string) string {
	if version == "" || version == "latest" {
		version = "1.23.0"
	}
	return fmt.Sprintf(`
set -e

# Download and install Go
cd /tmp
wget https://go.dev/dl/go%s.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go%s.linux-amd64.tar.gz
rm go%s.linux-amd64.tar.gz

# Add to PATH
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Verify installation
/usr/local/go/bin/go version
`, version, version, version)
}

func getPythonInstallScript(version string) string {
	if version == "" || version == "latest" {
		version = "3.11"
	}
	return fmt.Sprintf(`
set -e
export DEBIAN_FRONTEND=noninteractive

# Install Python
apt-get update
apt-get install -y python%s python%s-pip python%s-venv

# Create symbolic links
ln -sf /usr/bin/python%s /usr/local/bin/python
ln -sf /usr/bin/python%s /usr/local/bin/python3
ln -sf /usr/bin/pip%s /usr/local/bin/pip

# Verify installation
python3 --version
pip3 --version
`, version, version, version, version, version, version)
}

func getRustInstallScript(version string) string {
	return `
set -e

# Install Rust via rustup
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

# Source cargo environment
source "$HOME/.cargo/env"

# Add to PATH permanently
echo 'source "$HOME/.cargo/env"' >> ~/.bashrc

# Verify installation
~/.cargo/bin/cargo --version
~/.cargo/bin/rustc --version
`
}

func getGitInstallScript() string {
	return `
set -e
export DEBIAN_FRONTEND=noninteractive

# Install git
apt-get update
apt-get install -y git

# Verify installation
git --version
`
}

func getDockerInstallScript() string {
	return `
set -e
export DEBIAN_FRONTEND=noninteractive

# Install Docker
apt-get update
apt-get install -y ca-certificates curl gnupg
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
chmod a+r /etc/apt/keyrings/docker.gpg

echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null

apt-get update
apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# Verify installation
docker --version
`
}

// GetDefaultTools returns the default tools that should be installed in all VMs
func GetDefaultTools() []string {
	return []string{
		"git",
		"nodejs",
		"bun",
		"claude-code",
	}
}

// InstallToolsWithTimeout installs tools with a timeout
func (i *Installer) InstallToolsWithTimeout(ctx context.Context, vmID string, tools []string, versions map[string]string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	errChan := make(chan error, 1)

	go func() {
		errChan <- i.InstallTools(ctx, vmID, tools, versions)
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return fmt.Errorf("tool installation timed out after %v", timeout)
	}
}
