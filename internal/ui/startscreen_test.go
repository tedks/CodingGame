package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tedks/CodingGame/internal/testutil"
)

func TestNewStartScreen(t *testing.T) {
	var completedConfig GameConfig
	onComplete := func(cfg GameConfig) {
		completedConfig = cfg
	}

	ss := NewStartScreen(800, 600, onComplete)

	if ss == nil {
		t.Fatal("expected non-nil start screen")
	}
	if ss.width != 800 {
		t.Errorf("expected width 800, got %d", ss.width)
	}
	if ss.height != 600 {
		t.Errorf("expected height 600, got %d", ss.height)
	}
	if ss.state != StateMainMenu {
		t.Errorf("expected initial state StateMainMenu, got %d", ss.state)
	}
	if ss.mainMenu == nil {
		t.Error("expected mainMenu to be initialized")
	}
	if ss.harnessMenu == nil {
		t.Error("expected harnessMenu to be initialized")
	}
	if ss.modelMenu == nil {
		t.Error("expected modelMenu to be initialized")
	}
	if ss.projectMenu == nil {
		t.Error("expected projectMenu to be initialized")
	}

	// Verify the callback is stored
	if ss.onComplete == nil {
		t.Error("expected onComplete callback to be stored")
	}

	// Suppress unused variable warning
	_ = completedConfig
}

func TestStartScreen_OnEnter(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	// Change state
	ss.state = StateModelSelect
	ss.mainMenu.SelectedIndex = 1

	// OnEnter should reset
	ss.OnEnter()

	if ss.state != StateMainMenu {
		t.Errorf("expected state to reset to StateMainMenu, got %d", ss.state)
	}
	if ss.mainMenu.SelectedIndex != 0 {
		t.Errorf("expected selection to reset to 0, got %d", ss.mainMenu.SelectedIndex)
	}
}

func TestStartScreen_OnExit(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	// Should not panic
	ss.OnExit()
}

func TestStartScreen_SetRecentProjects(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	projects := []string{
		"/home/user/project1",
		"/home/user/project2",
	}
	ss.SetRecentProjects(projects)

	if len(ss.recentProjects) != 2 {
		t.Errorf("expected 2 recent projects, got %d", len(ss.recentProjects))
	}

	// Project menu should have 3 items: "Enter path..." + 2 recent projects
	if len(ss.projectMenu.Items) != 3 {
		t.Errorf("expected 3 project menu items, got %d", len(ss.projectMenu.Items))
	}

	// First item should be the manual entry option
	if ss.projectMenu.Items[0].Label != "Enter project path..." {
		t.Errorf("expected first item to be 'Enter project path...', got %q",
			ss.projectMenu.Items[0].Label)
	}
}

func TestStartScreen_MainMenuOptions(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	// Should have NEW GAME and CONTINUE
	if len(ss.mainMenu.Items) != 2 {
		t.Fatalf("expected 2 main menu items, got %d", len(ss.mainMenu.Items))
	}

	if ss.mainMenu.Items[0].Label != "NEW GAME" {
		t.Errorf("expected first item 'NEW GAME', got %q", ss.mainMenu.Items[0].Label)
	}
	if ss.mainMenu.Items[1].Label != "CONTINUE" {
		t.Errorf("expected second item 'CONTINUE', got %q", ss.mainMenu.Items[1].Label)
	}
}

func TestStartScreen_HarnessMenuOptions(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	// Should have harness options from the registry
	if len(ss.harnessMenu.Items) < 1 {
		t.Fatal("expected at least 1 harness option")
	}

	// Find Claude Code item by value
	var claudeCodeItem *MenuItem
	for _, item := range ss.harnessMenu.Items {
		if item.Value == "claude-code" {
			claudeCodeItem = item
			break
		}
	}

	if claudeCodeItem == nil {
		t.Fatal("expected Claude Code harness option")
	}

	// Claude Code enabled status depends on whether CLI is installed
	// In most dev environments, it will be installed
	// The test just verifies the item exists with correct value
	if claudeCodeItem.Value != "claude-code" {
		t.Errorf("expected value 'claude-code', got %q", claudeCodeItem.Value)
	}

	// Verify that disabled items have "(not installed)" in label
	for _, item := range ss.harnessMenu.Items {
		if !item.Enabled {
			if item.Label == "" || item.Label[len(item.Label)-1] != ')' {
				// Disabled items should indicate why (not installed)
				t.Logf("disabled item %q doesn't indicate reason", item.Label)
			}
		}
	}
}

func TestStartScreen_ModelMenuOptions(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)

	// Should have Opus, Sonnet, Haiku
	if len(ss.modelMenu.Items) != 3 {
		t.Fatalf("expected 3 model options, got %d", len(ss.modelMenu.Items))
	}

	expectedModels := []struct {
		value   string
		enabled bool
	}{
		{"opus", true},
		{"sonnet", true},
		{"haiku", true},
	}

	for i, expected := range expectedModels {
		item := ss.modelMenu.Items[i]
		if item.Value != expected.value {
			t.Errorf("expected model %d value %q, got %q", i, expected.value, item.Value)
		}
		if item.Enabled != expected.enabled {
			t.Errorf("expected model %d enabled=%v, got %v", i, expected.enabled, item.Enabled)
		}
	}
}

func TestStartScreenState_Constants(t *testing.T) {
	// Verify state constants are distinct
	states := []StartScreenState{
		StateMainMenu,
		StateHarnessSelect,
		StateModelSelect,
		StateProjectSelect,
	}

	seen := make(map[StartScreenState]bool)
	for _, s := range states {
		if seen[s] {
			t.Errorf("duplicate state value: %d", s)
		}
		seen[s] = true
	}
}

func TestGameConfig_ZeroValue(t *testing.T) {
	var cfg GameConfig

	if cfg.Harness != "" {
		t.Errorf("expected empty Harness, got %q", cfg.Harness)
	}
	if cfg.Model != "" {
		t.Errorf("expected empty Model, got %q", cfg.Model)
	}
	if cfg.ProjectPath != "" {
		t.Errorf("expected empty ProjectPath, got %q", cfg.ProjectPath)
	}
}

// Tests with TestInputSource for input injection

func TestStartScreen_BackspaceEmptyInput(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)
	ss.projectInputMode = true
	ss.projectInput = ""

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	source.QueueKeyPress(ebiten.KeyBackspace)
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if ss.projectInput != "" {
		t.Errorf("expected projectInput to remain empty, got %q", ss.projectInput)
	}
}

func TestStartScreen_UnicodeInput(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)
	ss.projectInputMode = true
	ss.projectInput = ""

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	source.QueueCharInput('世', '界', '🌍')
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	expected := "世界🌍"
	if ss.projectInput != expected {
		t.Errorf("expected projectInput=%q, got %q", expected, ss.projectInput)
	}
}

func TestStartScreen_EnterEmptyPath(t *testing.T) {
	var completedConfig *GameConfig
	onComplete := func(cfg GameConfig) {
		completedConfig = &cfg
	}

	ss := NewStartScreen(800, 600, onComplete)
	ss.projectInputMode = true
	ss.projectInput = "   "

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if completedConfig != nil {
		t.Error("onComplete should not be called for whitespace-only path")
	}
	if !ss.projectInputMode {
		t.Error("should remain in projectInputMode when path is empty")
	}
}

func TestStartScreen_TextInputSequence(t *testing.T) {
	var completedConfig *GameConfig
	onComplete := func(cfg GameConfig) {
		completedConfig = &cfg
	}

	ss := NewStartScreen(800, 600, onComplete)
	ss.projectInputMode = true
	ss.projectInput = ""

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	for _, char := range "/home/user/project" {
		source.QueueCharInput(char)
		source.AdvanceFrame()
		_, err := ss.Update()
		if err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
	}

	if ss.projectInput != "/home/user/project" {
		t.Errorf("expected projectInput=%q, got %q", "/home/user/project", ss.projectInput)
	}

	source.QueueKeyPress(ebiten.KeyEnter)
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if completedConfig == nil {
		t.Fatal("expected onComplete to be called")
	}
	if completedConfig.ProjectPath != "/home/user/project" {
		t.Errorf("expected ProjectPath=%q, got %q", "/home/user/project", completedConfig.ProjectPath)
	}
}

func TestStartScreen_EscapeCancelsInput(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)
	ss.projectInputMode = true
	ss.projectInput = "/some/path"

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	source.QueueKeyPress(ebiten.KeyEscape)
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if ss.projectInputMode {
		t.Error("expected projectInputMode=false after Escape")
	}
}

func TestStartScreen_BackspaceDeletesCharacter(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)
	ss.projectInputMode = true
	ss.projectInput = "test"

	source := testutil.NewTestInputSource()
	ss.SetInputSource(source)

	source.QueueKeyPress(ebiten.KeyBackspace)
	source.AdvanceFrame()

	_, err := ss.Update()
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if ss.projectInput != "tes" {
		t.Errorf("expected projectInput=%q after backspace, got %q", "tes", ss.projectInput)
	}
}

func TestStartScreen_SetInputSource_Nil(t *testing.T) {
	ss := NewStartScreen(800, 600, nil)
	ss.SetInputSource(nil)

	if ss.inputSource == nil {
		t.Error("inputSource should not be nil after SetInputSource(nil)")
	}
}
