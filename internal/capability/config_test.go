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

// Error path tests

func TestMCPDiscovererMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a malformed JSON file
	malformedConfig := `{ this is not valid json `
	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(malformedConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()

	// Should not return error (graceful degradation), but also no capabilities from malformed file
	if err != nil {
		t.Fatalf("Discover() should not return error for malformed JSON: %v", err)
	}

	// Check that no MCP capabilities were found from the malformed file
	for _, cap := range caps {
		if cap.Type == TypeMCP && cap.Source == configPath {
			t.Error("should not have found MCP capabilities from malformed JSON")
		}
	}
}

func TestMCPDiscovererEmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty JSON file
	emptyConfig := `{}`
	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()

	if err != nil {
		t.Fatalf("Discover() should not return error for empty JSON: %v", err)
	}

	// Check that no MCP capabilities were found
	for _, cap := range caps {
		if cap.Type == TypeMCP && cap.Source == configPath {
			t.Error("should not have found MCP capabilities from empty JSON")
		}
	}
}

func TestMCPDiscovererEmptyMCPServers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create JSON with empty mcpServers
	config := `{"mcpServers": {}}`
	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()

	if err != nil {
		t.Fatalf("Discover() should not return error for empty mcpServers: %v", err)
	}

	// Check that no MCP capabilities were found
	for _, cap := range caps {
		if cap.Type == TypeMCP && cap.Source == configPath {
			t.Error("should not have found MCP capabilities from empty mcpServers")
		}
	}
}

func TestMCPDiscovererMissingCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config with missing command field
	config := `{
		"mcpServers": {
			"test-server": {
				"args": ["--test"]
			}
		}
	}`
	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()

	// Should succeed but the capability will have empty command in description
	if err != nil {
		t.Fatalf("Discover() should not return error: %v", err)
	}

	// Verify the capability was created (even with empty command)
	found := false
	for _, cap := range caps {
		if cap.Name == "test-server" {
			found = true
			// Command is empty, so description should just be empty or have args
			break
		}
	}
	if !found {
		t.Error("expected test-server capability to be created even with missing command")
	}
}

func TestMCPDiscovererUnreadableFile(t *testing.T) {
	// Skip on platforms where we can't control permissions easily
	tmpDir := t.TempDir()

	// Create a file that's not readable
	configPath := filepath.Join(tmpDir, ".mcp.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0000); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewMCPDiscoverer(tmpDir)
	caps, err := d.Discover()

	// Should not error (graceful degradation)
	if err != nil {
		t.Fatalf("Discover() should not return error for unreadable file: %v", err)
	}

	// Restore permissions for cleanup
	os.Chmod(configPath, 0644)

	_ = caps
}
