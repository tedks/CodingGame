package advisor

import (
	"testing"
	"time"
)

func TestNewInsight(t *testing.T) {
	insight := NewInsight("security", "Hardcoded Secret", "Found API key in source", SeverityCritical, CategorySecurity)

	if insight.ID == "" {
		t.Error("NewInsight() ID is empty")
	}
	if insight.AdvisorID != "security" {
		t.Errorf("AdvisorID = %q, want security", insight.AdvisorID)
	}
	if insight.Title != "Hardcoded Secret" {
		t.Errorf("Title = %q, want 'Hardcoded Secret'", insight.Title)
	}
	if insight.Description != "Found API key in source" {
		t.Errorf("Description = %q, want 'Found API key in source'", insight.Description)
	}
	if insight.Severity != SeverityCritical {
		t.Errorf("Severity = %q, want %q", insight.Severity, SeverityCritical)
	}
	if insight.Category != CategorySecurity {
		t.Errorf("Category = %q, want %q", insight.Category, CategorySecurity)
	}
	if insight.State != InsightStatePending {
		t.Errorf("initial State = %q, want %q", insight.State, InsightStatePending)
	}
	if insight.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestInsight_WithLocation(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityWarning, CategoryGeneral)
	insight.WithLocation("src/auth/login.ts", 42, 15)

	if insight.FilePath != "src/auth/login.ts" {
		t.Errorf("FilePath = %q, want 'src/auth/login.ts'", insight.FilePath)
	}
	if insight.Line != 42 {
		t.Errorf("Line = %d, want 42", insight.Line)
	}
	if insight.Column != 15 {
		t.Errorf("Column = %d, want 15", insight.Column)
	}
}

func TestInsight_WithSuggestion(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityWarning, CategoryGeneral)
	insight.WithSuggestion(
		"Move secret to environment variable",
		"const API_KEY = 'abc123';",
		"const API_KEY = process.env.API_KEY;",
	)

	if insight.Suggestion != "Move secret to environment variable" {
		t.Errorf("Suggestion = %q, want 'Move secret to environment variable'", insight.Suggestion)
	}
	if insight.CodeBefore != "const API_KEY = 'abc123';" {
		t.Errorf("CodeBefore = %q", insight.CodeBefore)
	}
	if insight.CodeAfter != "const API_KEY = process.env.API_KEY;" {
		t.Errorf("CodeAfter = %q", insight.CodeAfter)
	}
}

func TestInsight_ChainedMethods(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityWarning, CategoryGeneral).
		WithLocation("file.go", 10, 5).
		WithSuggestion("Fix it", "before", "after")

	if insight.FilePath != "file.go" {
		t.Error("chained WithLocation failed")
	}
	if insight.Suggestion != "Fix it" {
		t.Error("chained WithSuggestion failed")
	}
}

func TestInsight_Accept(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)
	beforeTime := insight.StateTime

	time.Sleep(1 * time.Millisecond) // Ensure time difference
	insight.Accept()

	if insight.State != InsightStateAccepted {
		t.Errorf("after Accept(), State = %q, want %q", insight.State, InsightStateAccepted)
	}
	if !insight.StateTime.After(beforeTime) {
		t.Error("StateTime was not updated")
	}
}

func TestInsight_Reject(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)

	insight.Reject()

	if insight.State != InsightStateRejected {
		t.Errorf("after Reject(), State = %q, want %q", insight.State, InsightStateRejected)
	}
}

func TestInsight_Dismiss(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)

	insight.Dismiss()

	if insight.State != InsightStateDismissed {
		t.Errorf("after Dismiss(), State = %q, want %q", insight.State, InsightStateDismissed)
	}
}

func TestInsight_IsPending(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)

	if !insight.IsPending() {
		t.Error("new insight IsPending() = false, want true")
	}

	insight.Accept()

	if insight.IsPending() {
		t.Error("accepted insight IsPending() = true, want false")
	}
}

func TestInsight_IsResolved(t *testing.T) {
	tests := []struct {
		name     string
		action   func(*Insight)
		resolved bool
	}{
		{"pending", func(i *Insight) {}, false},
		{"accepted", func(i *Insight) { i.Accept() }, true},
		{"rejected", func(i *Insight) { i.Reject() }, true},
		{"dismissed", func(i *Insight) { i.Dismiss() }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)
			tt.action(insight)

			if insight.IsResolved() != tt.resolved {
				t.Errorf("IsResolved() = %v, want %v", insight.IsResolved(), tt.resolved)
			}
		})
	}
}

func TestInsight_HasLocation(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)

	if insight.HasLocation() {
		t.Error("new insight HasLocation() = true, want false")
	}

	insight.WithLocation("file.go", 10, 0)

	if !insight.HasLocation() {
		t.Error("after WithLocation(), HasLocation() = false, want true")
	}

	// Line 0 should still be false
	insight2 := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)
	insight2.FilePath = "file.go"
	insight2.Line = 0

	if insight2.HasLocation() {
		t.Error("insight with line 0 HasLocation() = true, want false")
	}
}

func TestInsight_HasSuggestion(t *testing.T) {
	insight := NewInsight("test", "Title", "Description", SeverityInfo, CategoryGeneral)

	if insight.HasSuggestion() {
		t.Error("new insight HasSuggestion() = true, want false")
	}

	insight.WithSuggestion("Fix it", "", "")

	if !insight.HasSuggestion() {
		t.Error("after WithSuggestion(), HasSuggestion() = false, want true")
	}
}

func TestInsightSeverity_Constants(t *testing.T) {
	severities := map[InsightSeverity]string{
		SeverityInfo:     "info",
		SeverityWarning:  "warning",
		SeverityCritical: "critical",
	}

	for severity, expected := range severities {
		if string(severity) != expected {
			t.Errorf("InsightSeverity %v = %q, want %q", severity, string(severity), expected)
		}
	}

	// Verify distinctness
	unique := make(map[InsightSeverity]bool)
	unique[SeverityInfo] = true
	unique[SeverityWarning] = true
	unique[SeverityCritical] = true

	if len(unique) != 3 {
		t.Error("InsightSeverity constants are not distinct")
	}
}

func TestInsightState_Constants(t *testing.T) {
	states := map[InsightState]string{
		InsightStatePending:   "pending",
		InsightStateAccepted:  "accepted",
		InsightStateRejected:  "rejected",
		InsightStateDismissed: "dismissed",
	}

	for state, expected := range states {
		if string(state) != expected {
			t.Errorf("InsightState %v = %q, want %q", state, string(state), expected)
		}
	}

	// Verify distinctness
	unique := make(map[InsightState]bool)
	unique[InsightStatePending] = true
	unique[InsightStateAccepted] = true
	unique[InsightStateRejected] = true
	unique[InsightStateDismissed] = true

	if len(unique) != 4 {
		t.Error("InsightState constants are not distinct")
	}
}

func TestInsightCategory_Constants(t *testing.T) {
	categories := map[InsightCategory]string{
		CategorySecurity:    "security",
		CategoryPerformance: "performance",
		CategoryRefactoring: "refactoring",
		CategoryTesting:     "testing",
		CategoryGeneral:     "general",
	}

	for cat, expected := range categories {
		if string(cat) != expected {
			t.Errorf("InsightCategory %v = %q, want %q", cat, string(cat), expected)
		}
	}

	// Verify distinctness
	unique := make(map[InsightCategory]bool)
	unique[CategorySecurity] = true
	unique[CategoryPerformance] = true
	unique[CategoryRefactoring] = true
	unique[CategoryTesting] = true
	unique[CategoryGeneral] = true

	if len(unique) != 5 {
		t.Error("InsightCategory constants are not distinct")
	}
}

func TestInsightBuilder(t *testing.T) {
	insight := BuildInsight("security").
		Title("SQL Injection Risk").
		Description("User input is not sanitized").
		Severity(SeverityCritical).
		Category(CategorySecurity).
		Location("db/query.go", 55, 10).
		Suggestion("Use parameterized queries", "db.Query(sql + userInput)", "db.Query(sql, userInput)").
		Build()

	if insight.AdvisorID != "security" {
		t.Errorf("AdvisorID = %q, want security", insight.AdvisorID)
	}
	if insight.Title != "SQL Injection Risk" {
		t.Errorf("Title = %q, want 'SQL Injection Risk'", insight.Title)
	}
	if insight.Severity != SeverityCritical {
		t.Errorf("Severity = %q, want %q", insight.Severity, SeverityCritical)
	}
	if insight.FilePath != "db/query.go" {
		t.Errorf("FilePath = %q, want 'db/query.go'", insight.FilePath)
	}
	if insight.Line != 55 {
		t.Errorf("Line = %d, want 55", insight.Line)
	}
	if insight.Suggestion != "Use parameterized queries" {
		t.Errorf("Suggestion = %q", insight.Suggestion)
	}
}

func TestInsightBuilder_Defaults(t *testing.T) {
	insight := BuildInsight("test").Build()

	// Check defaults
	if insight.Severity != SeverityInfo {
		t.Errorf("default Severity = %q, want %q", insight.Severity, SeverityInfo)
	}
	if insight.Category != CategoryGeneral {
		t.Errorf("default Category = %q, want %q", insight.Category, CategoryGeneral)
	}
	if insight.State != InsightStatePending {
		t.Errorf("default State = %q, want %q", insight.State, InsightStatePending)
	}
	if insight.ID == "" {
		t.Error("ID should be generated")
	}
}

func TestFilterInsights_BySeverity(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Info 1", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Warning 1", "desc", SeverityWarning, CategoryGeneral),
		NewInsight("a", "Critical 1", "desc", SeverityCritical, CategoryGeneral),
		NewInsight("a", "Warning 2", "desc", SeverityWarning, CategoryGeneral),
	}

	warnings := FilterInsights(insights).BySeverity(SeverityWarning).Results()
	if len(warnings) != 2 {
		t.Errorf("BySeverity(warning) count = %d, want 2", len(warnings))
	}

	critical := FilterInsights(insights).BySeverity(SeverityCritical).Results()
	if len(critical) != 1 {
		t.Errorf("BySeverity(critical) count = %d, want 1", len(critical))
	}
}

func TestFilterInsights_ByCategory(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Sec 1", "desc", SeverityInfo, CategorySecurity),
		NewInsight("a", "Test 1", "desc", SeverityInfo, CategoryTesting),
		NewInsight("a", "Sec 2", "desc", SeverityInfo, CategorySecurity),
	}

	security := FilterInsights(insights).ByCategory(CategorySecurity).Results()
	if len(security) != 2 {
		t.Errorf("ByCategory(security) count = %d, want 2", len(security))
	}
}

func TestFilterInsights_ByAdvisor(t *testing.T) {
	insights := []*Insight{
		NewInsight("security", "Insight 1", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("testing", "Insight 2", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("security", "Insight 3", "desc", SeverityInfo, CategoryGeneral),
	}

	fromSecurity := FilterInsights(insights).ByAdvisor("security").Results()
	if len(fromSecurity) != 2 {
		t.Errorf("ByAdvisor(security) count = %d, want 2", len(fromSecurity))
	}
}

func TestFilterInsights_ByState(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Pending", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Accepted", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Pending 2", "desc", SeverityInfo, CategoryGeneral),
	}
	insights[1].Accept()

	pending := FilterInsights(insights).ByState(InsightStatePending).Results()
	if len(pending) != 2 {
		t.Errorf("ByState(pending) count = %d, want 2", len(pending))
	}

	accepted := FilterInsights(insights).ByState(InsightStateAccepted).Results()
	if len(accepted) != 1 {
		t.Errorf("ByState(accepted) count = %d, want 1", len(accepted))
	}
}

func TestFilterInsights_ByFile(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Insight 1", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Insight 2", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Insight 3", "desc", SeverityInfo, CategoryGeneral),
	}
	insights[0].WithLocation("auth.go", 10, 0)
	insights[1].WithLocation("db.go", 20, 0)
	insights[2].WithLocation("auth.go", 30, 0)

	authInsights := FilterInsights(insights).ByFile("auth.go").Results()
	if len(authInsights) != 2 {
		t.Errorf("ByFile(auth.go) count = %d, want 2", len(authInsights))
	}
}

func TestFilterInsights_Pending(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Pending", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Accepted", "desc", SeverityInfo, CategoryGeneral),
	}
	insights[1].Accept()

	pending := FilterInsights(insights).Pending().Results()
	if len(pending) != 1 {
		t.Errorf("Pending() count = %d, want 1", len(pending))
	}
}

func TestFilterInsights_Critical(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "Info", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "Critical", "desc", SeverityCritical, CategoryGeneral),
	}

	critical := FilterInsights(insights).Critical().Results()
	if len(critical) != 1 {
		t.Errorf("Critical() count = %d, want 1", len(critical))
	}
}

func TestFilterInsights_Chained(t *testing.T) {
	insights := []*Insight{
		NewInsight("security", "Sec Critical", "desc", SeverityCritical, CategorySecurity),
		NewInsight("security", "Sec Info", "desc", SeverityInfo, CategorySecurity),
		NewInsight("testing", "Test Critical", "desc", SeverityCritical, CategoryTesting),
	}

	// Chain multiple filters
	result := FilterInsights(insights).
		ByAdvisor("security").
		BySeverity(SeverityCritical).
		Results()

	if len(result) != 1 {
		t.Errorf("chained filter count = %d, want 1", len(result))
	}
	if result[0].Title != "Sec Critical" {
		t.Errorf("wrong insight returned: %q", result[0].Title)
	}
}

func TestFilterInsights_Count(t *testing.T) {
	insights := []*Insight{
		NewInsight("a", "1", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "2", "desc", SeverityInfo, CategoryGeneral),
		NewInsight("a", "3", "desc", SeverityWarning, CategoryGeneral),
	}

	count := FilterInsights(insights).BySeverity(SeverityInfo).Count()
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestFilterInsights_Empty(t *testing.T) {
	var insights []*Insight

	results := FilterInsights(insights).Results()
	if len(results) != 0 {
		t.Errorf("empty filter Results() length = %d, want 0", len(results))
	}

	count := FilterInsights(insights).Count()
	if count != 0 {
		t.Errorf("empty filter Count() = %d, want 0", count)
	}
}

func TestGenerateInsightID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)

	for i := 0; i < 1000; i++ {
		id := generateInsightID()
		if ids[id] {
			t.Errorf("duplicate ID generated: %s", id)
		}
		ids[id] = true
	}
}
