package build

import (
	"fmt"
	"testing"
	"time"
)

// mockAdapter is a test implementation of the Adapter interface
type mockAdapter struct {
	name           string
	detectResult   bool
	detectError    error
	targets        []Target
	targetsError   error
	buildResult    *Result
	buildError     error
	buildCallCount int
}

func (m *mockAdapter) Name() string {
	return m.name
}

func (m *mockAdapter) Detect(projectPath string) (bool, error) {
	return m.detectResult, m.detectError
}

func (m *mockAdapter) GetTargets(projectPath string) ([]Target, error) {
	return m.targets, m.targetsError
}

func (m *mockAdapter) Build(projectPath string, targetID string, opts *BuildOptions) (*Result, error) {
	m.buildCallCount++
	return m.buildResult, m.buildError
}

func TestResultCacheHitRate(t *testing.T) {
	tests := []struct {
		name        string
		cacheHits   int64
		cacheMisses int64
		want        float64
	}{
		{
			name:        "no cache operations",
			cacheHits:   0,
			cacheMisses: 0,
			want:        0,
		},
		{
			name:        "100% cache hits",
			cacheHits:   10,
			cacheMisses: 0,
			want:        100.0,
		},
		{
			name:        "0% cache hits",
			cacheHits:   0,
			cacheMisses: 10,
			want:        0.0,
		},
		{
			name:        "50% cache hits",
			cacheHits:   5,
			cacheMisses: 5,
			want:        50.0,
		},
		{
			name:        "75% cache hits",
			cacheHits:   75,
			cacheMisses: 25,
			want:        75.0,
		},
		{
			name:        "33.33% cache hits",
			cacheHits:   1,
			cacheMisses: 2,
			want:        33.333333333333336, // Floating point precision
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{
				CacheHits:   tt.cacheHits,
				CacheMisses: tt.cacheMisses,
			}
			got := r.CacheHitRate()
			// Use tolerance for floating point comparison
			const tolerance = 0.0000001
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				t.Errorf("CacheHitRate() = %v, want %v (diff: %v)", got, tt.want, diff)
			}
		})
	}
}

func TestRegistryRegister(t *testing.T) {
	reg := NewRegistry()

	if len(reg.adapters) != 0 {
		t.Errorf("new registry should have 0 adapters, got %d", len(reg.adapters))
	}

	adapter1 := &mockAdapter{name: "test1"}
	reg.Register(adapter1)

	if len(reg.adapters) != 1 {
		t.Errorf("after Register, should have 1 adapter, got %d", len(reg.adapters))
	}

	adapter2 := &mockAdapter{name: "test2"}
	reg.Register(adapter2)

	if len(reg.adapters) != 2 {
		t.Errorf("after second Register, should have 2 adapters, got %d", len(reg.adapters))
	}
}

func TestRegistryDetect(t *testing.T) {
	tests := []struct {
		name     string
		adapters []*mockAdapter
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "no adapters registered",
			adapters: []*mockAdapter{},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name: "single adapter detected",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: true},
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "single adapter not detected",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: false},
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "multiple adapters, all detected",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: true},
				{name: "npm", detectResult: true},
				{name: "cargo", detectResult: true},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name: "multiple adapters, some detected",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: true},
				{name: "npm", detectResult: false},
				{name: "cargo", detectResult: true},
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name: "adapter returns error, continue checking others",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: false, detectError: fmt.Errorf("permission denied")},
				{name: "npm", detectResult: true},
			},
			wantLen: 1, // npm still detected despite bazel error
			wantErr: false,
		},
		{
			name: "all adapters return errors",
			adapters: []*mockAdapter{
				{name: "bazel", detectResult: false, detectError: fmt.Errorf("error 1")},
				{name: "npm", detectResult: false, detectError: fmt.Errorf("error 2")},
			},
			wantLen: 0,
			wantErr: false, // Errors are swallowed, not propagated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := NewRegistry()
			for _, adapter := range tt.adapters {
				reg.Register(adapter)
			}

			detected, err := reg.Detect("/test/project")

			if (err != nil) != tt.wantErr {
				t.Errorf("Detect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(detected) != tt.wantLen {
				t.Errorf("Detect() returned %d adapters, want %d", len(detected), tt.wantLen)
			}
		})
	}
}

func TestRegistryGetAdapter(t *testing.T) {
	reg := NewRegistry()
	adapter1 := &mockAdapter{name: "bazel"}
	adapter2 := &mockAdapter{name: "npm"}
	reg.Register(adapter1)
	reg.Register(adapter2)

	tests := []struct {
		name      string
		adapterID string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "get existing adapter",
			adapterID: "bazel",
			wantName:  "bazel",
			wantErr:   false,
		},
		{
			name:      "get another existing adapter",
			adapterID: "npm",
			wantName:  "npm",
			wantErr:   false,
		},
		{
			name:      "get non-existent adapter",
			adapterID: "cargo",
			wantName:  "",
			wantErr:   true,
		},
		{
			name:      "empty adapter name",
			adapterID: "",
			wantName:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := reg.GetAdapter(tt.adapterID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetAdapter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if adapter.Name() != tt.wantName {
					t.Errorf("GetAdapter() returned adapter with name %q, want %q", adapter.Name(), tt.wantName)
				}
			}
		})
	}
}

func TestBuildOptions(t *testing.T) {
	// Test that BuildOptions fields are accessible and have correct zero values
	opts := &BuildOptions{}

	if opts.Timeout != 0 {
		t.Errorf("default Timeout should be 0, got %v", opts.Timeout)
	}
	if opts.Verbose {
		t.Error("default Verbose should be false, got true")
	}
	if opts.Clean {
		t.Error("default Clean should be false, got true")
	}
	if opts.Jobs != 0 {
		t.Errorf("default Jobs should be 0, got %d", opts.Jobs)
	}

	// Test setting values
	opts.Timeout = 5 * time.Minute
	opts.Verbose = true
	opts.Clean = true
	opts.Jobs = 4

	if opts.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want 5m", opts.Timeout)
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
	if !opts.Clean {
		t.Error("Clean should be true")
	}
	if opts.Jobs != 4 {
		t.Errorf("Jobs = %d, want 4", opts.Jobs)
	}
}

func TestTargetType(t *testing.T) {
	// Verify that TargetType constants are distinct
	types := map[TargetType]bool{
		TargetTypeBinary:  true,
		TargetTypeLibrary: true,
		TargetTypeTest:    true,
		TargetTypeUnknown: true,
	}

	if len(types) != 4 {
		t.Error("TargetType constants should be distinct")
	}

	// Verify string values
	if TargetTypeBinary != "binary" {
		t.Errorf("TargetTypeBinary = %q, want \"binary\"", TargetTypeBinary)
	}
	if TargetTypeLibrary != "library" {
		t.Errorf("TargetTypeLibrary = %q, want \"library\"", TargetTypeLibrary)
	}
	if TargetTypeTest != "test" {
		t.Errorf("TargetTypeTest = %q, want \"test\"", TargetTypeTest)
	}
	if TargetTypeUnknown != "unknown" {
		t.Errorf("TargetTypeUnknown = %q, want \"unknown\"", TargetTypeUnknown)
	}
}

func TestTarget(t *testing.T) {
	// Test Target struct can be created and fields accessed
	target := Target{
		ID:           "//cmd/app:app",
		Name:         "app",
		Type:         TargetTypeBinary,
		Description:  "Main application",
		Dependencies: []string{"//lib:lib1", "//lib:lib2"},
		Tags:         []string{"prod", "release"},
	}

	if target.ID != "//cmd/app:app" {
		t.Errorf("ID = %q, want \"//cmd/app:app\"", target.ID)
	}
	if target.Name != "app" {
		t.Errorf("Name = %q, want \"app\"", target.Name)
	}
	if target.Type != TargetTypeBinary {
		t.Errorf("Type = %q, want %q", target.Type, TargetTypeBinary)
	}
	if len(target.Dependencies) != 2 {
		t.Errorf("len(Dependencies) = %d, want 2", len(target.Dependencies))
	}
	if len(target.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want 2", len(target.Tags))
	}
}

func TestResult(t *testing.T) {
	// Test Result struct with success case
	now := time.Now()
	result := Result{
		Success:      true,
		Duration:     5 * time.Second,
		CacheHits:    10,
		CacheMisses:  2,
		TargetsBuilt: 12,
		Output:       "Build successful",
		StartTime:    now,
		EndTime:      now.Add(5 * time.Second),
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", result.Duration)
	}
	if result.TargetsBuilt != 12 {
		t.Errorf("TargetsBuilt = %d, want 12", result.TargetsBuilt)
	}

	// Test cache hit rate calculation
	rate := result.CacheHitRate()
	if rate < 83.0 || rate > 84.0 {
		t.Errorf("CacheHitRate() = %v, want ~83.33%%", rate)
	}

	// Test Result with failure
	failResult := Result{
		Success:      false,
		Duration:     1 * time.Second,
		Output:       "Build failed: compilation error",
		ErrorMessage: "compilation error in main.go:42",
	}

	if failResult.Success {
		t.Error("Success should be false for failed build")
	}
	if failResult.ErrorMessage == "" {
		t.Error("ErrorMessage should not be empty for failed build")
	}
}
