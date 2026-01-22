package capability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MCPConfig represents the structure of MCP configuration files.
// Both ~/.claude.json and .mcp.json use this format.
type MCPConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents a single MCP server configuration.
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPDiscoverer discovers MCP servers from configuration files.
type MCPDiscoverer struct {
	projectPath string
	homeDir     string
}

// NewMCPDiscoverer creates a new MCP discoverer.
// projectPath is the root of the current project for finding .mcp.json.
func NewMCPDiscoverer(projectPath string) *MCPDiscoverer {
	homeDir, _ := os.UserHomeDir()
	return &MCPDiscoverer{
		projectPath: projectPath,
		homeDir:     homeDir,
	}
}

// Name implements Discoverer.
func (d *MCPDiscoverer) Name() string {
	return "mcp-servers"
}

// Discover implements Discoverer.
func (d *MCPDiscoverer) Discover() ([]*Capability, error) {
	var capabilities []*Capability

	// Check global config locations
	for _, path := range d.globalConfigPaths() {
		caps, err := d.parseConfigFile(path)
		if err != nil {
			// Skip files that don't exist or can't be parsed
			continue
		}
		capabilities = append(capabilities, caps...)
	}

	// Check project-local config
	if d.projectPath != "" {
		localPath := filepath.Join(d.projectPath, ".mcp.json")
		caps, err := d.parseConfigFile(localPath)
		if err == nil {
			capabilities = append(capabilities, caps...)
		}
	}

	return capabilities, nil
}

// WatchPaths implements Discoverer.
func (d *MCPDiscoverer) WatchPaths() []string {
	paths := d.globalConfigPaths()
	if d.projectPath != "" {
		paths = append(paths, filepath.Join(d.projectPath, ".mcp.json"))
	}
	return paths
}

// globalConfigPaths returns paths to global MCP config files.
func (d *MCPDiscoverer) globalConfigPaths() []string {
	if d.homeDir == "" {
		return nil
	}
	return []string{
		filepath.Join(d.homeDir, ".claude.json"),
		filepath.Join(d.homeDir, ".claude", "settings.json"),
	}
}

// parseConfigFile parses an MCP configuration file and returns capabilities.
func (d *MCPDiscoverer) parseConfigFile(path string) ([]*Capability, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var capabilities []*Capability
	for name, server := range config.MCPServers {
		cap := d.mcpServerToCapability(name, server, path)
		capabilities = append(capabilities, cap)
	}

	return capabilities, nil
}

// mcpServerToCapability converts an MCP server config to a Capability.
func (d *MCPDiscoverer) mcpServerToCapability(name string, server MCPServer, source string) *Capability {
	// Infer domain from server name/command
	domain := inferMCPDomain(name, server.Command)

	// Create description from command
	desc := server.Command
	if len(server.Args) > 0 {
		desc += " " + strings.Join(server.Args, " ")
	}

	return NewCapability(
		"mcp-"+name,
		name,
		TypeMCP,
		domain,
	).WithDescription(desc).WithSource(source)
}

// inferMCPDomain attempts to determine the domain from MCP server name/command.
func inferMCPDomain(name, command string) Domain {
	nameCmd := strings.ToLower(name + " " + command)

	// Version control patterns
	if strings.Contains(nameCmd, "git") ||
		strings.Contains(nameCmd, "github") ||
		strings.Contains(nameCmd, "gitlab") {
		return DomainVersionCtrl
	}

	// Build patterns
	if strings.Contains(nameCmd, "build") ||
		strings.Contains(nameCmd, "bazel") ||
		strings.Contains(nameCmd, "npm") ||
		strings.Contains(nameCmd, "cargo") ||
		strings.Contains(nameCmd, "make") {
		return DomainBuild
	}

	// Deployment patterns
	if strings.Contains(nameCmd, "deploy") ||
		strings.Contains(nameCmd, "kubernetes") ||
		strings.Contains(nameCmd, "k8s") ||
		strings.Contains(nameCmd, "docker") ||
		strings.Contains(nameCmd, "aws") ||
		strings.Contains(nameCmd, "gcp") ||
		strings.Contains(nameCmd, "azure") ||
		strings.Contains(nameCmd, "cloud") {
		return DomainDeployment
	}

	// Analysis patterns
	if strings.Contains(nameCmd, "lint") ||
		strings.Contains(nameCmd, "test") ||
		strings.Contains(nameCmd, "coverage") ||
		strings.Contains(nameCmd, "analyze") ||
		strings.Contains(nameCmd, "eslint") ||
		strings.Contains(nameCmd, "prettier") {
		return DomainAnalysis
	}

	// Default to core for unrecognized servers
	return DomainCore
}
