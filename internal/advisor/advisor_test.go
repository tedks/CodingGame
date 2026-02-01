package advisor

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	config := Config{
		ID:            "security",
		Name:          "Security Advisor",
		Icon:          "shield",
		SystemPrompt:  "Analyze security issues",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*.go"},
	}

	a := New(config, 100.0, 200.0)

	if a.ID() != "security" {
		t.Errorf("ID() = %q, want security", a.ID())
	}
	if a.Name() != "Security Advisor" {
		t.Errorf("Name() = %q, want Security Advisor", a.Name())
	}
	if a.Icon() != "shield" {
		t.Errorf("Icon() = %q, want shield", a.Icon())
	}
	if a.SystemPrompt() != "Analyze security issues" {
		t.Errorf("SystemPrompt() = %q, want 'Analyze security issues'", a.SystemPrompt())
	}
	if a.Trigger() != TriggerManual {
		t.Errorf("Trigger() = %q, want manual", a.Trigger())
	}

	x, y := a.Position()
	if x != 100.0 || y != 200.0 {
		t.Errorf("Position() = (%v, %v), want (100, 200)", x, y)
	}

	if a.State() != StateIdle {
		t.Errorf("initial State() = %q, want %q", a.State(), StateIdle)
	}

	if a.IsRunning() {
		t.Error("initial IsRunning() = true, want false")
	}
}

func TestAdvisor_FocusPatterns(t *testing.T) {
	config := Config{
		ID:            "test",
		Name:          "Test",
		SystemPrompt:  "test",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*.go", "*.ts"},
	}

	a := New(config, 0, 0)
	patterns := a.FocusPatterns()

	if len(patterns) != 2 {
		t.Errorf("FocusPatterns() length = %d, want 2", len(patterns))
	}

	// Verify it's a copy
	patterns[0] = "modified"
	if a.FocusPatterns()[0] == "modified" {
		t.Error("FocusPatterns() returned reference instead of copy")
	}
}

func TestAdvisor_Config(t *testing.T) {
	config := Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "test",
		Trigger:      TriggerManual,
	}

	a := New(config, 0, 0)
	gotConfig := a.Config()

	if gotConfig.ID != config.ID {
		t.Errorf("Config().ID = %q, want %q", gotConfig.ID, config.ID)
	}
}

func TestAdvisor_SetPosition(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	a.SetPosition(50.0, 75.0)

	x, y := a.Position()
	if x != 50.0 || y != 75.0 {
		t.Errorf("Position() = (%v, %v), want (50, 75)", x, y)
	}
}

func TestAdvisor_StartAnalysis(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// First start should succeed
	if !a.StartAnalysis() {
		t.Error("first StartAnalysis() returned false, want true")
	}

	if a.State() != StateThinking {
		t.Errorf("after StartAnalysis(), State() = %q, want %q", a.State(), StateThinking)
	}

	if !a.IsRunning() {
		t.Error("after StartAnalysis(), IsRunning() = false, want true")
	}

	// Second start should fail (already running)
	if a.StartAnalysis() {
		t.Error("second StartAnalysis() returned true, want false")
	}
}

func TestAdvisor_CompleteAnalysis_Success(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	a.StartAnalysis()
	a.CompleteAnalysis(5*time.Second, 1000, 500, nil)

	if a.State() != StateIdle {
		t.Errorf("after successful completion, State() = %q, want %q", a.State(), StateIdle)
	}

	if a.IsRunning() {
		t.Error("after completion, IsRunning() = true, want false")
	}

	if a.LastError() != nil {
		t.Errorf("after successful completion, LastError() = %v, want nil", a.LastError())
	}

	metrics := a.Metrics()
	if metrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", metrics.TotalRuns)
	}
	if metrics.SuccessCount != 1 {
		t.Errorf("SuccessCount = %d, want 1", metrics.SuccessCount)
	}
	if metrics.ErrorCount != 0 {
		t.Errorf("ErrorCount = %d, want 0", metrics.ErrorCount)
	}
	if metrics.TotalTokensIn != 1000 {
		t.Errorf("TotalTokensIn = %d, want 1000", metrics.TotalTokensIn)
	}
	if metrics.TotalTokensOut != 500 {
		t.Errorf("TotalTokensOut = %d, want 500", metrics.TotalTokensOut)
	}
	if metrics.LastDuration != 5*time.Second {
		t.Errorf("LastDuration = %v, want 5s", metrics.LastDuration)
	}
}

func TestAdvisor_CompleteAnalysis_Error(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	testErr := errors.New("test error")
	a.StartAnalysis()
	a.CompleteAnalysis(2*time.Second, 500, 0, testErr)

	if a.State() != StateError {
		t.Errorf("after error completion, State() = %q, want %q", a.State(), StateError)
	}

	if a.LastError() != testErr {
		t.Errorf("LastError() = %v, want %v", a.LastError(), testErr)
	}

	metrics := a.Metrics()
	if metrics.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", metrics.ErrorCount)
	}
	if metrics.SuccessCount != 0 {
		t.Errorf("SuccessCount = %d, want 0", metrics.SuccessCount)
	}
}

func TestAdvisor_CompleteAnalysis_WithInsights(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	a.StartAnalysis()
	a.AddInsight(NewInsight("test", "Test Insight", "Description", SeverityInfo, CategoryGeneral))
	a.CompleteAnalysis(1*time.Second, 100, 50, nil)

	if a.State() != StateHasInsights {
		t.Errorf("after completion with insights, State() = %q, want %q", a.State(), StateHasInsights)
	}
}

func TestAdvisor_CancelAnalysis(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// Cancel when not running should fail
	if a.CancelAnalysis() {
		t.Error("CancelAnalysis() when not running returned true, want false")
	}

	a.StartAnalysis()

	// Cancel when running should succeed
	if !a.CancelAnalysis() {
		t.Error("CancelAnalysis() when running returned false, want true")
	}

	if a.State() != StateIdle {
		t.Errorf("after cancel, State() = %q, want %q", a.State(), StateIdle)
	}

	if a.IsRunning() {
		t.Error("after cancel, IsRunning() = true, want false")
	}

	metrics := a.Metrics()
	if metrics.CancelCount != 1 {
		t.Errorf("CancelCount = %d, want 1", metrics.CancelCount)
	}
}

func TestAdvisor_AddInsight(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// Adding nil should be safe
	a.AddInsight(nil)
	if len(a.Insights()) != 0 {
		t.Error("AddInsight(nil) should not add insight")
	}

	// Add valid insight
	insight := NewInsight("test", "Title", "Description", SeverityWarning, CategorySecurity)
	a.AddInsight(insight)

	insights := a.Insights()
	if len(insights) != 1 {
		t.Errorf("Insights() length = %d, want 1", len(insights))
	}

	if a.State() != StateHasInsights {
		t.Errorf("after adding insight, State() = %q, want %q", a.State(), StateHasInsights)
	}

	metrics := a.Metrics()
	if metrics.InsightCount != 1 {
		t.Errorf("InsightCount = %d, want 1", metrics.InsightCount)
	}
}

func TestAdvisor_UnreadInsightCount(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// Add multiple insights
	for i := 0; i < 3; i++ {
		a.AddInsight(NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral))
	}

	if a.UnreadInsightCount() != 3 {
		t.Errorf("UnreadInsightCount() = %d, want 3", a.UnreadInsightCount())
	}

	// Accept one
	insights := a.Insights()
	a.MarkInsightAccepted(insights[0].ID)

	if a.UnreadInsightCount() != 2 {
		t.Errorf("after accepting one, UnreadInsightCount() = %d, want 2", a.UnreadInsightCount())
	}
}

func TestAdvisor_ClearInsights(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	a.AddInsight(NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral))

	if a.State() != StateHasInsights {
		t.Errorf("before clear, State() = %q, want %q", a.State(), StateHasInsights)
	}

	a.ClearInsights()

	if len(a.Insights()) != 0 {
		t.Errorf("after ClearInsights(), Insights() length = %d, want 0", len(a.Insights()))
	}

	if a.State() != StateIdle {
		t.Errorf("after ClearInsights(), State() = %q, want %q", a.State(), StateIdle)
	}
}

func TestAdvisor_MarkInsightAccepted(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)
	a.AddInsight(insight)

	// Mark non-existent ID
	if a.MarkInsightAccepted("nonexistent") {
		t.Error("MarkInsightAccepted(nonexistent) returned true, want false")
	}

	// Mark valid ID
	if !a.MarkInsightAccepted(insight.ID) {
		t.Error("MarkInsightAccepted() returned false, want true")
	}

	metrics := a.Metrics()
	if metrics.AcceptedInsights != 1 {
		t.Errorf("AcceptedInsights = %d, want 1", metrics.AcceptedInsights)
	}

	// Verify insight state changed
	insights := a.Insights()
	if insights[0].State != InsightStateAccepted {
		t.Errorf("insight State = %q, want %q", insights[0].State, InsightStateAccepted)
	}
}

func TestAdvisor_MarkInsightRejected(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)
	a.AddInsight(insight)

	// Mark non-existent ID
	if a.MarkInsightRejected("nonexistent") {
		t.Error("MarkInsightRejected(nonexistent) returned true, want false")
	}

	// Mark valid ID
	if !a.MarkInsightRejected(insight.ID) {
		t.Error("MarkInsightRejected() returned false, want true")
	}

	metrics := a.Metrics()
	if metrics.RejectedInsights != 1 {
		t.Errorf("RejectedInsights = %d, want 1", metrics.RejectedInsights)
	}
}

func TestAdvisor_MatchesFile(t *testing.T) {
	a := New(Config{
		ID:            "test",
		Name:          "Test",
		SystemPrompt:  "test",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*.go", "*.ts"},
	}, 0, 0)

	if !a.MatchesFile("main.go") {
		t.Error("MatchesFile(main.go) = false, want true")
	}

	if a.MatchesFile("main.py") {
		t.Error("MatchesFile(main.py) = true, want false")
	}
}

func TestAdvisor_ShouldTriggerOnFileChange(t *testing.T) {
	// Manual trigger should not trigger on file change
	manual := New(Config{
		ID:            "manual",
		Name:          "Manual",
		SystemPrompt:  "test",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*.go"},
	}, 0, 0)

	if manual.ShouldTriggerOnFileChange("test.go") {
		t.Error("manual advisor ShouldTriggerOnFileChange() = true, want false")
	}

	// OnFileChange trigger should trigger for matching files
	onChange := New(Config{
		ID:            "onchange",
		Name:          "OnChange",
		SystemPrompt:  "test",
		Trigger:       TriggerOnFileChange,
		FocusPatterns: []string{"*.go"},
	}, 0, 0)

	if !onChange.ShouldTriggerOnFileChange("test.go") {
		t.Error("onChange advisor ShouldTriggerOnFileChange(test.go) = false, want true")
	}

	if onChange.ShouldTriggerOnFileChange("test.py") {
		t.Error("onChange advisor ShouldTriggerOnFileChange(test.py) = true, want false")
	}
}

func TestAdvisor_BackgroundInterval(t *testing.T) {
	// Non-background trigger should return 0
	manual := New(Config{
		ID:           "manual",
		Name:         "Manual",
		SystemPrompt: "test",
		Trigger:      TriggerManual,
	}, 0, 0)

	if manual.BackgroundInterval() != 0 {
		t.Errorf("manual advisor BackgroundInterval() = %v, want 0", manual.BackgroundInterval())
	}

	// Background trigger should return configured interval
	background := New(Config{
		ID:                     "background",
		Name:                   "Background",
		SystemPrompt:           "test",
		Trigger:                TriggerBackground,
		BackgroundIntervalSecs: 300,
	}, 0, 0)

	expected := 300 * time.Second
	if background.BackgroundInterval() != expected {
		t.Errorf("background advisor BackgroundInterval() = %v, want %v", background.BackgroundInterval(), expected)
	}
}

func TestAdvisor_SuccessRate(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// No runs should return 0
	if a.SuccessRate() != 0 {
		t.Errorf("initial SuccessRate() = %v, want 0", a.SuccessRate())
	}

	// Record some runs
	a.StartAnalysis()
	a.CompleteAnalysis(time.Second, 100, 50, nil) // Success

	a.StartAnalysis()
	a.CompleteAnalysis(time.Second, 100, 50, nil) // Success

	a.StartAnalysis()
	a.CompleteAnalysis(time.Second, 100, 0, errors.New("error")) // Failure

	a.StartAnalysis()
	a.CompleteAnalysis(time.Second, 100, 50, nil) // Success

	// 3/4 = 75%
	expected := 75.0
	if a.SuccessRate() != expected {
		t.Errorf("SuccessRate() = %v, want %v", a.SuccessRate(), expected)
	}
}

func TestAdvisor_AcceptanceRate(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// No insights should return 0
	if a.AcceptanceRate() != 0 {
		t.Errorf("initial AcceptanceRate() = %v, want 0", a.AcceptanceRate())
	}

	// Add and act on insights
	for i := 0; i < 4; i++ {
		a.AddInsight(NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral))
	}

	insights := a.Insights()
	a.MarkInsightAccepted(insights[0].ID)
	a.MarkInsightAccepted(insights[1].ID)
	a.MarkInsightAccepted(insights[2].ID)
	a.MarkInsightRejected(insights[3].ID)

	// 3/4 = 75%
	expected := 75.0
	if a.AcceptanceRate() != expected {
		t.Errorf("AcceptanceRate() = %v, want %v", a.AcceptanceRate(), expected)
	}
}

func TestAdvisorMetrics_ZeroValues(t *testing.T) {
	metrics := AdvisorMetrics{}

	if metrics.TotalRuns != 0 {
		t.Errorf("TotalRuns zero value = %d, want 0", metrics.TotalRuns)
	}
	if metrics.TotalTokensIn != 0 {
		t.Errorf("TotalTokensIn zero value = %d, want 0", metrics.TotalTokensIn)
	}
	if metrics.TotalTokensOut != 0 {
		t.Errorf("TotalTokensOut zero value = %d, want 0", metrics.TotalTokensOut)
	}
	if metrics.MinDuration != 0 {
		t.Errorf("MinDuration zero value = %v, want 0", metrics.MinDuration)
	}
	if metrics.MaxDuration != 0 {
		t.Errorf("MaxDuration zero value = %v, want 0", metrics.MaxDuration)
	}
	if metrics.AvgDuration != 0 {
		t.Errorf("AvgDuration zero value = %v, want 0", metrics.AvgDuration)
	}
	if metrics.LastDuration != 0 {
		t.Errorf("LastDuration zero value = %v, want 0", metrics.LastDuration)
	}
}

func TestAdvisor_Metrics_MultipleRuns(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	// Record multiple runs with different durations
	durations := []time.Duration{5 * time.Second, 3 * time.Second, 7 * time.Second, 4 * time.Second}
	for _, d := range durations {
		a.StartAnalysis()
		a.CompleteAnalysis(d, 100, 50, nil)
	}

	metrics := a.Metrics()

	if metrics.TotalRuns != 4 {
		t.Errorf("TotalRuns = %d, want 4", metrics.TotalRuns)
	}

	// Min should be 3s
	if metrics.MinDuration != 3*time.Second {
		t.Errorf("MinDuration = %v, want 3s", metrics.MinDuration)
	}

	// Max should be 7s
	if metrics.MaxDuration != 7*time.Second {
		t.Errorf("MaxDuration = %v, want 7s", metrics.MaxDuration)
	}

	// Last should be 4s
	if metrics.LastDuration != 4*time.Second {
		t.Errorf("LastDuration = %v, want 4s", metrics.LastDuration)
	}

	// Avg should be (5+3+7+4)/4 = 4.75s
	expectedAvg := (5 + 3 + 7 + 4) * time.Second / 4
	if metrics.AvgDuration != expectedAvg {
		t.Errorf("AvgDuration = %v, want %v", metrics.AvgDuration, expectedAvg)
	}

	// Total tokens
	if metrics.TotalTokensIn != 400 {
		t.Errorf("TotalTokensIn = %d, want 400", metrics.TotalTokensIn)
	}
	if metrics.TotalTokensOut != 200 {
		t.Errorf("TotalTokensOut = %d, want 200", metrics.TotalTokensOut)
	}
}

func TestAdvisorState_Constants(t *testing.T) {
	// Verify state constants are distinct
	states := map[AdvisorState]bool{
		StateIdle:        true,
		StateThinking:    true,
		StateHasInsights: true,
		StateError:       true,
	}

	if len(states) != 4 {
		t.Error("AdvisorState constants should be distinct")
	}
}

func TestAdvisor_Concurrency(t *testing.T) {
	a := New(Config{ID: "test", Name: "Test", SystemPrompt: "test", Trigger: TriggerManual}, 0, 0)

	var wg sync.WaitGroup
	done := make(chan struct{})
	insightsDone := make(chan struct{})

	// Multiple goroutines adding insights (these finish first)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				a.AddInsight(NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral))
			}
		}()
	}

	// Wait for insight goroutines to finish
	go func() {
		wg.Wait()
		close(insightsDone)
	}()

	// Multiple goroutines reading advisor state - each does 100 iterations
	var readersWg sync.WaitGroup
	for i := 0; i < 10; i++ {
		readersWg.Add(1)
		go func(idx int) {
			defer readersWg.Done()
			for iterations := 0; iterations < 100; iterations++ {
				select {
				case <-done:
					return
				default:
					_ = a.ID()
					_ = a.Name()
					_ = a.State()
					_ = a.Metrics()
					_ = a.Insights()
					a.SetPosition(float64(idx), float64(idx))
				}
			}
		}(i)
	}

	// Wait for insights to be added
	<-insightsDone

	// Signal readers to finish if they haven't already
	close(done)
	readersWg.Wait()

	// If we get here without race conditions, test passes
	// Run with -race to detect races
}
