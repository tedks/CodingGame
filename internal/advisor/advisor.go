package advisor

import (
	"sync"
	"time"
)

// AdvisorState represents the current state of an advisor
type AdvisorState string

const (
	// StateIdle means the advisor is not currently analyzing
	StateIdle AdvisorState = "idle"
	// StateThinking means the advisor is currently analyzing
	StateThinking AdvisorState = "thinking"
	// StateHasInsights means the advisor has unread insights
	StateHasInsights AdvisorState = "has_insights"
	// StateError means the advisor encountered an error
	StateError AdvisorState = "error"
)

// Advisor represents a specialized Claude subagent with a focused context
type Advisor struct {
	mu sync.RWMutex

	// Configuration
	config Config

	// State
	state    AdvisorState
	lastRun  time.Time
	lastErr  error
	running  bool
	insights []*Insight

	// Metrics (real usage data)
	metrics AdvisorMetrics

	// Location (for map rendering)
	x float64
	y float64
}

// AdvisorMetrics contains real usage statistics for an advisor
type AdvisorMetrics struct {
	// Execution counts
	TotalRuns    int
	SuccessCount int
	ErrorCount   int
	InsightCount int
	CancelCount  int

	// Timing statistics (zero when TotalRuns == 0)
	TotalDuration time.Duration
	AvgDuration   time.Duration
	MinDuration   time.Duration
	MaxDuration   time.Duration
	LastDuration  time.Duration

	// Token usage (real API costs)
	TotalTokensIn  int64
	TotalTokensOut int64

	// Insight quality (user feedback)
	AcceptedInsights int
	RejectedInsights int
}

// New creates a new advisor from configuration
//
// Assumptions:
// - config has been validated
// - x, y are valid map coordinates
//
// Edge cases:
// - config.ID is empty -> will use empty string as ID
func New(config Config, x, y float64) *Advisor {
	return &Advisor{
		config:   config,
		state:    StateIdle,
		insights: make([]*Insight, 0),
		x:        x,
		y:        y,
	}
}

// ID returns the advisor's unique identifier
func (a *Advisor) ID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.ID
}

// Name returns the advisor's display name
func (a *Advisor) Name() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Name
}

// Icon returns the advisor's icon identifier
func (a *Advisor) Icon() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Icon
}

// SystemPrompt returns the advisor's system prompt
func (a *Advisor) SystemPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.SystemPrompt
}

// Trigger returns the advisor's trigger mode
func (a *Advisor) Trigger() TriggerMode {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.Trigger
}

// FocusPatterns returns a copy of the advisor's focus patterns
func (a *Advisor) FocusPatterns() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	patterns := make([]string, len(a.config.FocusPatterns))
	copy(patterns, a.config.FocusPatterns)
	return patterns
}

// Config returns a copy of the advisor's configuration
func (a *Advisor) Config() Config {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

// State returns the current advisor state
func (a *Advisor) State() AdvisorState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.state
}

// Position returns the advisor's map coordinates
func (a *Advisor) Position() (x, y float64) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.x, a.y
}

// SetPosition updates the advisor's map coordinates
func (a *Advisor) SetPosition(x, y float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.x = x
	a.y = y
}

// Metrics returns the advisor's usage metrics
func (a *Advisor) Metrics() AdvisorMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.metrics
}

// LastRun returns when the advisor last ran
func (a *Advisor) LastRun() time.Time {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastRun
}

// LastError returns the last error the advisor encountered
func (a *Advisor) LastError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.lastErr
}

// IsRunning returns whether the advisor is currently running
func (a *Advisor) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}

// StartAnalysis marks the advisor as currently analyzing
//
// Returns false if the advisor is already running
func (a *Advisor) StartAnalysis() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return false
	}

	a.running = true
	a.state = StateThinking
	a.lastRun = time.Now()
	return true
}

// CompleteAnalysis marks the analysis as complete and records metrics
//
// Assumptions:
// - StartAnalysis was called first
// - duration is the actual wall-clock time taken
// - tokensIn/tokensOut are the real API token counts
//
// Edge cases:
// - Called when not running -> still updates metrics
// - duration is 0 -> accepts it
func (a *Advisor) CompleteAnalysis(duration time.Duration, tokensIn, tokensOut int64, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.running = false
	a.lastErr = err

	// Update metrics
	a.metrics.TotalRuns++
	a.metrics.TotalDuration += duration
	a.metrics.LastDuration = duration
	a.metrics.TotalTokensIn += tokensIn
	a.metrics.TotalTokensOut += tokensOut

	// Update min/max
	if a.metrics.TotalRuns == 1 {
		a.metrics.MinDuration = duration
		a.metrics.MaxDuration = duration
	} else {
		if duration < a.metrics.MinDuration {
			a.metrics.MinDuration = duration
		}
		if duration > a.metrics.MaxDuration {
			a.metrics.MaxDuration = duration
		}
	}

	// Update average
	if a.metrics.TotalRuns > 0 {
		a.metrics.AvgDuration = a.metrics.TotalDuration / time.Duration(a.metrics.TotalRuns)
	}

	// Update state based on result
	if err != nil {
		a.state = StateError
		a.metrics.ErrorCount++
	} else {
		a.metrics.SuccessCount++
		if len(a.insights) > 0 {
			a.state = StateHasInsights
		} else {
			a.state = StateIdle
		}
	}
}

// CancelAnalysis cancels a running analysis
//
// Returns false if no analysis was running
func (a *Advisor) CancelAnalysis() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return false
	}

	a.running = false
	a.state = StateIdle
	a.metrics.CancelCount++
	return true
}

// AddInsight adds a new insight from this advisor
//
// Assumptions:
// - insight is not nil
// - insight has been created by this advisor
func (a *Advisor) AddInsight(insight *Insight) {
	if insight == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.insights = append(a.insights, insight)
	a.metrics.InsightCount++

	if !a.running {
		a.state = StateHasInsights
	}
}

// Insights returns a copy of unread insights
func (a *Advisor) Insights() []*Insight {
	a.mu.RLock()
	defer a.mu.RUnlock()

	insights := make([]*Insight, len(a.insights))
	copy(insights, a.insights)
	return insights
}

// UnreadInsightCount returns the number of unread insights
func (a *Advisor) UnreadInsightCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	count := 0
	for _, insight := range a.insights {
		if insight.State == InsightStatePending {
			count++
		}
	}
	return count
}

// ClearInsights removes all insights
func (a *Advisor) ClearInsights() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.insights = make([]*Insight, 0)
	if a.state == StateHasInsights {
		a.state = StateIdle
	}
}

// MarkInsightAccepted records that a user accepted an insight
func (a *Advisor) MarkInsightAccepted(insightID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, insight := range a.insights {
		if insight.ID == insightID {
			insight.Accept()
			a.metrics.AcceptedInsights++
			return true
		}
	}
	return false
}

// MarkInsightRejected records that a user rejected an insight
func (a *Advisor) MarkInsightRejected(insightID string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, insight := range a.insights {
		if insight.ID == insightID {
			insight.Reject()
			a.metrics.RejectedInsights++
			return true
		}
	}
	return false
}

// MatchesFile checks if this advisor should analyze the given file
func (a *Advisor) MatchesFile(filePath string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config.MatchesFile(filePath)
}

// ShouldTriggerOnFileChange returns true if this advisor should run when a file changes
func (a *Advisor) ShouldTriggerOnFileChange(filePath string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.config.Trigger != TriggerOnFileChange {
		return false
	}

	return a.config.MatchesFile(filePath)
}

// BackgroundInterval returns the interval for background analysis
// Returns 0 if the advisor is not configured for background trigger
func (a *Advisor) BackgroundInterval() time.Duration {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.config.Trigger != TriggerBackground {
		return 0
	}

	return time.Duration(a.config.BackgroundIntervalSecs) * time.Second
}

// SuccessRate returns the percentage of successful runs
func (a *Advisor) SuccessRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.metrics.TotalRuns == 0 {
		return 0
	}

	return float64(a.metrics.SuccessCount) / float64(a.metrics.TotalRuns) * 100.0
}

// AcceptanceRate returns the percentage of insights that were accepted
func (a *Advisor) AcceptanceRate() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	total := a.metrics.AcceptedInsights + a.metrics.RejectedInsights
	if total == 0 {
		return 0
	}

	return float64(a.metrics.AcceptedInsights) / float64(total) * 100.0
}
