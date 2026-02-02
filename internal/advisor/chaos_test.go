package advisor

import (
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"
)

// TestChaos_ConcurrentInsightModification verifies that Insight is thread-safe
// when accessed from multiple goroutines concurrently.
func TestChaos_ConcurrentInsightModification(t *testing.T) {
	insight := NewInsight("test", "Title", "Desc", SeverityInfo, CategoryGeneral)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			insight.Accept()
		}()
		go func() {
			defer wg.Done()
			insight.Reject()
		}()
		go func() {
			defer wg.Done()
			_ = insight.IsPending()
		}()
	}
	wg.Wait()

	// If we get here without race detector complaints, the mutex is working
	if !insight.IsResolved() {
		t.Errorf("Expected insight to be resolved after concurrent modifications")
	}
}

// TestChaos_ConcurrentInsightReadWrite tests concurrent reads and writes
func TestChaos_ConcurrentInsightReadWrite(t *testing.T) {
	insight := NewInsight("test", "Title", "Desc", SeverityInfo, CategoryGeneral)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(6)
		// Writers
		go func() {
			defer wg.Done()
			insight.WithLocation("file.go", 10, 5)
		}()
		go func() {
			defer wg.Done()
			insight.WithSuggestion("fix", "before", "after")
		}()
		go func() {
			defer wg.Done()
			insight.Dismiss()
		}()
		// Readers
		go func() {
			defer wg.Done()
			_ = insight.HasLocation()
		}()
		go func() {
			defer wg.Done()
			_ = insight.HasSuggestion()
		}()
		go func() {
			defer wg.Done()
			_ = insight.IsResolved()
		}()
	}
	wg.Wait()

	// Test passes if no race conditions detected
}

// panicListener is a PoolListener that panics on every call
type panicListener struct{}

func (p panicListener) OnAdvisorAdded(*Advisor) { panic("boom on add") }
func (p panicListener) OnAdvisorRemoved(string)  { panic("boom on remove") }

// TestChaos_ListenerPanic verifies that a panicking listener doesn't crash the pool
func TestChaos_ListenerPanic(t *testing.T) {
	pool := NewPool()
	pool.AddListener(panicListener{})

	advisor := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)

	// Should NOT panic
	err := pool.Add(advisor)
	if err != nil {
		t.Errorf("Add failed despite panic recovery: %v", err)
	}
	if pool.Count() != 1 {
		t.Errorf("Advisor not added despite listener panic")
	}

	// Remove should also not panic
	err = pool.Remove("test")
	if err != nil {
		t.Errorf("Remove failed despite panic recovery: %v", err)
	}
	if pool.Count() != 0 {
		t.Errorf("Advisor not removed despite listener panic")
	}
}

// TestChaos_ListenerPanicOnClear verifies Clear handles panicking listeners
func TestChaos_ListenerPanicOnClear(t *testing.T) {
	pool := NewPool()
	pool.AddListener(panicListener{})

	// Add some advisors
	for i := 0; i < 5; i++ {
		pool.Add(New(Config{
			ID:           string(rune('a' + i)),
			Name:         "Test",
			SystemPrompt: "prompt",
			Trigger:      TriggerManual,
		}, 0, 0))
	}

	// Clear should not panic
	pool.Clear()

	if pool.Count() != 0 {
		t.Errorf("Clear failed, count = %d, want 0", pool.Count())
	}
}

// TestChaos_RapidStateTransitions tests rapid state transitions on Advisor
func TestChaos_RapidStateTransitions(t *testing.T) {
	a := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			a.StartAnalysis()
		}()
		go func() {
			defer wg.Done()
			a.CompleteAnalysis(time.Millisecond, 100, 50, nil)
		}()
		go func() {
			defer wg.Done()
			a.CancelAnalysis()
		}()
		go func() {
			defer wg.Done()
			a.AddInsight(NewInsight(a.ID(), "Test", "Desc", SeverityInfo, CategoryGeneral))
		}()
	}
	wg.Wait()

	// Verify state is consistent (one of the valid states)
	state := a.State()
	validStates := map[AdvisorState]bool{
		StateIdle:        true,
		StateThinking:    true,
		StateHasInsights: true,
		StateError:       true,
	}
	if !validStates[state] {
		t.Errorf("Invalid state after rapid transitions: %v", state)
	}
}

// TestChaos_ConcurrentPoolOperations tests concurrent pool operations
func TestChaos_ConcurrentPoolOperations(t *testing.T) {
	pool := NewPool()

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Spawn add/remove operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			advisor := New(Config{
				ID:           string(rune('a' + id)),
				Name:         "Test",
				SystemPrompt: "prompt",
				Trigger:      TriggerManual,
			}, 0, 0)

			for j := 0; j < 10; j++ {
				select {
				case <-done:
					return
				default:
					pool.Add(advisor)
					pool.Remove(advisor.ID())
				}
			}
		}(i)
	}

	// Spawn readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				select {
				case <-done:
					return
				default:
					_ = pool.Count()
					_ = pool.GetAll()
					_ = pool.GetAllInsights()
				}
			}
		}()
	}

	// Wait a bit then signal done
	time.Sleep(50 * time.Millisecond)
	close(done)
	wg.Wait()

	// If no deadlocks or races, test passes
}

// randomError returns nil or an error randomly
func randomError() error {
	if rand.Float32() < 0.2 {
		return &chaosTestError{"random error"}
	}
	return nil
}

type chaosTestError struct{ msg string }

func (e *chaosTestError) Error() string { return e.msg }

// TestProperty_MetricsConsistent verifies that metrics remain internally consistent
// after random sequences of operations
func TestProperty_MetricsConsistent(t *testing.T) {
	a := New(Config{
		ID:           "test",
		Name:         "Test",
		SystemPrompt: "prompt",
		Trigger:      TriggerManual,
	}, 0, 0)

	// Run random sequence of operations
	for i := 0; i < 100; i++ {
		a.StartAnalysis()
		if rand.Float32() < 0.8 {
			a.CompleteAnalysis(
				time.Duration(rand.Intn(1000))*time.Millisecond,
				int64(rand.Intn(1000)),
				int64(rand.Intn(500)),
				randomError(),
			)
		} else {
			a.CancelAnalysis()
		}
	}

	m := a.Metrics()

	// Invariant: TotalRuns == SuccessCount + ErrorCount (cancels don't count as runs)
	if m.TotalRuns != m.SuccessCount+m.ErrorCount {
		t.Errorf("Metrics inconsistent: TotalRuns(%d) != SuccessCount(%d) + ErrorCount(%d)",
			m.TotalRuns, m.SuccessCount, m.ErrorCount)
	}

	// Invariant: MinDuration <= MaxDuration (when we have runs)
	if m.TotalRuns > 0 && m.MinDuration > m.MaxDuration {
		t.Errorf("Duration invariant violated: min=%v > max=%v",
			m.MinDuration, m.MaxDuration)
	}
}

// extractIDs extracts IDs from insights and sorts them
func extractIDs(insights []*Insight) []string {
	ids := make([]string, len(insights))
	for i, insight := range insights {
		ids[i] = insight.ID
	}
	sort.Strings(ids)
	return ids
}

// generateRandomInsights generates n random insights
func generateRandomInsights(n int) []*Insight {
	severities := []InsightSeverity{SeverityInfo, SeverityWarning, SeverityCritical}
	categories := []InsightCategory{CategorySecurity, CategoryPerformance, CategoryRefactoring, CategoryTesting, CategoryGeneral}

	insights := make([]*Insight, n)
	for i := 0; i < n; i++ {
		insights[i] = NewInsight(
			"test",
			"Title",
			"Desc",
			severities[rand.Intn(len(severities))],
			categories[rand.Intn(len(categories))],
		)
	}
	return insights
}

// TestProperty_FilterCommutativity verifies that filter order doesn't matter
func TestProperty_FilterCommutativity(t *testing.T) {
	insights := generateRandomInsights(100)

	// Apply filters in different orders
	r1 := FilterInsights(insights).BySeverity(SeverityCritical).ByCategory(CategorySecurity).Results()
	r2 := FilterInsights(insights).ByCategory(CategorySecurity).BySeverity(SeverityCritical).Results()

	if len(r1) != len(r2) {
		t.Errorf("Filter order matters: %d vs %d results", len(r1), len(r2))
	}

	// Check same elements (order may differ)
	ids1 := extractIDs(r1)
	ids2 := extractIDs(r2)

	if len(ids1) != len(ids2) {
		t.Errorf("Different result counts: %d vs %d", len(ids1), len(ids2))
		return
	}

	for i := range ids1 {
		if ids1[i] != ids2[i] {
			t.Errorf("Different results at position %d: %s vs %s", i, ids1[i], ids2[i])
		}
	}
}

// TestProperty_FilterIdempotent verifies that applying the same filter twice gives same result
func TestProperty_FilterIdempotent(t *testing.T) {
	insights := generateRandomInsights(50)

	r1 := FilterInsights(insights).BySeverity(SeverityWarning).Results()
	r2 := FilterInsights(r1).BySeverity(SeverityWarning).Results()

	if len(r1) != len(r2) {
		t.Errorf("Filter not idempotent: %d vs %d results", len(r1), len(r2))
	}
}
