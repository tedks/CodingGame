package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the configuration for starting a harness
type Config struct {
	// Model specifies which model to use (e.g., "opus", "sonnet", "gpt-5-codex")
	Model string

	// WorkingDir is the project directory to operate in
	WorkingDir string

	// SystemPrompt is an optional system prompt override
	SystemPrompt string

	// MCPServers lists MCP server configurations to use
	MCPServers []MCPServer

	// Env contains environment variables (API keys, etc.)
	Env map[string]string

	// OutputFormat specifies the output format (usually "json" for parsing)
	OutputFormat string

	// Verbose enables verbose output if supported
	Verbose bool

	// AdvisorMode indicates this harness is being used for an advisor subagent
	AdvisorMode bool

	// AdvisorID identifies the advisor using this harness (if AdvisorMode is true)
	AdvisorID string

	// MaxTokens limits the response length if supported
	MaxTokens int

	// Temperature controls randomness if supported (0.0-1.0)
	Temperature float64

	// ResumeSession allows resuming a previous session if supported
	ResumeSession string
}

// NewConfig creates a new config with sensible defaults
func NewConfig(workingDir string) Config {
	return Config{
		WorkingDir:   workingDir,
		OutputFormat: "json",
		Env:          make(map[string]string),
		MCPServers:   make([]MCPServer, 0),
	}
}

// WithModel sets the model and returns the config for chaining
func (c Config) WithModel(model string) Config {
	c.Model = model
	return c
}

// WithSystemPrompt sets the system prompt and returns the config
func (c Config) WithSystemPrompt(prompt string) Config {
	c.SystemPrompt = prompt
	return c
}

// WithEnv adds an environment variable and returns the config
func (c Config) WithEnv(key, value string) Config {
	if c.Env == nil {
		c.Env = make(map[string]string)
	}
	c.Env[key] = value
	return c
}

// WithMCPServer adds an MCP server and returns the config
func (c Config) WithMCPServer(server MCPServer) Config {
	c.MCPServers = append(c.MCPServers, server)
	return c
}

// ForAdvisor configures the harness for advisor mode
func (c Config) ForAdvisor(advisorID string) Config {
	c.AdvisorMode = true
	c.AdvisorID = advisorID
	return c
}

// Validate checks that the config is valid
func (c Config) Validate() error {
	if c.WorkingDir == "" {
		return fmt.Errorf("working directory is required")
	}

	// Check working directory exists
	info, err := os.Stat(c.WorkingDir)
	if err != nil {
		return fmt.Errorf("working directory error: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("working directory is not a directory: %s", c.WorkingDir)
	}

	// Validate temperature if explicitly set (non-zero).
	// Zero means "use default" and is always allowed.
	if c.Temperature != 0 && (c.Temperature < 0 || c.Temperature > 1) {
		return fmt.Errorf("temperature must be between 0 and 1, got %f", c.Temperature)
	}

	return nil
}

// HarnessDefinition describes a harness and its configuration options
type HarnessDefinition struct {
	// Name is the harness identifier (e.g., "claude-code")
	Name string `json:"name"`

	// DisplayName is the human-readable name
	DisplayName string `json:"display_name"`

	// Description describes the harness
	Description string `json:"description"`

	// Command is the CLI command to run
	Command string `json:"command"`

	// DefaultModel is the default model for this harness
	DefaultModel string `json:"default_model"`

	// Models lists available models
	Models []ModelDefinition `json:"models"`

	// Features lists supported features
	Features HarnessFeatures `json:"features"`
}

// ModelDefinition describes a model option
type ModelDefinition struct {
	ID          string `json:"id"`
	Name        string `json:"display_name"`
	Description string `json:"description"`
	Default     bool   `json:"default"`
}

// HarnessFeatures describes what features a harness supports
type HarnessFeatures struct {
	Hooks     bool `json:"hooks"`
	MCP       bool `json:"mcp"`
	Streaming bool `json:"streaming"`
	Resume    bool `json:"resume"`
}

// LoadHarnessDefinitions loads harness definitions from a JSON file
func LoadHarnessDefinitions(path string) ([]HarnessDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading harness definitions: %w", err)
	}

	var defs []HarnessDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		return nil, fmt.Errorf("parsing harness definitions: %w", err)
	}

	return defs, nil
}

// DefaultHarnessDefinitions returns the built-in harness definitions
func DefaultHarnessDefinitions() []HarnessDefinition {
	return []HarnessDefinition{
		{
			Name:         "claude-code",
			DisplayName:  "Claude Code",
			Description:  "Anthropic's official CLI for Claude",
			Command:      "claude",
			DefaultModel: "sonnet",
			Models: []ModelDefinition{
				{ID: "opus", Name: "Claude Opus", Description: "Most capable model", Default: false},
				{ID: "sonnet", Name: "Claude Sonnet", Description: "Balanced performance", Default: true},
				{ID: "haiku", Name: "Claude Haiku", Description: "Fastest responses", Default: false},
			},
			Features: HarnessFeatures{
				Hooks:     true,
				MCP:       true,
				Streaming: true,
				Resume:    true,
			},
		},
		{
			Name:         "codex",
			DisplayName:  "Codex CLI",
			Description:  "OpenAI's coding assistant",
			Command:      "codex",
			DefaultModel: "gpt-4.1",
			Models: []ModelDefinition{
				{ID: "o3", Name: "O3", Description: "Most capable reasoning model", Default: false},
				{ID: "gpt-4.1", Name: "GPT-4.1", Description: "Latest GPT-4 model", Default: true},
			},
			Features: HarnessFeatures{
				Hooks:     false,
				MCP:       false,
				Streaming: true,
				Resume:    false,
			},
		},
		{
			Name:         "gemini",
			DisplayName:  "Gemini CLI",
			Description:  "Google's AI coding assistant",
			Command:      "gemini",
			DefaultModel: "gemini-2.5-pro",
			Models: []ModelDefinition{
				{ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Description: "Most capable Gemini", Default: true},
				{ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Description: "Fast responses", Default: false},
			},
			Features: HarnessFeatures{
				Hooks:     false,
				MCP:       false,
				Streaming: true,
				Resume:    false,
			},
		},
	}
}

// FindHarnessConfig looks for a .codinggame/harness.json config file
func FindHarnessConfig(projectPath string) (string, error) {
	configPath := filepath.Join(projectPath, ".codinggame", "harness.json")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	// Check home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil // No config found, not an error
	}

	configPath = filepath.Join(home, ".config", "codinggame", "harness.json")
	if _, err := os.Stat(configPath); err == nil {
		return configPath, nil
	}

	return "", nil // No config found
}
