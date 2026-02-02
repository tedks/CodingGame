package advisor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/tedks/CodingGame/internal/harness"
)

// Integration tests for the advisor system

func TestIntegration_FullAdvisorLifecycle(t *testing.T) {
	// Create pool and load configs
	pool := NewPool()

	configs := []Config{
		{
			ID:            "security",
			Name:          "Security Advisor",
			Icon:          "shield",
			SystemPrompt:  "Analyze for security vulnerabilities",
			Trigger:       TriggerOnFileChange,
			FocusPatterns: []string{"*.go", "*.ts"},
		},
		{
			ID:           "refactoring",
			Name:         "Refactoring Advisor",
			Icon:         "wrench",
			SystemPrompt: "Find code smells",
			Trigger:      TriggerManual,
		},
	}

	if err := pool.LoadFromConfig(configs); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Verify advisors loaded
	if pool.Count() != 2 {
		t.Errorf("pool.Count() = %d, want 2", pool.Count())
	}

	// Simulate file change triggering security advisor
	triggered := pool.TriggerOnFileChange("auth/login.go")
	if len(triggered) != 1 {
		t.Fatalf("TriggerOnFileChange() returned %d advisors, want 1", len(triggered))
	}
	if triggered[0].ID() != "security" {
		t.Errorf("triggered advisor ID = %q, want security", triggered[0].ID())
	}

	// Start analysis
	securityAdvisor := pool.Get("security")
	if !securityAdvisor.StartAnalysis() {
		t.Fatal("StartAnalysis() returned false")
	}

	// Verify state changed
	if securityAdvisor.State() != StateThinking {
		t.Errorf("after StartAnalysis(), State() = %q, want %q", securityAdvisor.State(), StateThinking)
	}

	// Generate insight
	insight := BuildInsight("security").
		Title("Hardcoded API Key").
		Description("Found hardcoded API key in source code").
		Severity(SeverityCritical).
		Category(CategorySecurity).
		Location("auth/login.go", 42, 10).
		Suggestion("Use environment variable", "const KEY = 'abc123'", "const KEY = os.Getenv('API_KEY')").
		Build()

	securityAdvisor.AddInsight(insight)

	// Complete analysis
	securityAdvisor.CompleteAnalysis(2*time.Second, 1500, 300, nil)

	// Verify state
	if securityAdvisor.State() != StateHasInsights {
		t.Errorf("after completion with insights, State() = %q, want %q", securityAdvisor.State(), StateHasInsights)
	}

	// Check pool-level insights
	critical := pool.GetCriticalInsights()
	if len(critical) != 1 {
		t.Errorf("GetCriticalInsights() count = %d, want 1", len(critical))
	}

	// Accept insight
	securityAdvisor.MarkInsightAccepted(insight.ID)

	// Verify metrics
	metrics := securityAdvisor.Metrics()
	if metrics.AcceptedInsights != 1 {
		t.Errorf("AcceptedInsights = %d, want 1", metrics.AcceptedInsights)
	}
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
}

func TestIntegration_ConfigParsingAndPoolLoading(t *testing.T) {
	// Parse config from JSON
	json := `{
		"advisors": [
			{
				"id": "testing",
				"name": "Testing Advisor",
				"icon": "check",
				"system_prompt": "Analyze test coverage and quality",
				"trigger": "on_file_change",
				"focus_patterns": ["*_test.go", "*.test.ts"]
			},
			{
				"id": "docs",
				"name": "Documentation Advisor",
				"icon": "book",
				"system_prompt": "Check documentation completeness",
				"trigger": "manual"
			}
		]
	}`

	configs, err := ParseConfig(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}

	// Load into pool
	pool := NewPool()
	if err := pool.LoadFromConfig(configs); err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	// Verify
	testingAdvisor := pool.Get("testing")
	if testingAdvisor == nil {
		t.Fatal("testing advisor not found")
	}
	if testingAdvisor.Icon() != "check" {
		t.Errorf("testing advisor Icon() = %q, want check", testingAdvisor.Icon())
	}
	if testingAdvisor.Trigger() != TriggerOnFileChange {
		t.Errorf("testing advisor Trigger() = %q, want on_file_change", testingAdvisor.Trigger())
	}

	// Verify file matching
	if !testingAdvisor.MatchesFile("api_test.go") {
		t.Error("testing advisor should match api_test.go")
	}
	if testingAdvisor.MatchesFile("api.go") {
		t.Error("testing advisor should not match api.go")
	}
}

func TestIntegration_InsightFilteringAcrossAdvisors(t *testing.T) {
	pool := NewPool()

	// Create advisors
	security := New(Config{ID: "security", Name: "Security", SystemPrompt: "p", Trigger: TriggerManual}, 0, 0)
	testing := New(Config{ID: "testing", Name: "Testing", SystemPrompt: "p", Trigger: TriggerManual}, 0, 0)

	pool.Add(security)
	pool.Add(testing)

	// Add various insights
	security.AddInsight(NewInsight("security", "SQL Injection", "desc", SeverityCritical, CategorySecurity))
	security.AddInsight(NewInsight("security", "XSS Risk", "desc", SeverityWarning, CategorySecurity))
	testing.AddInsight(NewInsight("testing", "Low Coverage", "desc", SeverityWarning, CategoryTesting))
	testing.AddInsight(NewInsight("testing", "Flaky Test", "desc", SeverityInfo, CategoryTesting))

	// Get all insights
	all := pool.GetAllInsights()
	if len(all) != 4 {
		t.Errorf("GetAllInsights() count = %d, want 4", len(all))
	}

	// Filter by severity
	critical := FilterInsights(all).BySeverity(SeverityCritical).Results()
	if len(critical) != 1 {
		t.Errorf("critical insights count = %d, want 1", len(critical))
	}

	// Filter by category
	securityInsights := FilterInsights(all).ByCategory(CategorySecurity).Results()
	if len(securityInsights) != 2 {
		t.Errorf("security insights count = %d, want 2", len(securityInsights))
	}

	// Chained filter
	criticalSecurity := FilterInsights(all).
		ByCategory(CategorySecurity).
		BySeverity(SeverityCritical).
		Results()
	if len(criticalSecurity) != 1 {
		t.Errorf("critical security insights count = %d, want 1", len(criticalSecurity))
	}
}

func TestIntegration_MultipleAnalysisRuns(t *testing.T) {
	advisor := New(Config{
		ID:           "perf",
		Name:         "Performance",
		SystemPrompt: "Analyze performance",
		Trigger:      TriggerManual,
	}, 0, 0)

	// Run multiple analyses
	runs := []struct {
		duration  time.Duration
		tokensIn  int64
		tokensOut int64
		hasError  bool
	}{
		{1 * time.Second, 500, 100, false},
		{3 * time.Second, 1000, 200, false},
		{500 * time.Millisecond, 200, 50, true}, // Error case
		{2 * time.Second, 800, 150, false},
	}

	for i, run := range runs {
		advisor.StartAnalysis()

		var err error
		if run.hasError {
			err = &testError{msg: "analysis failed"}
		}

		advisor.CompleteAnalysis(run.duration, run.tokensIn, run.tokensOut, err)

		if advisor.IsRunning() {
			t.Errorf("run %d: advisor still running after completion", i)
		}
	}

	// Verify metrics
	metrics := advisor.Metrics()
	if metrics.TotalRuns != 4 {
		t.Errorf("TotalRuns = %d, want 4", metrics.TotalRuns)
	}
	if metrics.SuccessCount != 3 {
		t.Errorf("SuccessCount = %d, want 3", metrics.SuccessCount)
	}
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", metrics.ErrorCount)
	}

	// Token totals
	expectedTokensIn := int64(500 + 1000 + 200 + 800)
	if metrics.TotalTokensIn != expectedTokensIn {
		t.Errorf("TotalTokensIn = %d, want %d", metrics.TotalTokensIn, expectedTokensIn)
	}

	// Duration stats
	if metrics.MinDuration != 500*time.Millisecond {
		t.Errorf("MinDuration = %v, want 500ms", metrics.MinDuration)
	}
	if metrics.MaxDuration != 3*time.Second {
		t.Errorf("MaxDuration = %v, want 3s", metrics.MaxDuration)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestIntegration_FileChangeTriggering(t *testing.T) {
	pool := NewPool()

	// Different advisors watching different file patterns
	goAdvisor := New(Config{
		ID:            "go-linter",
		Name:          "Go Linter",
		SystemPrompt:  "Lint Go code",
		Trigger:       TriggerOnFileChange,
		FocusPatterns: []string{"*.go"},
	}, 0, 0)

	tsAdvisor := New(Config{
		ID:            "ts-checker",
		Name:          "TypeScript Checker",
		SystemPrompt:  "Check TypeScript",
		Trigger:       TriggerOnFileChange,
		FocusPatterns: []string{"*.ts", "*.tsx"},
	}, 0, 0)

	manualAdvisor := New(Config{
		ID:            "manual",
		Name:          "Manual",
		SystemPrompt:  "Manual only",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*"},
	}, 0, 0)

	pool.Add(goAdvisor)
	pool.Add(tsAdvisor)
	pool.Add(manualAdvisor)

	// Test triggering
	tests := []struct {
		file        string
		expectedIDs []string
	}{
		{"main.go", []string{"go-linter"}},
		{"app.ts", []string{"ts-checker"}},
		{"component.tsx", []string{"ts-checker"}},
		{"styles.css", []string{}}, // No match
		{"readme.md", []string{}},  // No match
	}

	for _, tt := range tests {
		triggered := pool.TriggerOnFileChange(tt.file)
		if len(triggered) != len(tt.expectedIDs) {
			t.Errorf("TriggerOnFileChange(%s) returned %d advisors, want %d",
				tt.file, len(triggered), len(tt.expectedIDs))
			continue
		}

		for _, expected := range tt.expectedIDs {
			found := false
			for _, a := range triggered {
				if a.ID() == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("TriggerOnFileChange(%s) missing expected advisor %q", tt.file, expected)
			}
		}
	}
}

func TestIntegration_DefaultAdvisorsAreValid(t *testing.T) {
	pool := NewPool()

	// Load default configs
	configs := DefaultConfigs()
	if err := pool.LoadFromConfig(configs); err != nil {
		t.Fatalf("LoadFromConfig(DefaultConfigs()) error = %v", err)
	}

	// All defaults should work
	for _, advisor := range pool.GetAll() {
		// Start and complete analysis
		if !advisor.StartAnalysis() {
			t.Errorf("advisor %q: StartAnalysis() failed", advisor.ID())
		}

		advisor.AddInsight(NewInsight(
			advisor.ID(),
			"Test Insight",
			"Test description",
			SeverityInfo,
			CategoryGeneral,
		))

		advisor.CompleteAnalysis(time.Second, 100, 50, nil)

		// Verify state
		if advisor.State() != StateHasInsights {
			t.Errorf("advisor %q: unexpected state %q", advisor.ID(), advisor.State())
		}

		// Clear for next iteration
		advisor.ClearInsights()
	}
}

func TestIntegration_PoolStartStopWithBackgroundAdvisor(t *testing.T) {
	pool := NewPool()

	// Add a background advisor
	background := New(Config{
		ID:                     "monitor",
		Name:                   "Monitor",
		SystemPrompt:           "Monitor",
		Trigger:                TriggerBackground,
		BackgroundIntervalSecs: 1, // 1 second for testing
	}, 0, 0)

	pool.Add(background)

	// Start pool
	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !pool.IsRunning() {
		t.Error("after Start(), IsRunning() = false")
	}

	// Stop pool (tests clean start/stop lifecycle)
	if err := pool.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if pool.IsRunning() {
		t.Error("after Stop(), IsRunning() = true")
	}
}

func TestIntegration_InsightUserActions(t *testing.T) {
	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "Test",
		Trigger:      TriggerManual,
	}, 0, 0)

	// Add insights
	insight1 := NewInsight("test", "Insight 1", "desc", SeverityInfo, CategoryGeneral)
	insight2 := NewInsight("test", "Insight 2", "desc", SeverityWarning, CategoryGeneral)
	insight3 := NewInsight("test", "Insight 3", "desc", SeverityCritical, CategoryGeneral)

	advisor.AddInsight(insight1)
	advisor.AddInsight(insight2)
	advisor.AddInsight(insight3)

	// All should be pending
	if advisor.UnreadInsightCount() != 3 {
		t.Errorf("initial UnreadInsightCount() = %d, want 3", advisor.UnreadInsightCount())
	}

	// Accept one
	advisor.MarkInsightAccepted(insight1.ID)
	if advisor.UnreadInsightCount() != 2 {
		t.Errorf("after accept, UnreadInsightCount() = %d, want 2", advisor.UnreadInsightCount())
	}

	// Reject one
	advisor.MarkInsightRejected(insight2.ID)
	if advisor.UnreadInsightCount() != 1 {
		t.Errorf("after reject, UnreadInsightCount() = %d, want 1", advisor.UnreadInsightCount())
	}

	// Verify acceptance rate
	// 1 accepted / (1 accepted + 1 rejected) = 50%
	if advisor.AcceptanceRate() != 50.0 {
		t.Errorf("AcceptanceRate() = %v, want 50.0", advisor.AcceptanceRate())
	}

	// Clear all
	advisor.ClearInsights()
	if advisor.UnreadInsightCount() != 0 {
		t.Errorf("after clear, UnreadInsightCount() = %d, want 0", advisor.UnreadInsightCount())
	}
}

func TestIntegration_PoolMetricsAggregation(t *testing.T) {
	pool := NewPool()

	// Create multiple advisors with different metrics
	advisor1 := New(Config{ID: "a1", Name: "A1", SystemPrompt: "p", Trigger: TriggerManual}, 0, 0)
	advisor2 := New(Config{ID: "a2", Name: "A2", SystemPrompt: "p", Trigger: TriggerManual}, 0, 0)

	// Run some analyses
	advisor1.StartAnalysis()
	advisor1.AddInsight(NewInsight("a1", "I1", "d", SeverityInfo, CategoryGeneral))
	advisor1.CompleteAnalysis(time.Second, 1000, 200, nil)
	advisor1.MarkInsightAccepted(advisor1.Insights()[0].ID)

	advisor2.StartAnalysis()
	advisor2.AddInsight(NewInsight("a2", "I2", "d", SeverityInfo, CategoryGeneral))
	advisor2.AddInsight(NewInsight("a2", "I3", "d", SeverityInfo, CategoryGeneral))
	advisor2.CompleteAnalysis(2*time.Second, 2000, 400, nil)
	advisor2.MarkInsightRejected(advisor2.Insights()[0].ID)

	pool.Add(advisor1)
	pool.Add(advisor2)

	metrics := pool.AggregateMetrics()

	if metrics.AdvisorCount != 2 {
		t.Errorf("AdvisorCount = %d, want 2", metrics.AdvisorCount)
	}
	if metrics.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", metrics.TotalRuns)
	}
	if metrics.TotalTokensIn != 3000 {
		t.Errorf("TotalTokensIn = %d, want 3000", metrics.TotalTokensIn)
	}
	if metrics.TotalTokensOut != 600 {
		t.Errorf("TotalTokensOut = %d, want 600", metrics.TotalTokensOut)
	}
	if metrics.TotalInsights != 3 {
		t.Errorf("TotalInsights = %d, want 3", metrics.TotalInsights)
	}
	if metrics.AcceptedInsights != 1 {
		t.Errorf("AcceptedInsights = %d, want 1", metrics.AcceptedInsights)
	}
	if metrics.RejectedInsights != 1 {
		t.Errorf("RejectedInsights = %d, want 1", metrics.RejectedInsights)
	}
}

// RunAdvisor integration tests

// TestIntegration_RunAdvisor_HappyPath tests the full RunAdvisor flow with a mock harness
func TestIntegration_RunAdvisor_HappyPath(t *testing.T) {
	// Create a temp directory for working dir
	tmpDir := t.TempDir()

	// Setup mock harness
	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })

	// Setup pool
	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetMainHarness("mock")
	pool.SetWorkingDir(tmpDir)

	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pool.Stop()

	// Create advisor
	advisor := New(Config{
		ID:           "test",
		Name:         "Test Advisor",
		SystemPrompt: "Analyze code for issues",
		Trigger:      TriggerManual,
	}, 0, 0)

	if err := pool.Add(advisor); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Simulate turn complete in background
	go func() {
		// Wait a bit for SendPrompt to be called
		time.Sleep(20 * time.Millisecond)
		mock.SimulateTurnComplete()
	}()

	// Run advisor
	ctx := context.Background()
	err := pool.RunAdvisor(ctx, advisor, []string{"file1.go", "file2.go"})

	// Verify
	if err != nil {
		t.Fatalf("RunAdvisor() error = %v", err)
	}

	// Check prompt was sent
	if mock.PromptCount() != 1 {
		t.Errorf("PromptCount() = %d, want 1", mock.PromptCount())
	}

	// Check advisor state is idle after completion
	if advisor.State() != StateIdle {
		t.Errorf("State() = %v, want %v", advisor.State(), StateIdle)
	}

	// Check metrics were updated
	metrics := advisor.Metrics()
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}
}

// TestIntegration_RunAdvisor_NoRegistry tests RunAdvisor returns error when registry not configured
func TestIntegration_RunAdvisor_NoRegistry(t *testing.T) {
	pool := NewPool()
	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pool.Stop()

	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)
	pool.Add(advisor)

	err := pool.RunAdvisor(context.Background(), advisor, nil)
	if err == nil {
		t.Error("RunAdvisor() expected error when registry not configured")
	}
	if !strings.Contains(err.Error(), "registry not configured") {
		t.Errorf("RunAdvisor() error = %v, want error containing 'registry not configured'", err)
	}
}

// TestIntegration_RunAdvisor_NoHarness tests RunAdvisor returns error when no harness configured
func TestIntegration_RunAdvisor_NoHarness(t *testing.T) {
	tmpDir := t.TempDir()

	registry := harness.NewRegistry()
	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetWorkingDir(tmpDir)
	// Note: NOT setting main harness

	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pool.Stop()

	// Advisor with no harness configured
	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
		// No HarnessName set
	}, 0, 0)
	pool.Add(advisor)

	err := pool.RunAdvisor(context.Background(), advisor, nil)
	if err == nil {
		t.Error("RunAdvisor() expected error when no harness configured")
	}
	if !strings.Contains(err.Error(), "no harness configured") {
		t.Errorf("RunAdvisor() error = %v, want error containing 'no harness configured'", err)
	}
}

// TestIntegration_RunAdvisor_ContextCancellation tests RunAdvisor handles context cancellation
func TestIntegration_RunAdvisor_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })

	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetMainHarness("mock")
	pool.SetWorkingDir(tmpDir)

	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pool.Stop()

	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)
	pool.Add(advisor)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel shortly after start
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := pool.RunAdvisor(ctx, advisor, []string{"file.go"})

	// Should return context error
	if !errors.Is(err, context.Canceled) {
		t.Errorf("RunAdvisor() error = %v, want context.Canceled", err)
	}

	// Advisor should be in error state or have recorded the error
	if advisor.LastError() == nil {
		t.Error("LastError() should be set after cancellation")
	}
}

// TestIntegration_RunAdvisorAsync_StopWaits tests that Pool.Stop() waits for async advisors
func TestIntegration_RunAdvisorAsync_StopWaits(t *testing.T) {
	tmpDir := t.TempDir()

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })

	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetMainHarness("mock")
	pool.SetWorkingDir(tmpDir)

	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)
	pool.Add(advisor)

	// Start async advisor
	pool.RunAdvisorAsync(context.Background(), advisor, []string{"file.go"})

	// Wait a bit for async to start
	time.Sleep(10 * time.Millisecond)

	// Stop should wait for async advisor
	done := make(chan struct{})
	go func() {
		pool.Stop()
		close(done)
	}()

	// Simulate completion
	go func() {
		time.Sleep(20 * time.Millisecond)
		mock.SimulateTurnComplete()
	}()

	select {
	case <-done:
		// Expected: Stop completed after advisor finished
	case <-time.After(time.Second):
		t.Fatal("Stop() blocked too long, should have completed after advisor finished")
	}
}

// TestIntegration_RunAdvisorAsync_NotRunning tests that RunAdvisorAsync does nothing when pool not running
func TestIntegration_RunAdvisorAsync_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()

	mock := harness.NewMockHarness()
	registry := harness.NewRegistry()
	registry.Register("mock", func() harness.Harness { return mock })

	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetMainHarness("mock")
	pool.SetWorkingDir(tmpDir)
	// Note: NOT starting pool

	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)
	pool.Add(advisor)

	// This should do nothing since pool not running
	pool.RunAdvisorAsync(context.Background(), advisor, []string{"file.go"})

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Mock should not have received any prompts
	if mock.PromptCount() != 0 {
		t.Errorf("PromptCount() = %d, want 0 (pool not running)", mock.PromptCount())
	}
}

// TestIntegration_RunAdvisor_AdvisorSpecificHarness tests using advisor-specific harness
func TestIntegration_RunAdvisor_AdvisorSpecificHarness(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two different mock harnesses
	mainMock := harness.NewMockHarness()
	advisorMock := harness.NewMockHarness()

	registry := harness.NewRegistry()
	registry.Register("main-harness", func() harness.Harness { return mainMock })
	registry.Register("advisor-harness", func() harness.Harness { return advisorMock })

	pool := NewPool()
	pool.SetHarnessRegistry(registry)
	pool.SetMainHarness("main-harness")
	pool.SetWorkingDir(tmpDir)

	if err := pool.Start(); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer pool.Stop()

	// Advisor with its own harness
	advisor := New(Config{
		ID:           "custom",
		Name:         "Custom Harness Advisor",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
		HarnessName:  "advisor-harness", // Use specific harness
	}, 0, 0)
	pool.Add(advisor)

	// Complete analysis in background
	go func() {
		time.Sleep(20 * time.Millisecond)
		advisorMock.SimulateTurnComplete()
	}()

	err := pool.RunAdvisor(context.Background(), advisor, []string{"file.go"})
	if err != nil {
		t.Fatalf("RunAdvisor() error = %v", err)
	}

	// Should have used advisor-specific harness, not main
	if mainMock.PromptCount() != 0 {
		t.Errorf("Main harness PromptCount() = %d, want 0", mainMock.PromptCount())
	}
	if advisorMock.PromptCount() != 1 {
		t.Errorf("Advisor harness PromptCount() = %d, want 1", advisorMock.PromptCount())
	}
}
