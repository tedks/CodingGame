package capability

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMCPDiscovererName(t *testing.T) {
	d := NewMCPDiscoverer("/tmp/test")
	if d.Name() != "mcp-servers" {
		t.Errorf("expected name 'mcp-servers', got %q", d.Name())
	}
}

func TestMCPDiscovererWithNoConfigs(t *testing.T) {
	// Use a temp directory with no configs
	tmpDir := t.TempDir()

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	// Should return empty (no error, just no capabilities)
	// Note: might find global configs if they exist on the machine
	_ = caps
}

func TestMCPDiscovererParsesMCPJson(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test .mcp.json file
	mcpConfig := `{
		"mcpServers": {
			"github": {
				"command": "github-mcp",
				"args": ["--token", "xxx"]
			},
			"docker": {
				"command": "docker-mcp"
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(mcpConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	// Find MCP servers
	mcpCaps := make(map[string]*Capability)
	for _, cap := range caps {
		if cap.Type == TypeMCP {
			mcpCaps[cap.Name] = cap
		}
	}

	// Check github MCP
	if github, exists := mcpCaps["github"]; !exists {
		t.Error("expected github MCP server not found")
	} else {
		if github.Domain != DomainVersionCtrl {
			t.Errorf("github MCP should be in version_ctrl domain, got %v", github.Domain)
		}
		if github.Source != configPath {
			t.Errorf("expected source %q, got %q", configPath, github.Source)
		}
	}

	// Check docker MCP
	if docker, exists := mcpCaps["docker"]; !exists {
		t.Error("expected docker MCP server not found")
	} else {
		if docker.Domain != DomainDeployment {
			t.Errorf("docker MCP should be in deployment domain, got %v", docker.Domain)
		}
	}
}

func TestMCPDiscovererWatchPaths(t *testing.T) {
	d := NewMCPDiscoverer("/tmp/test-project")
	paths := d.WatchPaths()

	// Should include both global and project paths
	if len(paths) == 0 {
		t.Error("expected at least one watch path")
	}

	// Check for project-local path
	found := false
	for _, p := range paths {
		if p == "/tmp/test-project/.mcp.json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected project .mcp.json in watch paths")
	}
}

func TestInferMCPDomain(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected Domain
	}{
		{"github-mcp", "github-mcp-server", DomainVersionCtrl},
		{"git-tools", "git-helper", DomainVersionCtrl},
		{"bazel-build", "bazel-mcp", DomainBuild},
		{"npm-runner", "npm-mcp", DomainBuild},
		{"k8s-tools", "kubernetes-mcp", DomainDeployment},
		{"aws-cli", "aws-mcp", DomainDeployment},
		{"docker-helper", "docker-mcp", DomainDeployment},
		{"eslint-server", "eslint-mcp", DomainAnalysis},
		{"test-runner", "jest-mcp", DomainAnalysis},
		{"unknown-tool", "random-mcp", DomainCore}, // Default
	}

	for _, tc := range tests {
		got := inferMCPDomain(tc.name, tc.command)
		if got != tc.expected {
			t.Errorf("inferMCPDomain(%q, %q) = %v, expected %v",
				tc.name, tc.command, got, tc.expected)
		}
	}
}
