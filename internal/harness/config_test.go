package harness

import (
	"os"
	"testing"
)

func TestNewConfig(t *testing.T) {
	config := NewConfig("/test/path")

	if config.WorkingDir != "/test/path" {
		t.Errorf("WorkingDir = %q, want /test/path", config.WorkingDir)
	}
	if config.OutputFormat != "json" {
		t.Errorf("OutputFormat = %q, want json", config.OutputFormat)
	}
	if config.Env == nil {
		t.Error("Env should be initialized")
	}
	if config.MCPServers == nil {
		t.Error("MCPServers should be initialized")
	}
}

func TestConfigWithMethods(t *testing.T) {
	config := NewConfig("/test/path").
		WithModel("opus").
		WithSystemPrompt("You are a helpful assistant").
		WithEnv("API_KEY", "secret").
		WithMCPServer(MCPServer{Name: "test", Command: "test-server"})

	if config.Model != "opus" {
		t.Errorf("Model = %q, want opus", config.Model)
	}
	if config.SystemPrompt != "You are a helpful assistant" {
		t.Errorf("SystemPrompt = %q, want 'You are a helpful assistant'", config.SystemPrompt)
	}
	if config.Env["API_KEY"] != "secret" {
		t.Errorf("Env[API_KEY] = %q, want secret", config.Env["API_KEY"])
	}
	if len(config.MCPServers) != 1 || config.MCPServers[0].Name != "test" {
		t.Errorf("MCPServers = %v, want one server named 'test'", config.MCPServers)
	}
}

func TestConfigForAdvisor(t *testing.T) {
	config := NewConfig("/test/path").ForAdvisor("security-advisor")

	if !config.AdvisorMode {
		t.Error("AdvisorMode should be true")
	}
	if config.AdvisorID != "security-advisor" {
		t.Errorf("AdvisorID = %q, want security-advisor", config.AdvisorID)
	}
}

func TestConfigValidate(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "harness-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "empty working dir",
			config:  Config{},
			wantErr: true,
		},
		{
			name:    "non-existent directory",
			config:  Config{WorkingDir: "/nonexistent/path"},
			wantErr: true,
		},
		{
			name:    "valid directory",
			config:  Config{WorkingDir: tmpDir},
			wantErr: false,
		},
		{
			name:    "invalid temperature",
			config:  Config{WorkingDir: tmpDir, Temperature: 1.5},
			wantErr: true,
		},
		{
			name:    "valid temperature",
			config:  Config{WorkingDir: tmpDir, Temperature: 0.7},
			wantErr: false,
		},
		{
			name:    "zero temperature (default)",
			config:  Config{WorkingDir: tmpDir, Temperature: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultHarnessDefinitions(t *testing.T) {
	defs := DefaultHarnessDefinitions()

	if len(defs) < 3 {
		t.Errorf("Expected at least 3 default definitions, got %d", len(defs))
	}

	// Verify Claude Code is included
	var foundClaude bool
	for _, def := range defs {
		if def.Name == "claude-code" {
			foundClaude = true
			if def.Command != "claude" {
				t.Errorf("Claude command = %q, want claude", def.Command)
			}
			if len(def.Models) == 0 {
				t.Error("Claude should have models defined")
			}
			if !def.Features.Hooks {
				t.Error("Claude should support hooks")
			}
			if !def.Features.MCP {
				t.Error("Claude should support MCP")
			}
		}
	}

	if !foundClaude {
		t.Error("Claude Code definition not found")
	}
}

func TestHarnessDefinitionModels(t *testing.T) {
	defs := DefaultHarnessDefinitions()

	for _, def := range defs {
		if def.Name == "claude-code" {
			// Check that exactly one model is default
			defaultCount := 0
			for _, m := range def.Models {
				if m.Default {
					defaultCount++
				}
			}
			if defaultCount != 1 {
				t.Errorf("Expected exactly 1 default model, got %d", defaultCount)
			}

			// Check default model matches
			if def.DefaultModel == "" {
				t.Error("DefaultModel should not be empty")
			}
			break
		}
	}
}
