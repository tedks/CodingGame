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
