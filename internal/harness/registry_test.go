package harness

import (
	"context"
	"testing"
)

// mockHarnessFactory returns a factory function for creating MockHarness instances.
// This is used for registry tests that need a HarnessFactory.
func mockHarnessFactory() Harness {
	return NewMockHarness()
}

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()

	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	// Should have default definitions loaded
	defs := r.Defined()
	if len(defs) == 0 {
		t.Error("Registry should have default definitions")
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry()

	r.Register("mock", mockHarnessFactory)

	if !r.IsRegistered("mock") {
		t.Error("mock should be registered")
	}
	if r.IsRegistered("nonexistent") {
		t.Error("nonexistent should not be registered")
	}
}

func TestRegistryCreate(t *testing.T) {
	r := NewRegistry()
	r.Register("mock", mockHarnessFactory)

	h, err := r.Create("mock")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if h.Name() != "mock" {
		t.Errorf("Name() = %q, want mock", h.Name())
	}
}

func TestRegistryCreateUnknown(t *testing.T) {
	r := NewRegistry()

	_, err := r.Create("nonexistent")
	if err == nil {
		t.Error("Create() should return error for unknown harness")
	}
}

func TestRegistryAvailable(t *testing.T) {
	r := NewRegistry()
	r.Register("mock1", mockHarnessFactory)
	r.Register("mock2", mockHarnessFactory)

	available := r.Available()
	if len(available) != 2 {
		t.Errorf("Available() = %v, want 2 items", available)
	}

	hasM1, hasM2 := false, false
	for _, name := range available {
		if name == "mock1" {
			hasM1 = true
		}
		if name == "mock2" {
			hasM2 = true
		}
	}

	if !hasM1 || !hasM2 {
		t.Error("Available() should include both mock1 and mock2")
	}
}

func TestRegistryDefinition(t *testing.T) {
	r := NewRegistry()

	def, ok := r.Definition("claude-code")
	if !ok {
		t.Fatal("Definition(claude-code) should exist")
	}

	if def.Name != "claude-code" {
		t.Errorf("Name = %q, want claude-code", def.Name)
	}
	if def.Command != "claude" {
		t.Errorf("Command = %q, want claude", def.Command)
	}
}

func TestRegistryRegisterWithDefinition(t *testing.T) {
	r := NewRegistry()

	def := HarnessDefinition{
		Name:         "custom",
		DisplayName:  "Custom Harness",
		Description:  "A custom test harness",
		Command:      "custom-cli",
		DefaultModel: "default",
		Models: []ModelDefinition{
			{ID: "default", Name: "Default", Default: true},
		},
	}

	r.RegisterWithDefinition(def, mockHarnessFactory)

	if !r.IsRegistered("custom") {
		t.Error("custom should be registered")
	}

	gotDef, ok := r.Definition("custom")
	if !ok {
		t.Fatal("Definition(custom) should exist")
	}
	if gotDef.DisplayName != "Custom Harness" {
		t.Errorf("DisplayName = %q, want Custom Harness", gotDef.DisplayName)
	}
}

func TestRegistryModels(t *testing.T) {
	r := NewRegistry()

	models := r.Models("claude-code")
	if len(models) == 0 {
		t.Fatal("Models(claude-code) should return models")
	}

	var hasOpus, hasSonnet bool
	for _, m := range models {
		if m.ID == "opus" {
			hasOpus = true
		}
		if m.ID == "sonnet" {
			hasSonnet = true
		}
	}

	if !hasOpus || !hasSonnet {
		t.Error("Claude models should include opus and sonnet")
	}
}

func TestRegistryDefaultModel(t *testing.T) {
	r := NewRegistry()

	defaultModel := r.DefaultModel("claude-code")
	if defaultModel == "" {
		t.Error("DefaultModel(claude-code) should return a model")
	}
}

func TestRegistryInfo(t *testing.T) {
	r := NewRegistry()
	r.Register("claude-code", mockHarnessFactory)

	info := r.Info("claude-code")
	if info == nil {
		t.Fatal("Info(claude-code) should return info")
	}

	if info.Name != "claude-code" {
		t.Errorf("Name = %q, want claude-code", info.Name)
	}
	if !info.Registered {
		t.Error("Registered should be true after registration")
	}
	if info.DisplayName == "" {
		t.Error("DisplayName should not be empty")
	}
}

func TestRegistryInfoUnknown(t *testing.T) {
	r := NewRegistry()

	info := r.Info("nonexistent")
	if info != nil {
		t.Error("Info(nonexistent) should return nil")
	}
}

func TestRegistryAllInfo(t *testing.T) {
	r := NewRegistry()
	r.Register("claude-code", mockHarnessFactory)

	infos := r.AllInfo()
	if len(infos) == 0 {
		t.Error("AllInfo() should return at least one entry")
	}

	var foundClaude bool
	for _, info := range infos {
		if info.Name == "claude-code" {
			foundClaude = true
			if !info.Registered {
				t.Error("claude-code should be registered")
			}
		}
	}

	if !foundClaude {
		t.Error("AllInfo should include claude-code")
	}
}

func TestRegistryGetCapabilities(t *testing.T) {
	r := NewRegistry()

	caps := r.GetCapabilities("claude-code")
	if caps == nil {
		t.Fatal("GetCapabilities(claude-code) should return capabilities")
	}

	if len(caps.SupportedModels) == 0 {
		t.Error("Claude should have supported models")
	}
	if !caps.SupportsHooks {
		t.Error("Claude should support hooks")
	}
	if !caps.SupportsMCP {
		t.Error("Claude should support MCP")
	}
}

func TestRegistryGetCapabilitiesUnknown(t *testing.T) {
	r := NewRegistry()

	caps := r.GetCapabilities("nonexistent")
	if caps != nil {
		t.Error("GetCapabilities(nonexistent) should return nil")
	}
}

func TestBaseHarness(t *testing.T) {
	b := NewBaseHarness("test")

	if b.Name() != "test" {
		t.Errorf("Name() = %q, want test", b.Name())
	}
	if b.Version() != "" {
		t.Errorf("Version() = %q, want empty", b.Version())
	}
	if b.IsRunning() {
		t.Error("IsRunning() should be false initially")
	}
	if b.IsStopped() {
		t.Error("IsStopped() should be false initially")
	}

	b.SetVersion("1.0.0")
	if b.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want 1.0.0", b.Version())
	}

	b.SetRunning(true)
	if !b.IsRunning() {
		t.Error("IsRunning() should be true after SetRunning(true)")
	}

	b.SetRunning(false)
	if b.IsRunning() {
		t.Error("IsRunning() should be false after SetRunning(false)")
	}
	if !b.IsStopped() {
		t.Error("IsStopped() should be true after SetRunning(false)")
	}

	b.SetRunning(true)
	if b.IsRunning() {
		t.Error("IsRunning() should remain false after stop")
	}
}

func TestMockHarnessLifecycle(t *testing.T) {
	h := NewMockHarness()

	if h.IsRunning() {
		t.Error("Should not be running initially")
	}

	config := NewConfig("/tmp")
	err := h.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !h.IsRunning() {
		t.Error("Should be running after Start()")
	}

	err = h.SendPrompt("test prompt")
	if err != nil {
		t.Fatalf("SendPrompt() error = %v", err)
	}
	if h.LastPrompt() != "test prompt" {
		t.Errorf("LastPrompt() = %q, want 'test prompt'", h.LastPrompt())
	}

	err = h.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if h.IsRunning() {
		t.Error("Should not be running after Stop()")
	}
}
