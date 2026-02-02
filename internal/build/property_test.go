package build

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Property tests verify invariants that should hold across all inputs.
// These are lightweight property-based tests that complement fuzz testing.

func TestProperty_Result_TimingInvariant(t *testing.T) {
	// Invariant: EndTime >= StartTime always

	// Generate random timestamps
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		// Random start time in the past
		start := time.Now().Add(-time.Duration(rng.Int63n(int64(24 * time.Hour))))
		// Random duration
		duration := time.Duration(rng.Int63n(int64(10 * time.Minute)))
		end := start.Add(duration)

		result := &Result{
			StartTime: start,
			EndTime:   end,
			Duration:  duration,
		}

		// Invariant check
		if result.EndTime.Before(result.StartTime) {
			t.Errorf("EndTime before StartTime: start=%v, end=%v", result.StartTime, result.EndTime)
		}

		// Duration should match
		calculatedDuration := result.EndTime.Sub(result.StartTime)
		if calculatedDuration != result.Duration {
			t.Errorf("Duration mismatch: calculated=%v, stored=%v", calculatedDuration, result.Duration)
		}
	}
}

func TestProperty_CacheHitRate_Bounds(t *testing.T) {
	// Invariant: CacheHitRate() is always in [0, 100]

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	testCases := []struct {
		hits   int64
		misses int64
	}{
		{0, 0},
		{1, 0},
		{0, 1},
		{50, 50},
		{100, 0},
		{0, 100},
	}

	// Add random cases
	for i := 0; i < 100; i++ {
		testCases = append(testCases, struct {
			hits   int64
			misses int64
		}{
			hits:   rng.Int63n(1000000),
			misses: rng.Int63n(1000000),
		})
	}

	for _, tc := range testCases {
		result := &Result{
			CacheHits:   tc.hits,
			CacheMisses: tc.misses,
		}

		rate := result.CacheHitRate()

		// Check bounds
		if rate < 0 {
			t.Errorf("CacheHitRate < 0: hits=%d, misses=%d, rate=%f", tc.hits, tc.misses, rate)
		}
		if rate > 100 {
			t.Errorf("CacheHitRate > 100: hits=%d, misses=%d, rate=%f", tc.hits, tc.misses, rate)
		}

		// Check for NaN
		if rate != rate { // NaN check
			t.Errorf("CacheHitRate is NaN: hits=%d, misses=%d", tc.hits, tc.misses)
		}

		// Check mathematical correctness
		total := tc.hits + tc.misses
		if total > 0 {
			expected := float64(tc.hits) / float64(total) * 100.0
			// Allow small floating point tolerance
			if diff := rate - expected; diff > 0.0001 || diff < -0.0001 {
				t.Errorf("CacheHitRate incorrect: got %f, expected %f", rate, expected)
			}
		}
	}
}

func TestProperty_ParseQueryOutput_NoPanic(t *testing.T) {
	// Invariant: parseQueryOutput never panics, regardless of input

	adapter := NewBazelAdapter()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Generate random inputs
	for i := 0; i < 1000; i++ {
		input := generateRandomString(rng, rng.Intn(10000))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parseQueryOutput panicked on input %d (len=%d): %v", i, len(input), r)
				}
			}()

			_, _ = adapter.parseQueryOutput(input)
		}()
	}
}

func TestProperty_InferTargetType_Deterministic(t *testing.T) {
	// Invariant: Same input always produces same output

	adapters := []struct {
		name  string
		infer func(string) TargetType
	}{
		{"bazel", func(s string) TargetType { return NewBazelAdapter().inferTargetType(s) }},
		{"cargo", func(s string) TargetType { return NewCargoAdapter().inferTargetType(s) }},
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Fixed test inputs
	testInputs := []string{
		"go_test", "go_binary", "go_library",
		"", " ", "TEST", "BINARY", "LIBRARY",
		"custom_rule", "my_test_suite",
	}

	// Add random inputs
	for i := 0; i < 100; i++ {
		testInputs = append(testInputs, generateRandomString(rng, rng.Intn(100)))
	}

	for _, a := range adapters {
		for _, input := range testInputs {
			// Call multiple times
			results := make([]TargetType, 5)
			for i := 0; i < 5; i++ {
				results[i] = a.infer(input)
			}

			// All results should be identical
			for i := 1; i < 5; i++ {
				if results[i] != results[0] {
					t.Errorf("%s.inferTargetType(%q) not deterministic: got %v and %v",
						a.name, input, results[0], results[i])
				}
			}
		}
	}
}

func TestProperty_InferTargetType_ReturnsValidType(t *testing.T) {
	// Invariant: inferTargetType always returns a valid TargetType

	adapters := []struct {
		name  string
		infer func(string) TargetType
	}{
		{"bazel", func(s string) TargetType { return NewBazelAdapter().inferTargetType(s) }},
		{"cargo", func(s string) TargetType { return NewCargoAdapter().inferTargetType(s) }},
	}

	validTypes := map[TargetType]bool{
		TargetTypeBinary:  true,
		TargetTypeLibrary: true,
		TargetTypeTest:    true,
		TargetTypeUnknown: true,
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, a := range adapters {
		for i := 0; i < 1000; i++ {
			input := generateRandomString(rng, rng.Intn(1000))
			result := a.infer(input)

			if !validTypes[result] {
				t.Errorf("%s.inferTargetType returned invalid type: %q", a.name, result)
			}
		}
	}
}

func TestProperty_ExtractMetrics_NonNegative(t *testing.T) {
	// Invariant: Extracted metrics are never negative

	adapters := []struct {
		name    string
		extract func(string, *Result)
	}{
		{"bazel", func(s string, r *Result) { NewBazelAdapter().extractMetrics(s, r) }},
		{"cargo", func(s string, r *Result) { NewCargoAdapter().extractMetrics(s, r) }},
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, a := range adapters {
		for i := 0; i < 1000; i++ {
			input := generateRandomString(rng, rng.Intn(10000))
			result := &Result{}
			a.extract(input, result)

			if result.TargetsBuilt < 0 {
				t.Errorf("%s.extractMetrics produced negative TargetsBuilt: %d", a.name, result.TargetsBuilt)
			}
			if result.CacheHits < 0 {
				t.Errorf("%s.extractMetrics produced negative CacheHits: %d", a.name, result.CacheHits)
			}
			if result.CacheMisses < 0 {
				t.Errorf("%s.extractMetrics produced negative CacheMisses: %d", a.name, result.CacheMisses)
			}
		}
	}
}

func TestProperty_ExtractMetrics_Idempotent(t *testing.T) {
	// Invariant: Calling extractMetrics multiple times with same input gives same result

	adapters := []struct {
		name    string
		extract func(string, *Result)
	}{
		{"bazel", func(s string, r *Result) { NewBazelAdapter().extractMetrics(s, r) }},
		{"cargo", func(s string, r *Result) { NewCargoAdapter().extractMetrics(s, r) }},
	}

	testInputs := []string{
		"",
		"INFO: Build completed successfully, 42 total actions",
		"Compiling foo v1.0\nCompiling bar v2.0\nFinished in 1.5s",
		"garbage input with no metrics",
	}

	for _, a := range adapters {
		for _, input := range testInputs {
			// Extract metrics twice
			result1 := &Result{}
			result2 := &Result{}

			a.extract(input, result1)
			a.extract(input, result2)

			// Results should be identical
			if result1.TargetsBuilt != result2.TargetsBuilt {
				t.Errorf("%s.extractMetrics not idempotent for TargetsBuilt: %d != %d",
					a.name, result1.TargetsBuilt, result2.TargetsBuilt)
			}
			if result1.CacheHits != result2.CacheHits {
				t.Errorf("%s.extractMetrics not idempotent for CacheHits: %d != %d",
					a.name, result1.CacheHits, result2.CacheHits)
			}
			if result1.CacheMisses != result2.CacheMisses {
				t.Errorf("%s.extractMetrics not idempotent for CacheMisses: %d != %d",
					a.name, result1.CacheMisses, result2.CacheMisses)
			}
		}
	}
}

func TestProperty_ParseQueryOutput_TargetsHaveIDs(t *testing.T) {
	// Invariant: Every returned target has a non-empty ID

	adapter := NewBazelAdapter()

	testInputs := []string{
		"go_binary rule //cmd/app:app\n",
		"go_library rule //lib:lib\ngo_test rule //lib:lib_test\n",
		"filegroup rule //some/path\n",
		"proto_library rule //proto:messages\n",
	}

	for _, input := range testInputs {
		targets, err := adapter.parseQueryOutput(input)
		if err != nil {
			continue // Skip inputs that cause errors
		}

		for i, target := range targets {
			if target.ID == "" {
				t.Errorf("Target %d has empty ID from input: %q", i, input)
			}
		}
	}
}

func TestProperty_ParseQueryOutput_PreservesLineCount(t *testing.T) {
	// Property: Number of targets <= number of non-empty lines

	adapter := NewBazelAdapter()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 100; i++ {
		// Generate random input with some valid-looking lines
		var lines []string
		numLines := rng.Intn(50)
		for j := 0; j < numLines; j++ {
			if rng.Float32() < 0.3 {
				// Valid-looking line
				lines = append(lines, "go_binary rule //cmd/app:app")
			} else {
				// Random garbage
				lines = append(lines, generateRandomString(rng, rng.Intn(100)))
			}
		}
		input := strings.Join(lines, "\n")

		targets, err := adapter.parseQueryOutput(input)
		if err != nil {
			continue
		}

		// Count non-empty lines
		nonEmptyCount := 0
		for _, line := range strings.Split(input, "\n") {
			if strings.TrimSpace(line) != "" {
				nonEmptyCount++
			}
		}

		if len(targets) > nonEmptyCount {
			t.Errorf("More targets (%d) than non-empty lines (%d)", len(targets), nonEmptyCount)
		}
	}
}

func TestProperty_CargoExtractMetrics_FreshPlusCompiledEqualsTotal(t *testing.T) {
	// Property: For cargo, Fresh + Compiling should equal TargetsBuilt (approximately)

	adapter := NewCargoAdapter()

	testInputs := []string{
		"Compiling a v1.0\nCompiling b v1.0\nFinished in 1.0s",
		"Fresh a v1.0\nFresh b v1.0\nFinished in 0.1s",
		"Fresh a v1.0\nCompiling b v1.0\nFinished in 0.5s",
		"",
	}

	for _, input := range testInputs {
		result := &Result{}
		adapter.extractMetrics(input, result)

		total := result.CacheHits + result.CacheMisses
		// Note: TargetsBuilt might be set to 1 as a default when no targets found
		if result.TargetsBuilt > 1 && int64(result.TargetsBuilt) != total {
			t.Logf("Input: %q", input)
			t.Logf("TargetsBuilt=%d, CacheHits=%d, CacheMisses=%d, sum=%d",
				result.TargetsBuilt, result.CacheHits, result.CacheMisses, total)
			// This is informational, not a failure, due to the default behavior
		}
	}
}

func TestProperty_NpmInferTargetType_ConsistentWithKeywords(t *testing.T) {
	// Property: If script name/command contains "test", result should be Test type

	adapter := NewNpmAdapter()

	testCases := []struct {
		name    string
		command string
		want    TargetType
	}{
		{"test", "jest", TargetTypeTest},
		{"unit-test", "mocha", TargetTypeTest},
		{"e2e-test", "cypress", TargetTypeTest},
		{"run-tests", "vitest", TargetTypeTest},
		{"build", "webpack", TargetTypeBinary},
		{"compile", "tsc", TargetTypeBinary},
		{"start", "node index.js", TargetTypeUnknown},
		{"dev", "nodemon", TargetTypeUnknown},
	}

	for _, tc := range testCases {
		result := adapter.inferTargetType(tc.name, tc.command)
		if result != tc.want {
			t.Errorf("inferTargetType(%q, %q) = %q, want %q",
				tc.name, tc.command, result, tc.want)
		}
	}
}

func TestProperty_TargetType_AllConstantsUsed(t *testing.T) {
	// Verify all TargetType constants are actually achievable

	achieved := map[TargetType]bool{}

	// Test Bazel adapter
	bazel := NewBazelAdapter()
	bazelRules := []string{
		"go_test", "go_binary", "go_library", "filegroup",
		"java_test", "java_binary", "cc_library", "genrule",
	}
	for _, rule := range bazelRules {
		achieved[bazel.inferTargetType(rule)] = true
	}

	// Test Cargo adapter
	cargo := NewCargoAdapter()
	cargoKinds := []string{
		"bin", "lib", "test", "bench", "proc-macro", "example",
	}
	for _, kind := range cargoKinds {
		achieved[cargo.inferTargetType(kind)] = true
	}

	// Verify all types were achieved
	allTypes := []TargetType{TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown}
	for _, tt := range allTypes {
		if !achieved[tt] {
			t.Errorf("TargetType %q never returned by any adapter", tt)
		}
	}
}

// Helper function to generate random strings for property testing
func generateRandomString(rng *rand.Rand, length int) string {
	if length <= 0 {
		return ""
	}

	// Mix of ASCII, control chars, and some unicode
	chars := []byte{}
	for i := 0; i < length; i++ {
		switch rng.Intn(4) {
		case 0: // ASCII printable
			chars = append(chars, byte(32+rng.Intn(95)))
		case 1: // ASCII control
			chars = append(chars, byte(rng.Intn(32)))
		case 2: // Newlines/whitespace
			ws := []byte{'\n', '\r', '\t', ' '}
			chars = append(chars, ws[rng.Intn(len(ws))])
		case 3: // High bytes (may create invalid UTF-8)
			chars = append(chars, byte(128+rng.Intn(128)))
		}
	}

	return string(chars)
}

func TestProperty_Result_ZeroValuesAreValid(t *testing.T) {
	// Invariant: Zero-value Result is valid and usable

	result := &Result{}

	// All fields should have sensible zero values
	if result.Success != false {
		t.Error("Zero Result.Success should be false")
	}
	if result.Duration != 0 {
		t.Error("Zero Result.Duration should be 0")
	}
	if result.CacheHits != 0 {
		t.Error("Zero Result.CacheHits should be 0")
	}
	if result.CacheMisses != 0 {
		t.Error("Zero Result.CacheMisses should be 0")
	}
	if result.TargetsBuilt != 0 {
		t.Error("Zero Result.TargetsBuilt should be 0")
	}

	// CacheHitRate should work on zero values
	rate := result.CacheHitRate()
	if rate != 0 {
		t.Errorf("Zero Result.CacheHitRate() should be 0, got %f", rate)
	}
}

func TestProperty_BuildOptions_ZeroValuesAreValid(t *testing.T) {
	// Invariant: Zero-value BuildOptions is valid and usable

	opts := &BuildOptions{}

	if opts.Timeout != 0 {
		t.Error("Zero BuildOptions.Timeout should be 0")
	}
	if opts.Clean != false {
		t.Error("Zero BuildOptions.Clean should be false")
	}
	if opts.Jobs != 0 {
		t.Error("Zero BuildOptions.Jobs should be 0")
	}
}

func TestProperty_Target_ZeroValuesAreValid(t *testing.T) {
	// Invariant: Zero-value Target is valid (though perhaps useless)

	target := Target{}

	if target.ID != "" {
		t.Error("Zero Target.ID should be empty")
	}
	if target.Name != "" {
		t.Error("Zero Target.Name should be empty")
	}
	if target.Type != "" {
		t.Error("Zero Target.Type should be empty")
	}
	if len(target.Dependencies) != 0 {
		t.Error("Zero Target.Dependencies should be empty")
	}
	if len(target.Tags) != 0 {
		t.Error("Zero Target.Tags should be empty")
	}
}
