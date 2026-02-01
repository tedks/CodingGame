package advisor

import (
	"sync"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool := NewPool()

	if pool == nil {
		t.Fatal("NewPool() returned nil")
	}
	if pool.Count() != 0 {
		t.Errorf("initial Count() = %d, want 0", pool.Count())
	}
	if pool.IsRunning() {
		t.Error("initial IsRunning() = true, want false")
	}
}

func TestPool_Add(t *testing.T) {
	pool := NewPool()
	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)

	err := pool.Add(advisor)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if pool.Count() != 1 {
		t.Errorf("after Add(), Count() = %d, want 1", pool.Count())
	}
}

func TestPool_Add_Nil(t *testing.T) {
	pool := NewPool()

	err := pool.Add(nil)
	if err == nil {
		t.Error("Add(nil) expected error")
	}
}

func TestPool_Add_Duplicate(t *testing.T) {
	pool := NewPool()
	advisor1 := New(Config{ID: "test", Name: "Test 1", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	advisor2 := New(Config{ID: "test", Name: "Test 2", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)

	err := pool.Add(advisor1)
	if err != nil {
		t.Fatalf("first Add() error = %v", err)
	}

	err = pool.Add(advisor2)
	if err == nil {
		t.Error("second Add() with same ID expected error")
	}
}

func TestPool_Remove(t *testing.T) {
	pool := NewPool()
	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	pool.Add(advisor)

	err := pool.Remove("test")
	if err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	if pool.Count() != 0 {
		t.Errorf("after Remove(), Count() = %d, want 0", pool.Count())
	}
}

func TestPool_Remove_NotFound(t *testing.T) {
	pool := NewPool()

	err := pool.Remove("nonexistent")
	if err == nil {
		t.Error("Remove(nonexistent) expected error")
	}
}

func TestPool_Get(t *testing.T) {
	pool := NewPool()
	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	pool.Add(advisor)

	got := pool.Get("test")
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.ID() != "test" {
		t.Errorf("Get().ID() = %q, want test", got.ID())
	}

	// Get non-existent
	notFound := pool.Get("nonexistent")
	if notFound != nil {
		t.Error("Get(nonexistent) should return nil")
	}
}

func TestPool_GetAll(t *testing.T) {
	pool := NewPool()

	// Empty pool
	all := pool.GetAll()
	if len(all) != 0 {
		t.Errorf("empty pool GetAll() length = %d, want 0", len(all))
	}

	// Add some advisors
	for i := 0; i < 3; i++ {
		advisor := New(Config{
			ID:           string(rune('a' + i)),
			Name:         "Test",
			SystemPrompt: "prompt",
			Trigger:      TriggerManual,
		}, 0, 0)
		pool.Add(advisor)
	}

	all = pool.GetAll()
	if len(all) != 3 {
		t.Errorf("GetAll() length = %d, want 3", len(all))
	}
}

func TestPool_LoadFromConfig(t *testing.T) {
	pool := NewPool()

	configs := []Config{
		{ID: "a", Name: "Advisor A", SystemPrompt: "prompt", Trigger: TriggerManual},
		{ID: "b", Name: "Advisor B", SystemPrompt: "prompt", Trigger: TriggerOnFileChange},
	}

	err := pool.LoadFromConfig(configs)
	if err != nil {
		t.Fatalf("LoadFromConfig() error = %v", err)
	}

	if pool.Count() != 2 {
		t.Errorf("Count() = %d, want 2", pool.Count())
	}

	advisorA := pool.Get("a")
	if advisorA == nil {
		t.Error("advisor 'a' not found")
	}
}

func TestPool_LoadFromConfig_InvalidConfig(t *testing.T) {
	pool := NewPool()

	configs := []Config{
		{ID: "valid", Name: "Valid", SystemPrompt: "prompt", Trigger: TriggerManual},
		{ID: "", Name: "Invalid", SystemPrompt: "prompt", Trigger: TriggerManual}, // Invalid - empty ID
	}

	err := pool.LoadFromConfig(configs)
	if err == nil {
		t.Error("LoadFromConfig() with invalid config expected error")
	}
}

func TestPool_StartStop(t *testing.T) {
	pool := NewPool()

	err := pool.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !pool.IsRunning() {
		t.Error("after Start(), IsRunning() = false, want true")
	}

	// Double start should fail
	err = pool.Start()
	if err == nil {
		t.Error("double Start() expected error")
	}

	err = pool.Stop()
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if pool.IsRunning() {
		t.Error("after Stop(), IsRunning() = true, want false")
	}

	// Double stop should be safe
	err = pool.Stop()
	if err != nil {
		t.Errorf("double Stop() error = %v", err)
	}
}

func TestPool_TriggerOnFileChange(t *testing.T) {
	pool := NewPool()

	// Add advisors with different triggers
	manual := New(Config{
		ID:            "manual",
		Name:          "Manual",
		SystemPrompt:  "prompt",
		Trigger:       TriggerManual,
		FocusPatterns: []string{"*.go"},
	}, 0, 0)

	onChange := New(Config{
		ID:            "onchange",
		Name:          "OnChange",
		SystemPrompt:  "prompt",
		Trigger:       TriggerOnFileChange,
		FocusPatterns: []string{"*.go"},
	}, 0, 0)

	pool.Add(manual)
	pool.Add(onChange)

	triggered := pool.TriggerOnFileChange("test.go")

	if len(triggered) != 1 {
		t.Errorf("TriggerOnFileChange() returned %d advisors, want 1", len(triggered))
	}
	if len(triggered) > 0 && triggered[0].ID() != "onchange" {
		t.Errorf("triggered advisor ID = %q, want onchange", triggered[0].ID())
	}

	// Non-matching file
	triggered = pool.TriggerOnFileChange("test.py")
	if len(triggered) != 0 {
		t.Errorf("TriggerOnFileChange(test.py) returned %d advisors, want 0", len(triggered))
	}
}

func TestPool_GetAdvisorsByState(t *testing.T) {
	pool := NewPool()

	idle := New(Config{ID: "idle", Name: "Idle", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	thinking := New(Config{ID: "thinking", Name: "Thinking", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	thinking.StartAnalysis()

	pool.Add(idle)
	pool.Add(thinking)

	idleAdvisors := pool.GetAdvisorsByState(StateIdle)
	if len(idleAdvisors) != 1 {
		t.Errorf("GetAdvisorsByState(idle) count = %d, want 1", len(idleAdvisors))
	}

	thinkingAdvisors := pool.GetAdvisorsByState(StateThinking)
	if len(thinkingAdvisors) != 1 {
		t.Errorf("GetAdvisorsByState(thinking) count = %d, want 1", len(thinkingAdvisors))
	}
}

func TestPool_GetAdvisorsByTrigger(t *testing.T) {
	pool := NewPool()

	manual1 := New(Config{ID: "m1", Name: "Manual1", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	manual2 := New(Config{ID: "m2", Name: "Manual2", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	onChange := New(Config{ID: "oc", Name: "OnChange", SystemPrompt: "prompt", Trigger: TriggerOnFileChange}, 0, 0)

	pool.Add(manual1)
	pool.Add(manual2)
	pool.Add(onChange)

	manualAdvisors := pool.GetAdvisorsByTrigger(TriggerManual)
	if len(manualAdvisors) != 2 {
		t.Errorf("GetAdvisorsByTrigger(manual) count = %d, want 2", len(manualAdvisors))
	}

	onChangeAdvisors := pool.GetAdvisorsByTrigger(TriggerOnFileChange)
	if len(onChangeAdvisors) != 1 {
		t.Errorf("GetAdvisorsByTrigger(on_file_change) count = %d, want 1", len(onChangeAdvisors))
	}
}

func TestPool_GetAllInsights(t *testing.T) {
	pool := NewPool()

	a1 := New(Config{ID: "a1", Name: "A1", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	a2 := New(Config{ID: "a2", Name: "A2", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)

	a1.AddInsight(NewInsight("a1", "Insight 1", "desc", SeverityInfo, CategoryGeneral))
	a1.AddInsight(NewInsight("a1", "Insight 2", "desc", SeverityInfo, CategoryGeneral))
	a2.AddInsight(NewInsight("a2", "Insight 3", "desc", SeverityInfo, CategoryGeneral))

	pool.Add(a1)
	pool.Add(a2)

	all := pool.GetAllInsights()
	if len(all) != 3 {
		t.Errorf("GetAllInsights() count = %d, want 3", len(all))
	}
}

func TestPool_GetPendingInsights(t *testing.T) {
	pool := NewPool()

	a := New(Config{ID: "a", Name: "A", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	insight1 := NewInsight("a", "Pending", "desc", SeverityInfo, CategoryGeneral)
	insight2 := NewInsight("a", "Accepted", "desc", SeverityInfo, CategoryGeneral)
	insight2.Accept()

	a.AddInsight(insight1)
	a.AddInsight(insight2)
	pool.Add(a)

	pending := pool.GetPendingInsights()
	if len(pending) != 1 {
		t.Errorf("GetPendingInsights() count = %d, want 1", len(pending))
	}
}

func TestPool_GetCriticalInsights(t *testing.T) {
	pool := NewPool()

	a := New(Config{ID: "a", Name: "A", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	a.AddInsight(NewInsight("a", "Info", "desc", SeverityInfo, CategoryGeneral))
	a.AddInsight(NewInsight("a", "Critical", "desc", SeverityCritical, CategoryGeneral))

	pool.Add(a)

	critical := pool.GetCriticalInsights()
	if len(critical) != 1 {
		t.Errorf("GetCriticalInsights() count = %d, want 1", len(critical))
	}
}

func TestPool_TotalInsightCount(t *testing.T) {
	pool := NewPool()

	a1 := New(Config{ID: "a1", Name: "A1", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	a2 := New(Config{ID: "a2", Name: "A2", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)

	a1.AddInsight(NewInsight("a1", "1", "desc", SeverityInfo, CategoryGeneral))
	a1.AddInsight(NewInsight("a1", "2", "desc", SeverityInfo, CategoryGeneral))
	a2.AddInsight(NewInsight("a2", "3", "desc", SeverityInfo, CategoryGeneral))

	pool.Add(a1)
	pool.Add(a2)

	if pool.TotalInsightCount() != 3 {
		t.Errorf("TotalInsightCount() = %d, want 3", pool.TotalInsightCount())
	}
}

func TestPool_PendingInsightCount(t *testing.T) {
	pool := NewPool()

	a := New(Config{ID: "a", Name: "A", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	insight1 := NewInsight("a", "1", "desc", SeverityInfo, CategoryGeneral)
	insight2 := NewInsight("a", "2", "desc", SeverityInfo, CategoryGeneral)

	a.AddInsight(insight1)
	a.AddInsight(insight2)
	a.MarkInsightAccepted(insight1.ID)

	pool.Add(a)

	if pool.PendingInsightCount() != 1 {
		t.Errorf("PendingInsightCount() = %d, want 1", pool.PendingInsightCount())
	}
}

func TestPool_AggregateMetrics(t *testing.T) {
	pool := NewPool()

	a1 := New(Config{ID: "a1", Name: "A1", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	a2 := New(Config{ID: "a2", Name: "A2", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)

	// Record some runs
	a1.StartAnalysis()
	a1.CompleteAnalysis(time.Second, 100, 50, nil)
	a2.StartAnalysis()
	a2.CompleteAnalysis(time.Second, 200, 100, nil)

	pool.Add(a1)
	pool.Add(a2)

	metrics := pool.AggregateMetrics()

	if metrics.AdvisorCount != 2 {
		t.Errorf("AdvisorCount = %d, want 2", metrics.AdvisorCount)
	}
	if metrics.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", metrics.TotalRuns)
	}
	if metrics.TotalTokensIn != 300 {
		t.Errorf("TotalTokensIn = %d, want 300", metrics.TotalTokensIn)
	}
	if metrics.TotalTokensOut != 150 {
		t.Errorf("TotalTokensOut = %d, want 150", metrics.TotalTokensOut)
	}
}

func TestPool_Clear(t *testing.T) {
	pool := NewPool()

	pool.Add(New(Config{ID: "a", Name: "A", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0))
	pool.Add(New(Config{ID: "b", Name: "B", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0))

	if pool.Count() != 2 {
		t.Fatalf("before Clear(), Count() = %d, want 2", pool.Count())
	}

	pool.Clear()

	if pool.Count() != 0 {
		t.Errorf("after Clear(), Count() = %d, want 0", pool.Count())
	}
}

// Mock listener for testing
type mockListener struct {
	mu      sync.Mutex
	added   []*Advisor
	removed []string
	// Channels for synchronization in tests
	addedCh   chan *Advisor
	removedCh chan string
}

func newMockListener() *mockListener {
	return &mockListener{
		addedCh:   make(chan *Advisor, 10),
		removedCh: make(chan string, 10),
	}
}

func (m *mockListener) OnAdvisorAdded(advisor *Advisor) {
	m.mu.Lock()
	m.added = append(m.added, advisor)
	m.mu.Unlock()
	// Signal via channel for synchronization
	select {
	case m.addedCh <- advisor:
	default:
	}
}

func (m *mockListener) OnAdvisorRemoved(advisorID string) {
	m.mu.Lock()
	m.removed = append(m.removed, advisorID)
	m.mu.Unlock()
	// Signal via channel for synchronization
	select {
	case m.removedCh <- advisorID:
	default:
	}
}

func TestPool_Listener_OnAdvisorAdded(t *testing.T) {
	pool := NewPool()
	listener := newMockListener()
	pool.AddListener(listener)

	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	pool.Add(advisor)

	// Wait for async notification via channel
	select {
	case <-listener.addedCh:
		// Notification received
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for OnAdvisorAdded notification")
	}

	listener.mu.Lock()
	addedCount := len(listener.added)
	listener.mu.Unlock()

	if addedCount != 1 {
		t.Errorf("listener.added count = %d, want 1", addedCount)
	}
}

func TestPool_Listener_OnAdvisorRemoved(t *testing.T) {
	pool := NewPool()
	listener := newMockListener()
	pool.AddListener(listener)

	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	pool.Add(advisor)

	// Wait for add notification first
	select {
	case <-listener.addedCh:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for OnAdvisorAdded notification")
	}

	pool.Remove("test")

	// Wait for remove notification via channel
	select {
	case <-listener.removedCh:
		// Notification received
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for OnAdvisorRemoved notification")
	}

	listener.mu.Lock()
	removedCount := len(listener.removed)
	listener.mu.Unlock()

	if removedCount != 1 {
		t.Errorf("listener.removed count = %d, want 1", removedCount)
	}
}

func TestPool_RemoveListener(t *testing.T) {
	pool := NewPool()
	listener := newMockListener()
	pool.AddListener(listener)
	pool.RemoveListener(listener)

	advisor := New(Config{ID: "test", Name: "Test", SystemPrompt: "prompt", Trigger: TriggerManual}, 0, 0)
	pool.Add(advisor)

	// Verify no notification arrives (use short timeout since we expect nothing)
	select {
	case <-listener.addedCh:
		t.Error("received notification after RemoveListener, expected none")
	case <-time.After(50 * time.Millisecond):
		// Expected: no notification
	}

	listener.mu.Lock()
	addedCount := len(listener.added)
	listener.mu.Unlock()

	if addedCount != 0 {
		t.Errorf("after RemoveListener, listener.added count = %d, want 0", addedCount)
	}
}

func TestPool_Concurrency(t *testing.T) {
	pool := NewPool()
	var wg sync.WaitGroup
	done := make(chan struct{})
	addsDone := make(chan struct{})

	// Add advisors concurrently
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
			pool.Add(advisor)
		}(i)
	}

	// Wait for all adds to complete, then signal readers to stop
	go func() {
		wg.Wait()
		close(addsDone)
	}()

	// Read concurrently - each reader does 100 iterations then exits
	var readersWg sync.WaitGroup
	for i := 0; i < 5; i++ {
		readersWg.Add(1)
		go func() {
			defer readersWg.Done()
			iterations := 0
			for iterations < 100 {
				select {
				case <-done:
					return
				default:
					_ = pool.Count()
					_ = pool.GetAll()
					_ = pool.GetAllInsights()
					_ = pool.AggregateMetrics()
					iterations++
				}
			}
		}()
	}

	// Wait for adds to complete
	<-addsDone

	// Signal readers to finish if they haven't already
	close(done)
	readersWg.Wait()

	// If no race conditions, test passes
}
