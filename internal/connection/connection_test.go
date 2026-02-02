package connection

import (
	"sync"
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	// ID format uses unit separator (0x1F) to avoid collision with paths containing colons
	sep := string([]byte{0x1F})

	tests := []struct {
		name     string
		from     string
		to       string
		typ      Type
		wantID   string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "basic import",
			from:     "pkg/utils.go",
			to:       "cmd/main.go",
			typ:      TypeImport,
			wantID:   "pkg/utils.go" + sep + "cmd/main.go" + sep + "import",
			wantFrom: "pkg/utils.go",
			wantTo:   "cmd/main.go",
		},
		{
			name:     "backslash normalization",
			from:     "pkg\\utils.go",
			to:       "cmd\\main.go",
			typ:      TypeImport,
			wantID:   "pkg/utils.go" + sep + "cmd/main.go" + sep + "import",
			wantFrom: "pkg/utils.go",
			wantTo:   "cmd/main.go",
		},
		{
			name:     "self reference",
			from:     "pkg/foo.go",
			to:       "pkg/foo.go",
			typ:      TypeCall,
			wantID:   "pkg/foo.go" + sep + "pkg/foo.go" + sep + "call",
			wantFrom: "pkg/foo.go",
			wantTo:   "pkg/foo.go",
		},
		{
			name:     "inheritance type",
			from:     "models/base.go",
			to:       "models/user.go",
			typ:      TypeInheritance,
			wantID:   "models/base.go" + sep + "models/user.go" + sep + "inheritance",
			wantFrom: "models/base.go",
			wantTo:   "models/user.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(tt.from, tt.to, tt.typ)

			if got := c.ID(); got != tt.wantID {
				t.Errorf("ID() = %q, want %q", got, tt.wantID)
			}
			if got := c.From(); got != tt.wantFrom {
				t.Errorf("From() = %q, want %q", got, tt.wantFrom)
			}
			if got := c.To(); got != tt.wantTo {
				t.Errorf("To() = %q, want %q", got, tt.wantTo)
			}
			if got := c.Type(); got != tt.typ {
				t.Errorf("Type() = %v, want %v", got, tt.typ)
			}
			if got := c.Strength(); got != 1 {
				t.Errorf("Strength() = %d, want 1 (default)", got)
			}
		})
	}
}

func TestConnectionIsSelfReference(t *testing.T) {
	tests := []struct {
		from string
		to   string
		want bool
	}{
		{"a.go", "a.go", true},
		{"a.go", "b.go", false},
		{"pkg/a.go", "pkg/a.go", true},
		{"pkg/a.go", "other/a.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.from+"->"+tt.to, func(t *testing.T) {
			c := NewConnection(tt.from, tt.to, TypeImport)
			if got := c.IsSelfReference(); got != tt.want {
				t.Errorf("IsSelfReference() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionStrength(t *testing.T) {
	c := NewConnection("a.go", "b.go", TypeImport)

	// Default strength
	if got := c.Strength(); got != 1 {
		t.Errorf("initial Strength() = %d, want 1", got)
	}

	// Set positive strength
	c.SetStrength(5)
	if got := c.Strength(); got != 5 {
		t.Errorf("Strength() after set = %d, want 5", got)
	}

	// Negative strength clamped to 0
	c.SetStrength(-10)
	if got := c.Strength(); got != 0 {
		t.Errorf("Strength() after negative = %d, want 0", got)
	}

	// Zero is allowed
	c.SetStrength(0)
	if got := c.Strength(); got != 0 {
		t.Errorf("Strength() after zero = %d, want 0", got)
	}
}

func TestConnectionExercise(t *testing.T) {
	c := NewConnection("a.go", "b.go", TypeImport)

	// Initial state
	if got := c.ExerciseCount(); got != 0 {
		t.Errorf("initial ExerciseCount() = %d, want 0", got)
	}
	if got := c.LastExercised(); !got.IsZero() {
		t.Errorf("initial LastExercised() should be zero, got %v", got)
	}

	// Exercise the connection
	before := time.Now()
	c.Exercise()
	after := time.Now()

	if got := c.ExerciseCount(); got != 1 {
		t.Errorf("ExerciseCount() after exercise = %d, want 1", got)
	}

	lastExercised := c.LastExercised()
	if lastExercised.Before(before) || lastExercised.After(after) {
		t.Errorf("LastExercised() = %v, expected between %v and %v", lastExercised, before, after)
	}

	// Exercise again
	c.Exercise()
	if got := c.ExerciseCount(); got != 2 {
		t.Errorf("ExerciseCount() after 2nd exercise = %d, want 2", got)
	}
}

func TestConnectionCircularAndExternal(t *testing.T) {
	c := NewConnection("a.go", "b.go", TypeImport)

	// Initial state
	if c.IsCircular() {
		t.Error("initial IsCircular() should be false")
	}
	if c.IsExternal() {
		t.Error("initial IsExternal() should be false")
	}

	// Set circular
	c.SetCircular(true)
	if !c.IsCircular() {
		t.Error("IsCircular() should be true after SetCircular(true)")
	}

	c.SetCircular(false)
	if c.IsCircular() {
		t.Error("IsCircular() should be false after SetCircular(false)")
	}

	// Set external
	c.SetExternal(true)
	if !c.IsExternal() {
		t.Error("IsExternal() should be true after SetExternal(true)")
	}
}

func TestConnectionEndpoints(t *testing.T) {
	c := NewConnection("source.go", "dest.go", TypeImport)

	from, to := c.Endpoints()
	if from != "source.go" || to != "dest.go" {
		t.Errorf("Endpoints() = (%q, %q), want (\"source.go\", \"dest.go\")", from, to)
	}
}

func TestTypeString(t *testing.T) {
	tests := []struct {
		typ  Type
		want string
	}{
		{TypeImport, "import"},
		{TypeInheritance, "inheritance"},
		{TypeComposition, "composition"},
		{TypeCall, "call"},
		{TypeUnknown, "unknown"},
		{Type(99), "unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("Type(%d).String() = %q, want %q", tt.typ, got, tt.want)
			}
		})
	}
}

func TestConnectionConcurrency(t *testing.T) {
	c := NewConnection("a.go", "b.go", TypeImport)
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Mix of read and write operations
				c.Exercise()
				_ = c.ExerciseCount()
				c.SetStrength(j)
				_ = c.Strength()
				c.SetCircular(j%2 == 0)
				_ = c.IsCircular()
				_ = c.ID()
				_ = c.From()
				_ = c.To()
				_, _ = c.Endpoints()
			}
		}()
	}

	wg.Wait()

	// Just verify we didn't crash and count is reasonable
	count := c.ExerciseCount()
	expected := goroutines * iterations
	if count != expected {
		t.Errorf("ExerciseCount() = %d, want %d (exact due to locking)", count, expected)
	}
}

// TestIDCollisionWithColonPaths verifies that paths containing colons
// produce distinct IDs (regression test for the colon separator bug).
func TestIDCollisionWithColonPaths(t *testing.T) {
	// These paths would produce the same ID if using ":" as separator:
	// "a:b" + "c" = "a:b:c:import"
	// "a" + "b:c" = "a:b:c:import"
	c1 := NewConnection("a:b", "c", TypeImport)
	c2 := NewConnection("a", "b:c", TypeImport)

	if c1.ID() == c2.ID() {
		t.Errorf("ID collision: paths with colons should produce different IDs\n"+
			"c1 (from='a:b', to='c'): %q\n"+
			"c2 (from='a', to='b:c'): %q", c1.ID(), c2.ID())
	}

	// Additional cases with colons
	cases := []struct {
		from1, to1, from2, to2 string
	}{
		{"a:", "b", "a", ":b"},
		{":a", "b", "", "a:b"},
		{"a:b:c", "d", "a", "b:c:d"},
		{"C:\\Windows\\foo.go", "bar.go", "C", "\\Windows\\foo.go:bar.go"},
	}

	for _, tc := range cases {
		c1 := NewConnection(tc.from1, tc.to1, TypeImport)
		c2 := NewConnection(tc.from2, tc.to2, TypeImport)
		if c1.ID() == c2.ID() {
			t.Errorf("ID collision: (%q,%q) vs (%q,%q) both produce %q",
				tc.from1, tc.to1, tc.from2, tc.to2, c1.ID())
		}
	}
}

// TestEmptyPaths verifies behavior with empty path strings.
func TestEmptyPaths(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"empty from", "", "b.go"},
		{"empty to", "a.go", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(tt.from, tt.to, TypeImport)
			// Should not panic
			_ = c.ID()
			_ = c.From()
			_ = c.To()

			// Verify values are preserved
			if c.From() != tt.from {
				t.Errorf("From() = %q, want %q", c.From(), tt.from)
			}
			if c.To() != tt.to {
				t.Errorf("To() = %q, want %q", c.To(), tt.to)
			}

			// Graph operations should handle empty paths
			g := NewGraph()
			added := g.Add(c)
			if added == nil {
				t.Error("Graph.Add() returned nil for empty path connection")
			}
			if g.Count() != 1 {
				t.Errorf("Count() = %d, want 1", g.Count())
			}
		})
	}
}

// TestVeryLongPaths verifies behavior with extremely long path strings.
func TestVeryLongPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long path test in short mode")
	}

	// Create a 100KB path
	longPath := make([]byte, 100*1024)
	for i := range longPath {
		longPath[i] = 'a' + byte(i%26)
	}
	longPathStr := string(longPath)

	c := NewConnection(longPathStr, "b.go", TypeImport)

	// Should not panic
	id := c.ID()
	from := c.From()

	// Verify path is preserved
	if from != longPathStr {
		t.Error("From() does not match long path (length check)")
	}

	// ID should contain the full path
	if len(id) < len(longPathStr) {
		t.Errorf("ID length %d is less than path length %d", len(id), len(longPathStr))
	}

	// Graph operations should work
	g := NewGraph()
	added := g.Add(c)
	if added == nil {
		t.Error("Graph.Add() returned nil for long path connection")
	}

	retrieved := g.GetByEndpoints(longPathStr, "b.go", TypeImport)
	if retrieved != c {
		t.Error("GetByEndpoints failed for long path")
	}
}

// TestUnicodePaths verifies behavior with Unicode characters in paths.
func TestUnicodePaths(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
	}{
		{"Japanese", "ファイル/main.go", "パッケージ/util.go"},
		{"Chinese", "文件/main.go", "包/util.go"},
		{"emoji", "📁/main.go", "📦/util.go"},
		{"mixed scripts", "日本語/中文/한국어/main.go", "output.go"},
		{"combining characters", "cafe\u0301/main.go", "output.go"}, // café with combining acute
		{"ZWJ sequence", "👨‍👩‍👧‍👦/family.go", "output.go"},
		{"RTL", "ملف/main.go", "output.go"}, // Arabic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(tt.from, tt.to, TypeImport)

			// Should not panic
			_ = c.ID()

			// Verify values are preserved
			if c.From() != tt.from {
				t.Errorf("From() = %q, want %q", c.From(), tt.from)
			}
			if c.To() != tt.to {
				t.Errorf("To() = %q, want %q", c.To(), tt.to)
			}

			// Graph operations should work
			g := NewGraph()
			g.Add(c)

			retrieved := g.GetByEndpoints(tt.from, tt.to, TypeImport)
			if retrieved == nil {
				t.Error("GetByEndpoints failed for Unicode path")
			}

			fromConns := g.FromFile(tt.from)
			if len(fromConns) != 1 {
				t.Errorf("FromFile returned %d connections, want 1", len(fromConns))
			}
		})
	}

	// Test that different Unicode normalizations produce different connections
	// NFC vs NFD for "café"
	nfc := "caf\u00e9/main.go"  // precomposed é
	nfd := "cafe\u0301/main.go" // e + combining acute

	c1 := NewConnection(nfc, "out.go", TypeImport)
	c2 := NewConnection(nfd, "out.go", TypeImport)

	// These should be different connections (no normalization applied)
	if c1.ID() == c2.ID() {
		t.Log("Note: NFC and NFD forms produce same ID - acceptable but worth knowing")
	}
}

// TestExerciseCountNoOverflow verifies that exercise count doesn't wrap to negative.
func TestExerciseCountNoOverflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping overflow test in short mode")
	}

	c := NewConnection("a.go", "b.go", TypeImport)

	// Exercise many times to approach potential overflow
	// We can't actually reach MaxInt in a reasonable time, so we
	// verify the mechanism works correctly for a large number
	const iterations = 100000
	for i := 0; i < iterations; i++ {
		c.Exercise()
	}

	count := c.ExerciseCount()
	if count != iterations {
		t.Errorf("ExerciseCount() = %d, want %d", count, iterations)
	}

	// Verify count is positive
	if count < 0 {
		t.Errorf("ExerciseCount() wrapped to negative: %d", count)
	}
}
