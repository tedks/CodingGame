package build

import (
	"testing"
)

// Fuzz tests use Go 1.18+ native fuzzing to discover edge cases.
// Run with: go test -fuzz=FuzzBazelParseQueryOutput -fuzztime=30s

func FuzzBazelParseQueryOutput(f *testing.F) {
	// Seed corpus with valid examples
	f.Add("go_binary rule //cmd/app:app\n")
	f.Add("go_library rule //lib:lib\ngo_test rule //lib:lib_test\n")
	f.Add("cc_binary rule //src:main\n")
	f.Add("java_test rule //tests:unit_tests\n")
	f.Add("proto_library rule //proto:messages\n")

	// Seed corpus with edge cases
	f.Add("")
	f.Add("\n")
	f.Add("   \n")
	f.Add("not a valid line")
	f.Add("go_binary rule //missing_colon\n")
	f.Add("rule go_binary //reversed:order\n")
	f.Add(string([]byte{0x00, 0x01, 0x02}))

	adapter := NewBazelAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		// Test that parseQueryOutput doesn't panic
		targets, err := adapter.parseQueryOutput(input)

		// Verify invariants on returned targets
		if err == nil {
			for i, target := range targets {
				// Every returned target should have a valid type
				switch target.Type {
				case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
					// OK
				default:
					t.Errorf("target %d has invalid type: %q", i, target.Type)
				}
			}
		}
	})
}

func FuzzBazelExtractMetrics(f *testing.F) {
	// Seed corpus with valid Bazel output examples
	f.Add("INFO: Build completed successfully, 42 total actions\n")
	f.Add("INFO: Build completed successfully, 100 total actions\nINFO: 75 remote cache hits\nINFO: 25 processes\n")
	f.Add("INFO: Analyzed target //cmd/app:app\nINFO: Build completed, 1 total action\n")
	f.Add("")

	// Edge cases
	f.Add("999999999999999999999 total actions")
	f.Add("-1 total actions")
	f.Add("0 total actions")
	f.Add("3.14159 total actions")
	f.Add(string([]byte{0x00}))

	adapter := NewBazelAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		result := &Result{}

		// Test that extractMetrics doesn't panic
		adapter.extractMetrics(input, result)

		// Verify invariants
		if result.TargetsBuilt < 0 {
			t.Errorf("TargetsBuilt is negative: %d", result.TargetsBuilt)
		}
		if result.CacheHits < 0 {
			t.Errorf("CacheHits is negative: %d", result.CacheHits)
		}
		if result.CacheMisses < 0 {
			t.Errorf("CacheMisses is negative: %d", result.CacheMisses)
		}

		// CacheHitRate should be in bounds
		rate := result.CacheHitRate()
		if rate < 0 || rate > 100 {
			// Check for NaN
			if rate != rate {
				t.Errorf("CacheHitRate is NaN")
			} else {
				t.Errorf("CacheHitRate out of bounds: %f", rate)
			}
		}
	})
}

func FuzzCargoParseMetadataOutput(f *testing.F) {
	// Seed corpus with valid cargo metadata output patterns
	f.Add(`{"packages":[{"name":"myapp","targets":[{"kind":["bin"],"name":"myapp"}]}]}`)
	f.Add(`{"targets":[{"kind":["lib"],"name":"mylib"},{"kind":["test"],"name":"mytest"}]}`)
	f.Add(`"name": "test"` + "\n" + `"kind": ["bin"]`)

	// Edge cases
	f.Add("")
	f.Add("{}")
	f.Add(`{"name": "`)         // Truncated JSON
	f.Add(`{"kind": ["bin"]}`)  // Missing name
	f.Add(`{"name": "foo"}`)    // Missing kind
	f.Add(string([]byte{0x00}))
	f.Add(`{"kind": ["unknown_kind"], "name": "test"}`)

	adapter := NewCargoAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		// Test that parseMetadataOutput doesn't panic
		targets, err := adapter.parseMetadataOutput(input)

		// Verify invariants on returned targets
		if err == nil {
			for i, target := range targets {
				// Every returned target should have a valid type
				switch target.Type {
				case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
					// OK
				default:
					t.Errorf("target %d has invalid type: %q", i, target.Type)
				}
			}
		}
	})
}

func FuzzCargoExtractMetrics(f *testing.F) {
	// Seed corpus with valid Cargo output examples
	f.Add("Compiling myapp v0.1.0\nFinished dev [unoptimized + debuginfo] target(s) in 1.23s\n")
	f.Add("Fresh mylib v0.1.0\nFresh myapp v0.1.0\nFinished in 0.05s\n")
	f.Add("Compiling a v1.0\nCompiling b v1.0\nCompiling c v1.0\nFinished in 5.0s\n")
	f.Add("")

	// Edge cases
	f.Add("Compiling")                       // Incomplete
	f.Add("Fresh")                           // Incomplete
	f.Add("Finished in 999999999999999999s") // Large number
	f.Add("Finished in -1s")                 // Negative
	f.Add(string([]byte{0x00}))

	adapter := NewCargoAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		result := &Result{}

		// Test that extractMetrics doesn't panic
		adapter.extractMetrics(input, result)

		// Verify invariants
		if result.TargetsBuilt < 0 {
			t.Errorf("TargetsBuilt is negative: %d", result.TargetsBuilt)
		}
		if result.CacheHits < 0 {
			t.Errorf("CacheHits is negative: %d", result.CacheHits)
		}
		if result.CacheMisses < 0 {
			t.Errorf("CacheMisses is negative: %d", result.CacheMisses)
		}

		// CacheHitRate should be in bounds
		rate := result.CacheHitRate()
		if rate < 0 || rate > 100 {
			if rate != rate {
				t.Errorf("CacheHitRate is NaN")
			} else {
				t.Errorf("CacheHitRate out of bounds: %f", rate)
			}
		}
	})
}

func FuzzBazelInferTargetType(f *testing.F) {
	// Seed corpus
	f.Add("go_test")
	f.Add("go_binary")
	f.Add("go_library")
	f.Add("java_test")
	f.Add("cc_binary")
	f.Add("proto_library")
	f.Add("filegroup")
	f.Add("genrule")
	f.Add("")
	f.Add("TEST")
	f.Add("BINARY")
	f.Add(string([]byte{0x00}))

	adapter := NewBazelAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		// Test that inferTargetType doesn't panic
		result := adapter.inferTargetType(input)

		// Verify result is a valid type
		switch result {
		case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
			// OK
		default:
			t.Errorf("inferTargetType returned invalid type: %q", result)
		}

		// Verify determinism
		result2 := adapter.inferTargetType(input)
		if result != result2 {
			t.Errorf("inferTargetType not deterministic: %q != %q", result, result2)
		}
	})
}

func FuzzCargoInferTargetType(f *testing.F) {
	// Seed corpus
	f.Add("bin")
	f.Add("lib")
	f.Add("test")
	f.Add("bench")
	f.Add("proc-macro")
	f.Add("cdylib")
	f.Add("staticlib")
	f.Add("rlib")
	f.Add("dylib")
	f.Add("")
	f.Add("BIN")
	f.Add("unknown")
	f.Add(string([]byte{0x00}))

	adapter := NewCargoAdapter()

	f.Fuzz(func(t *testing.T, input string) {
		// Test that inferTargetType doesn't panic
		result := adapter.inferTargetType(input)

		// Verify result is a valid type
		switch result {
		case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
			// OK
		default:
			t.Errorf("inferTargetType returned invalid type: %q", result)
		}

		// Verify determinism
		result2 := adapter.inferTargetType(input)
		if result != result2 {
			t.Errorf("inferTargetType not deterministic: %q != %q", result, result2)
		}
	})
}

func FuzzNpmInferTargetType(f *testing.F) {
	// Seed corpus (name, command pairs)
	f.Add("test", "jest")
	f.Add("build", "webpack")
	f.Add("start", "node index.js")
	f.Add("lint", "eslint .")
	f.Add("", "")
	f.Add("test", "")
	f.Add("", "jest")
	f.Add(string([]byte{0x00}), string([]byte{0x00}))

	adapter := NewNpmAdapter()

	f.Fuzz(func(t *testing.T, name, command string) {
		// Test that inferTargetType doesn't panic
		result := adapter.inferTargetType(name, command)

		// Verify result is a valid type
		switch result {
		case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
			// OK
		default:
			t.Errorf("inferTargetType returned invalid type: %q", result)
		}

		// Verify determinism
		result2 := adapter.inferTargetType(name, command)
		if result != result2 {
			t.Errorf("inferTargetType not deterministic: %q != %q", result, result2)
		}
	})
}

func FuzzResultCacheHitRate(f *testing.F) {
	// Seed corpus with various hit/miss combinations
	f.Add(int64(0), int64(0))
	f.Add(int64(100), int64(0))
	f.Add(int64(0), int64(100))
	f.Add(int64(50), int64(50))
	f.Add(int64(1), int64(1))
	f.Add(int64(9223372036854775807), int64(0)) // max int64
	f.Add(int64(0), int64(9223372036854775807))
	f.Add(int64(9223372036854775807), int64(9223372036854775807))

	f.Fuzz(func(t *testing.T, hits, misses int64) {
		// Skip negative inputs since our struct uses int64 but negative
		// values aren't semantically valid
		if hits < 0 || misses < 0 {
			return
		}

		result := &Result{
			CacheHits:   hits,
			CacheMisses: misses,
		}

		// Test that CacheHitRate doesn't panic
		rate := result.CacheHitRate()

		// Verify bounds
		if rate < 0 {
			t.Errorf("CacheHitRate < 0: %f (hits=%d, misses=%d)", rate, hits, misses)
		}
		if rate > 100 {
			t.Errorf("CacheHitRate > 100: %f (hits=%d, misses=%d)", rate, hits, misses)
		}

		// Check for NaN
		if rate != rate {
			t.Errorf("CacheHitRate is NaN (hits=%d, misses=%d)", hits, misses)
		}

		// Verify mathematical correctness for non-zero totals
		total := hits + misses
		if total > 0 && total > hits { // Check for overflow
			expected := float64(hits) / float64(total) * 100.0
			diff := rate - expected
			if diff > 0.0001 || diff < -0.0001 {
				t.Errorf("CacheHitRate incorrect: got %f, expected %f", rate, expected)
			}
		}
	})
}

// Fuzz test for the Registry.Detect method
func FuzzRegistryGetAdapter(f *testing.F) {
	// Seed with valid adapter names
	f.Add("bazel")
	f.Add("cargo")
	f.Add("npm")
	f.Add("")
	f.Add("nonexistent")
	f.Add(string([]byte{0x00}))
	f.Add("BAZEL") // Wrong case

	registry := NewRegistry()
	for _, adapter := range allAdapters() {
		registry.Register(adapter)
	}

	f.Fuzz(func(t *testing.T, name string) {
		// Test that GetAdapter doesn't panic
		adapter, err := registry.GetAdapter(name)

		// If found, verify it's the right adapter
		if err == nil {
			if adapter == nil {
				t.Error("GetAdapter returned nil adapter with nil error")
			} else if adapter.Name() != name {
				t.Errorf("GetAdapter(%q) returned adapter with name %q", name, adapter.Name())
			}
		}
	})
}
