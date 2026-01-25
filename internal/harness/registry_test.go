package harness

import (
	"context"
	"testing"
)

// MockHarness is a test harness implementation
type MockHarness struct {
	*BaseHarness
	startCalled bool
	stopCalled  bool
	lastPrompt  string
}

func NewMockHarness() Harness {
	return &MockHarness{
		BaseHarness: NewBaseHarness("mock"),
	}
}

func (m *MockHarness) Start(ctx context.Context, config Config) error {
	m.startCalled = true
	m.SetRunning(true)
	return nil
}

func (m *MockHarness) Stop() error {
	m.stopCalled = true
	m.SetRunning(false)
	m.CloseEvents()
	return nil
}

func (m *MockHarness) SendPrompt(prompt string) error {
	m.lastPrompt = prompt
	return nil
}

func (m *MockHarness) Capabilities() Capabilities {
	return Capabilities{
		SupportedModels: []Model{
			{ID: "test", Name: "Test Model", Default: true},
		},
		SupportsHooks:     true,
		SupportsMCP:       true,
		SupportsStreaming: true,
	}
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

	r.Register("mock", NewMockHarness)

	if !r.IsRegistered("mock") {
		t.Error("mock should be registered")
	}
	if r.IsRegistered("nonexistent") {
		t.Error("nonexistent should not be registered")
	}
}

func TestRegistryCreate(t *testing.T) {
	r := NewRegistry()
	r.Register("mock", NewMockHarness)

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
	r.Register("mock1", NewMockHarness)
	r.Register("mock2", NewMockHarness)

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

	r.RegisterWithDefinition(def, NewMockHarness)

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
	r.Register("claude-code", NewMockHarness)

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
	r.Register("claude-code", NewMockHarness)

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

	b.SetVersion("1.0.0")
	if b.Version() != "1.0.0" {
		t.Errorf("Version() = %q, want 1.0.0", b.Version())
	}

	b.SetRunning(true)
	if !b.IsRunning() {
		t.Error("IsRunning() should be true after SetRunning(true)")
	}
}

func TestMockHarnessLifecycle(t *testing.T) {
	h := NewMockHarness().(*MockHarness)

	if h.IsRunning() {
		t.Error("Should not be running initially")
	}

	config := NewConfig("/tmp")
	err := h.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if !h.startCalled {
		t.Error("Start() should set startCalled")
	}
	if !h.IsRunning() {
		t.Error("Should be running after Start()")
	}

	err = h.SendPrompt("test prompt")
	if err != nil {
		t.Fatalf("SendPrompt() error = %v", err)
	}
	if h.lastPrompt != "test prompt" {
		t.Errorf("lastPrompt = %q, want 'test prompt'", h.lastPrompt)
	}

	err = h.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if !h.stopCalled {
		t.Error("Stop() should set stopCalled")
	}
	if h.IsRunning() {
		t.Error("Should not be running after Stop()")
	}
}
