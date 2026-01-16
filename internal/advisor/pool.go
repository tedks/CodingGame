package advisor

import (
	"fmt"
	"sync"
	"time"
)

// Pool manages a collection of advisors
type Pool struct {
	mu sync.RWMutex

	advisors  map[string]*Advisor
	listeners []PoolListener
	running   bool

	// Background processing
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// PoolListener receives notifications about pool events
type PoolListener interface {
	// OnAdvisorAdded is called when an advisor is added to the pool
	OnAdvisorAdded(advisor *Advisor)
	// OnAdvisorRemoved is called when an advisor is removed from the pool
	OnAdvisorRemoved(advisorID string)
	// OnInsightGenerated is called when an advisor produces an insight
	OnInsightGenerated(insight *Insight)
	// OnAdvisorStateChanged is called when an advisor changes state
	OnAdvisorStateChanged(advisor *Advisor, oldState, newState AdvisorState)
}

// NewPool creates a new advisor pool
func NewPool() *Pool {
	return &Pool{
		advisors: make(map[string]*Advisor),
		stopCh:   make(chan struct{}),
	}
}

// AddListener registers a listener for pool events
func (p *Pool) AddListener(listener PoolListener) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.listeners = append(p.listeners, listener)
}

// RemoveListener unregisters a listener
func (p *Pool) RemoveListener(listener PoolListener) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, l := range p.listeners {
		if l == listener {
			p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
			return
		}
	}
}

// Add adds an advisor to the pool
//
// Returns error if an advisor with the same ID already exists
func (p *Pool) Add(advisor *Advisor) error {
	if advisor == nil {
		return fmt.Errorf("advisor is nil")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	id := advisor.ID()
	if _, exists := p.advisors[id]; exists {
		return fmt.Errorf("advisor %q already exists", id)
	}

	p.advisors[id] = advisor

	// Notify listeners
	for _, listener := range p.listeners {
		go listener.OnAdvisorAdded(advisor)
	}

	return nil
}

// Remove removes an advisor from the pool
//
// Returns error if the advisor doesn't exist
func (p *Pool) Remove(id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.advisors[id]; !exists {
		return fmt.Errorf("advisor %q not found", id)
	}

	delete(p.advisors, id)

	// Notify listeners
	for _, listener := range p.listeners {
		go listener.OnAdvisorRemoved(id)
	}

	return nil
}

// Get retrieves an advisor by ID
func (p *Pool) Get(id string) *Advisor {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.advisors[id]
}

// GetAll returns a slice of all advisors
func (p *Pool) GetAll() []*Advisor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	advisors := make([]*Advisor, 0, len(p.advisors))
	for _, advisor := range p.advisors {
		advisors = append(advisors, advisor)
	}
	return advisors
}

// Count returns the number of advisors in the pool
func (p *Pool) Count() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.advisors)
}

// LoadFromConfig loads advisors from configuration
func (p *Pool) LoadFromConfig(configs []Config) error {
	for i, cfg := range configs {
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("config %d: %w", i, err)
		}

		// Create advisor at a default position (can be repositioned later)
		advisor := New(cfg, float64(i*50), 0)
		if err := p.Add(advisor); err != nil {
			return fmt.Errorf("config %d: %w", i, err)
		}
	}
	return nil
}

// Start begins background processing for background-triggered advisors
func (p *Pool) Start() error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return fmt.Errorf("pool already running")
	}
	p.running = true
	p.stopCh = make(chan struct{})
	p.mu.Unlock()

	// Start background workers for each background-triggered advisor
	p.mu.RLock()
	for _, advisor := range p.advisors {
		if advisor.Trigger() == TriggerBackground {
			p.wg.Add(1)
			go p.runBackgroundAdvisor(advisor)
		}
	}
	p.mu.RUnlock()

	return nil
}

// Stop stops all background processing
func (p *Pool) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	close(p.stopCh)
	p.mu.Unlock()

	p.wg.Wait()
	return nil
}

// IsRunning returns whether the pool is running
func (p *Pool) IsRunning() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.running
}

// runBackgroundAdvisor runs a background advisor on its configured interval
func (p *Pool) runBackgroundAdvisor(advisor *Advisor) {
	defer p.wg.Done()

	interval := advisor.BackgroundInterval()
	if interval == 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			// This would trigger the actual analysis
			// For now, we just mark that we would run
			// The actual execution would involve calling Claude
		}
	}
}

// TriggerOnFileChange triggers advisors that should run when a file changes
//
// Returns the advisors that were triggered
func (p *Pool) TriggerOnFileChange(filePath string) []*Advisor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var triggered []*Advisor
	for _, advisor := range p.advisors {
		if advisor.ShouldTriggerOnFileChange(filePath) {
			triggered = append(triggered, advisor)
		}
	}
	return triggered
}

// GetAdvisorsByState returns all advisors in a given state
func (p *Pool) GetAdvisorsByState(state AdvisorState) []*Advisor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Advisor
	for _, advisor := range p.advisors {
		if advisor.State() == state {
			result = append(result, advisor)
		}
	}
	return result
}

// GetAdvisorsByTrigger returns all advisors with a given trigger mode
func (p *Pool) GetAdvisorsByTrigger(trigger TriggerMode) []*Advisor {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Advisor
	for _, advisor := range p.advisors {
		if advisor.Trigger() == trigger {
			result = append(result, advisor)
		}
	}
	return result
}

// GetAllInsights returns all insights from all advisors
func (p *Pool) GetAllInsights() []*Insight {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var all []*Insight
	for _, advisor := range p.advisors {
		all = append(all, advisor.Insights()...)
	}
	return all
}

// GetPendingInsights returns all pending insights from all advisors
func (p *Pool) GetPendingInsights() []*Insight {
	insights := p.GetAllInsights()
	return FilterInsights(insights).Pending().Results()
}

// GetCriticalInsights returns all critical insights from all advisors
func (p *Pool) GetCriticalInsights() []*Insight {
	insights := p.GetAllInsights()
	return FilterInsights(insights).Critical().Results()
}

// TotalInsightCount returns the total number of insights across all advisors
func (p *Pool) TotalInsightCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, advisor := range p.advisors {
		count += len(advisor.Insights())
	}
	return count
}

// PendingInsightCount returns the number of pending insights across all advisors
func (p *Pool) PendingInsightCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, advisor := range p.advisors {
		count += advisor.UnreadInsightCount()
	}
	return count
}

// AggregateMetrics returns aggregated metrics across all advisors
func (p *Pool) AggregateMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var metrics PoolMetrics
	metrics.AdvisorCount = len(p.advisors)

	for _, advisor := range p.advisors {
		m := advisor.Metrics()
		metrics.TotalRuns += m.TotalRuns
		metrics.TotalTokensIn += m.TotalTokensIn
		metrics.TotalTokensOut += m.TotalTokensOut
		metrics.TotalInsights += m.InsightCount
		metrics.AcceptedInsights += m.AcceptedInsights
		metrics.RejectedInsights += m.RejectedInsights
	}

	return metrics
}

// PoolMetrics contains aggregated metrics for all advisors
type PoolMetrics struct {
	AdvisorCount     int
	TotalRuns        int
	TotalTokensIn    int64
	TotalTokensOut   int64
	TotalInsights    int
	AcceptedInsights int
	RejectedInsights int
}

// notifyInsight notifies all listeners about a new insight
func (p *Pool) notifyInsight(insight *Insight) {
	p.mu.RLock()
	listeners := make([]PoolListener, len(p.listeners))
	copy(listeners, p.listeners)
	p.mu.RUnlock()

	for _, listener := range listeners {
		go listener.OnInsightGenerated(insight)
	}
}

// notifyStateChange notifies all listeners about an advisor state change
func (p *Pool) notifyStateChange(advisor *Advisor, oldState, newState AdvisorState) {
	p.mu.RLock()
	listeners := make([]PoolListener, len(p.listeners))
	copy(listeners, p.listeners)
	p.mu.RUnlock()

	for _, listener := range listeners {
		go listener.OnAdvisorStateChanged(advisor, oldState, newState)
	}
}

// Clear removes all advisors from the pool
func (p *Pool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id := range p.advisors {
		delete(p.advisors, id)
		for _, listener := range p.listeners {
			go listener.OnAdvisorRemoved(id)
		}
	}
}
