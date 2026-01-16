package advisor

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// InsightSeverity indicates the importance of an insight
type InsightSeverity string

const (
	// SeverityInfo is informational - nice to know
	SeverityInfo InsightSeverity = "info"
	// SeverityWarning requires attention but not urgent
	SeverityWarning InsightSeverity = "warning"
	// SeverityCritical requires immediate attention
	SeverityCritical InsightSeverity = "critical"
)

// InsightState tracks the lifecycle of an insight
type InsightState string

const (
	// InsightStatePending means the insight is awaiting user action
	InsightStatePending InsightState = "pending"
	// InsightStateAccepted means the user accepted the insight
	InsightStateAccepted InsightState = "accepted"
	// InsightStateRejected means the user rejected the insight
	InsightStateRejected InsightState = "rejected"
	// InsightStateDismissed means the user dismissed without action
	InsightStateDismissed InsightState = "dismissed"
)

// InsightCategory categorizes the type of insight
type InsightCategory string

const (
	// CategorySecurity for security-related insights
	CategorySecurity InsightCategory = "security"
	// CategoryPerformance for performance-related insights
	CategoryPerformance InsightCategory = "performance"
	// CategoryRefactoring for code quality insights
	CategoryRefactoring InsightCategory = "refactoring"
	// CategoryTesting for test-related insights
	CategoryTesting InsightCategory = "testing"
	// CategoryGeneral for other insights
	CategoryGeneral InsightCategory = "general"
)

// Insight represents a piece of advice from an advisor
type Insight struct {
	// ID is a unique identifier for this insight
	ID string
	// AdvisorID identifies which advisor generated this
	AdvisorID string
	// Timestamp is when the insight was generated
	Timestamp time.Time

	// Content
	Title       string          // Short summary (e.g., "Hardcoded secret detected")
	Description string          // Detailed explanation
	Severity    InsightSeverity // How important is this
	Category    InsightCategory // What type of insight

	// Location (optional - where in the code this applies)
	FilePath string // e.g., "src/auth/login.ts"
	Line     int    // e.g., 42
	Column   int    // e.g., 15 (optional)

	// Suggestion (optional - recommended action)
	Suggestion string // e.g., "Move secret to environment variable"
	CodeBefore string // Code snippet before fix
	CodeAfter  string // Code snippet after fix

	// State tracking
	State     InsightState
	StateTime time.Time // When state last changed
}

// NewInsight creates a new insight with a unique ID
//
// Assumptions:
// - advisorID is the ID of the advisor creating this insight
// - title and description are non-empty
//
// Edge cases:
// - Empty title/description -> allowed but not recommended
// - Empty advisorID -> allowed
func NewInsight(advisorID, title, description string, severity InsightSeverity, category InsightCategory) *Insight {
	return &Insight{
		ID:          generateInsightID(),
		AdvisorID:   advisorID,
		Timestamp:   time.Now(),
		Title:       title,
		Description: description,
		Severity:    severity,
		Category:    category,
		State:       InsightStatePending,
		StateTime:   time.Now(),
	}
}

// generateInsightID creates a unique identifier for an insight
func generateInsightID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))
	}
	return hex.EncodeToString(bytes)
}

// WithLocation adds file location information to the insight
func (i *Insight) WithLocation(filePath string, line, column int) *Insight {
	i.FilePath = filePath
	i.Line = line
	i.Column = column
	return i
}

// WithSuggestion adds a suggested fix to the insight
func (i *Insight) WithSuggestion(suggestion, codeBefore, codeAfter string) *Insight {
	i.Suggestion = suggestion
	i.CodeBefore = codeBefore
	i.CodeAfter = codeAfter
	return i
}

// Accept marks the insight as accepted by the user
func (i *Insight) Accept() {
	i.State = InsightStateAccepted
	i.StateTime = time.Now()
}

// Reject marks the insight as rejected by the user
func (i *Insight) Reject() {
	i.State = InsightStateRejected
	i.StateTime = time.Now()
}

// Dismiss marks the insight as dismissed without explicit action
func (i *Insight) Dismiss() {
	i.State = InsightStateDismissed
	i.StateTime = time.Now()
}

// IsPending returns true if the insight is awaiting user action
func (i *Insight) IsPending() bool {
	return i.State == InsightStatePending
}

// IsResolved returns true if the user has acted on the insight
func (i *Insight) IsResolved() bool {
	return i.State == InsightStateAccepted || i.State == InsightStateRejected || i.State == InsightStateDismissed
}

// HasLocation returns true if the insight has file location information
func (i *Insight) HasLocation() bool {
	return i.FilePath != "" && i.Line > 0
}

// HasSuggestion returns true if the insight has a suggested fix
func (i *Insight) HasSuggestion() bool {
	return i.Suggestion != ""
}

// InsightBuilder provides a fluent interface for creating insights
type InsightBuilder struct {
	insight *Insight
}

// BuildInsight starts building a new insight
func BuildInsight(advisorID string) *InsightBuilder {
	return &InsightBuilder{
		insight: &Insight{
			ID:        generateInsightID(),
			AdvisorID: advisorID,
			Timestamp: time.Now(),
			State:     InsightStatePending,
			StateTime: time.Now(),
			Severity:  SeverityInfo,
			Category:  CategoryGeneral,
		},
	}
}

// Title sets the insight title
func (b *InsightBuilder) Title(title string) *InsightBuilder {
	b.insight.Title = title
	return b
}

// Description sets the insight description
func (b *InsightBuilder) Description(description string) *InsightBuilder {
	b.insight.Description = description
	return b
}

// Severity sets the insight severity
func (b *InsightBuilder) Severity(severity InsightSeverity) *InsightBuilder {
	b.insight.Severity = severity
	return b
}

// Category sets the insight category
func (b *InsightBuilder) Category(category InsightCategory) *InsightBuilder {
	b.insight.Category = category
	return b
}

// Location sets the file location for the insight
func (b *InsightBuilder) Location(filePath string, line, column int) *InsightBuilder {
	b.insight.FilePath = filePath
	b.insight.Line = line
	b.insight.Column = column
	return b
}

// Suggestion sets the suggested fix
func (b *InsightBuilder) Suggestion(suggestion, codeBefore, codeAfter string) *InsightBuilder {
	b.insight.Suggestion = suggestion
	b.insight.CodeBefore = codeBefore
	b.insight.CodeAfter = codeAfter
	return b
}

// Build finalizes and returns the insight
func (b *InsightBuilder) Build() *Insight {
	return b.insight
}

// InsightFilter provides methods for filtering insights
type InsightFilter struct {
	insights []*Insight
}

// FilterInsights creates a new filter for the given insights
func FilterInsights(insights []*Insight) *InsightFilter {
	return &InsightFilter{insights: insights}
}

// BySeverity filters insights by severity
func (f *InsightFilter) BySeverity(severity InsightSeverity) *InsightFilter {
	var filtered []*Insight
	for _, i := range f.insights {
		if i.Severity == severity {
			filtered = append(filtered, i)
		}
	}
	return &InsightFilter{insights: filtered}
}

// ByCategory filters insights by category
func (f *InsightFilter) ByCategory(category InsightCategory) *InsightFilter {
	var filtered []*Insight
	for _, i := range f.insights {
		if i.Category == category {
			filtered = append(filtered, i)
		}
	}
	return &InsightFilter{insights: filtered}
}

// ByAdvisor filters insights by advisor ID
func (f *InsightFilter) ByAdvisor(advisorID string) *InsightFilter {
	var filtered []*Insight
	for _, i := range f.insights {
		if i.AdvisorID == advisorID {
			filtered = append(filtered, i)
		}
	}
	return &InsightFilter{insights: filtered}
}

// ByState filters insights by state
func (f *InsightFilter) ByState(state InsightState) *InsightFilter {
	var filtered []*Insight
	for _, i := range f.insights {
		if i.State == state {
			filtered = append(filtered, i)
		}
	}
	return &InsightFilter{insights: filtered}
}

// ByFile filters insights by file path
func (f *InsightFilter) ByFile(filePath string) *InsightFilter {
	var filtered []*Insight
	for _, i := range f.insights {
		if i.FilePath == filePath {
			filtered = append(filtered, i)
		}
	}
	return &InsightFilter{insights: filtered}
}

// Pending returns only pending insights
func (f *InsightFilter) Pending() *InsightFilter {
	return f.ByState(InsightStatePending)
}

// Critical returns only critical insights
func (f *InsightFilter) Critical() *InsightFilter {
	return f.BySeverity(SeverityCritical)
}

// Results returns the filtered insights
func (f *InsightFilter) Results() []*Insight {
	return f.insights
}

// Count returns the number of filtered insights
func (f *InsightFilter) Count() int {
	return len(f.insights)
}
