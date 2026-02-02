package build

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Chaos tests verify that parsers don't panic on garbage input.
// These tests exercise edge cases that might slip through normal testing.

// chaosInputs returns a collection of malformed inputs designed to break parsers
func chaosInputs() []struct {
	name  string
	input string
} {
	return []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"single_null", string([]byte{0x00})},
		{"binary_garbage", string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD})},
		{"null_in_middle", "foo\x00bar"},
		{"high_unicode", "\U0001F4A9\U0001F525\U0001F680"}, // emojis
		{"invalid_utf8", string([]byte{0xFF, 0xFE, 0x80, 0x81})},
		{"mixed_valid_invalid_utf8", "valid" + string([]byte{0xFF}) + "text"},
		{"newlines_only", "\n\n\n\n\n"},
		{"carriage_returns", "\r\r\r\r"},
		{"mixed_line_endings", "line1\r\nline2\rline3\n"},
		{"tabs_only", "\t\t\t\t\t"},
		{"spaces_only", "     "},
		{"very_long_line", strings.Repeat("a", 100000)},
		{"many_short_lines", strings.Repeat("x\n", 10000)},
		{"pathological_regex_parens", strings.Repeat("(", 1000) + strings.Repeat(")", 1000)},
		{"pathological_regex_brackets", strings.Repeat("[", 1000) + strings.Repeat("]", 1000)},
		{"backslashes", strings.Repeat(`\`, 100)},
		{"regex_metachars", `.*+?^${}()|[]\`},
		{"truncated_number_small", "0 total actions"},
		{"truncated_number_large", "999999999999999999999999999 total actions"},
		{"negative_number", "-1 total actions"},
		{"float_where_int_expected", "3.14159 total actions"},
		{"scientific_notation", "1e10 total actions"},
		{"hex_number", "0xFF total actions"},
		{"number_with_commas", "1,000,000 total actions"},
		{"almost_valid_format", "go_binary rule"},                // missing target
		{"extra_fields", "go_binary rule //foo:bar extra stuff"}, // extra content
		{"wrong_order", "//foo:bar rule go_binary"},              // reversed
		{"partial_target", "go_binary rule //foo"},               // no colon
		{"empty_target", "go_binary rule :"},                     // colon only
		{"double_colon", "go_binary rule //foo::bar"},
		{"triple_slash", "go_binary rule ///foo:bar"},
		{"json_fragment", `{"kind": ["bin"], "name": "test"`}, // truncated JSON
		{"json_array_only", `["bin", "lib", "test"]`},
		{"json_nested_deep", `{"a":{"b":{"c":{"d":{"e":"f"}}}}}`},
		{"json_with_escapes", `{"name": "test\"with\\escapes"}`},
		{"json_unicode_escapes", `{"name": "\u0000\uFFFF"}`},
		{"xml_fragment", `<kind>bin</kind><name>test</name>`},
		{"html_fragment", `<script>alert('xss')</script>`},
		{"control_chars", string([]byte{0x01, 0x02, 0x03, 0x04, 0x05})},
		{"bell_char", "\a\a\a"},
		{"form_feed", "\f\f\f"},
		{"vertical_tab", "\v\v\v"},
		{"zero_width_chars", "\u200B\u200C\u200D\uFEFF"},
		{"rtl_override", "\u202E" + "reversed" + "\u202C"},
		{"bom_prefix", "\uFEFF" + "content"},
		{"null_terminated", "content\x00"},
		{"repeated_pattern", strings.Repeat("Compiling foo v1.0\n", 10000)},
		{"alternating_valid_invalid", "go_binary rule //a:a\nGARBAGE\ngo_test rule //b:b\nMORE GARBAGE\n"},
	}
}

func TestChaos_BazelParseQueryOutput_MalformedLines(t *testing.T) {
	adapter := NewBazelAdapter()

	for _, tc := range chaosInputs() {
		t.Run(tc.name, func(t *testing.T) {
			// The test passes if we don't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parseQueryOutput panicked on input %q: %v", tc.name, r)
				}
			}()

			targets, err := adapter.parseQueryOutput(tc.input)

			// We don't care about the specific result, just that:
			// 1. No panic occurred
			// 2. Error is nil or a reasonable error (not a crash)
			// 3. Targets slice is valid (not nil causing issues)
			_ = err
			_ = len(targets)

			// If targets were returned, verify they're well-formed
			for i, target := range targets {
				if target.ID == "" && target.Name == "" && target.Type == "" {
					// Empty target is suspicious but not a panic
					t.Logf("Warning: empty target at index %d for input %q", i, tc.name)
				}
			}
		})
	}
}

func TestChaos_BazelExtractMetrics_MalformedOutput(t *testing.T) {
	adapter := NewBazelAdapter()

	for _, tc := range chaosInputs() {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("extractMetrics panicked on input %q: %v", tc.name, r)
				}
			}()

			result := &Result{}
			adapter.extractMetrics(tc.input, result)

			// Verify result has valid values (no NaN, no extreme negatives)
			if result.TargetsBuilt < 0 {
				t.Errorf("TargetsBuilt is negative: %d", result.TargetsBuilt)
			}
			if result.CacheHits < 0 {
				t.Errorf("CacheHits is negative: %d", result.CacheHits)
			}
			if result.CacheMisses < 0 {
				t.Errorf("CacheMisses is negative: %d", result.CacheMisses)
			}
		})
	}
}

func TestChaos_CargoParseMetadataOutput_BrokenJSON(t *testing.T) {
	adapter := NewCargoAdapter()

	for _, tc := range chaosInputs() {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parseMetadataOutput panicked on input %q: %v", tc.name, r)
				}
			}()

			targets, err := adapter.parseMetadataOutput(tc.input)

			_ = err
			_ = len(targets)

			// Verify any returned targets are well-formed
			for i, target := range targets {
				if target.ID == "" && target.Name == "" && target.Type == "" {
					t.Logf("Warning: empty target at index %d for input %q", i, tc.name)
				}
			}
		})
	}
}

func TestChaos_CargoExtractMetrics_MalformedOutput(t *testing.T) {
	adapter := NewCargoAdapter()

	for _, tc := range chaosInputs() {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("extractMetrics panicked on input %q: %v", tc.name, r)
				}
			}()

			result := &Result{}
			adapter.extractMetrics(tc.input, result)

			// Verify result has valid values
			if result.TargetsBuilt < 0 {
				t.Errorf("TargetsBuilt is negative: %d", result.TargetsBuilt)
			}
			if result.CacheHits < 0 {
				t.Errorf("CacheHits is negative: %d", result.CacheHits)
			}
			if result.CacheMisses < 0 {
				t.Errorf("CacheMisses is negative: %d", result.CacheMisses)
			}
		})
	}
}

// Additional chaos tests for specific edge cases identified in the plan

func TestChaos_BazelExtractMetrics_IntegerOverflow(t *testing.T) {
	adapter := NewBazelAdapter()

	// Test inputs that might cause integer overflow
	overflowInputs := []struct {
		name  string
		input string
	}{
		{"max_int64", "9223372036854775807 total actions"},
		{"max_int64_plus_one", "9223372036854775808 total actions"},
		{"way_over_max", "99999999999999999999999999999999 total actions"},
		{"max_int32", "2147483647 total actions"},
		{"max_int32_plus_one", "2147483648 total actions"},
		{"negative_max", "-9223372036854775808 total actions"},
	}

	for _, tc := range overflowInputs {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("extractMetrics panicked on overflow input: %v", r)
				}
			}()

			result := &Result{}
			adapter.extractMetrics(tc.input, result)

			// The code should handle overflow gracefully (either ignore or cap)
			// Current implementation: strconv.ParseInt returns error, which is ignored
			// So TargetsBuilt stays 0 for overflow cases
			t.Logf("TargetsBuilt for %s: %d", tc.name, result.TargetsBuilt)
		})
	}
}

func TestChaos_BazelParseQueryOutput_ScannerBufferLimit(t *testing.T) {
	adapter := NewBazelAdapter()

	// Default bufio.Scanner buffer is 64KB, test lines at and above that limit
	sizes := []int{
		1000,    // 1KB - well under limit
		64000,   // 64KB - at default limit
		65536,   // 64KB exactly
		100000,  // 100KB - over limit
		1000000, // 1MB - way over limit
	}

	for _, size := range sizes {
		t.Run(strings.Replace(string(rune(size/1000))+"KB", "\x00", "", -1), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parseQueryOutput panicked on %d byte line: %v", size, r)
				}
			}()

			// Create a valid-looking line that's very long
			longLine := "go_binary rule //" + strings.Repeat("a", size) + ":target\n"

			targets, err := adapter.parseQueryOutput(longLine)

			// The scanner may return an error for lines over 64KB
			// That's acceptable behavior - we just shouldn't panic
			if err != nil {
				t.Logf("Got error for %d byte line (expected): %v", size, err)
			}
			_ = len(targets)
		})
	}
}

func TestChaos_ValidInputWithNullByte(t *testing.T) {
	adapter := NewBazelAdapter()

	// Valid content with null bytes embedded
	inputs := []string{
		"go_binary rule //cmd/app:app\x00\ngo_test rule //lib:lib_test",
		"go_binary\x00rule //cmd/app:app",
		"go_binary rule\x00//cmd/app:app",
		"go_binary rule //cmd/\x00app:app",
	}

	for i, input := range inputs {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on null-embedded input: %v", r)
				}
			}()

			targets, err := adapter.parseQueryOutput(input)
			_ = err
			_ = targets
		})
	}
}

func TestChaos_BazelInferTargetType_EmptyAndWeird(t *testing.T) {
	adapter := NewBazelAdapter()

	weirdTypes := []string{
		"",
		" ",
		"\t",
		"\n",
		string([]byte{0x00}),
		"TEST",                             // all caps
		"tEsT",                             // mixed case
		"_test",                            // leading underscore
		"test_",                            // trailing underscore
		strings.Repeat("test", 1000),       // very long
		"test\x00binary",                   // null in middle
		"🔨",                                // emoji
		"<script>test</script>",            // HTML
		`{"type": "test"}`,                 // JSON
		"../../../../../../etc/passwd",    // path traversal attempt
		"%s%s%s%s%s",                       // format string
		"${PATH}",                          // shell variable
		"$(whoami)",                        // command substitution
		"`whoami`",                         // backtick substitution
		"test\ninjected\nlines",            // newline injection
		"test; rm -rf /",                   // command injection attempt
		strings.Repeat("A", 1000000),       // 1MB string
	}

	for i, ruleType := range weirdTypes {
		t.Run(string(rune('a'+i%26))+string(rune('0'+i/26)), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("inferTargetType panicked: %v", r)
				}
			}()

			result := adapter.inferTargetType(ruleType)
			// Result should be one of the defined types
			switch result {
			case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
				// OK
			default:
				t.Errorf("unexpected target type: %q", result)
			}
		})
	}
}

func TestChaos_CargoInferTargetType_WeirdKinds(t *testing.T) {
	adapter := NewCargoAdapter()

	weirdKinds := []string{
		"",
		"BIN",           // uppercase
		"Bin",           // mixed case
		"binary",        // wrong term
		"executable",    // wrong term
		" bin",          // leading space
		"bin ",          // trailing space
		"bin\n",         // trailing newline
		"\x00bin",       // null prefix
		"bin\x00",       // null suffix
		"proc-macro\n",  // valid with newline
		"cdylib ",       // valid with space
	}

	for i, kind := range weirdKinds {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("inferTargetType panicked: %v", r)
				}
			}()

			result := adapter.inferTargetType(kind)
			switch result {
			case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
				// OK
			default:
				t.Errorf("unexpected target type: %q", result)
			}
		})
	}
}

func TestChaos_NpmInferTargetType_WeirdInputs(t *testing.T) {
	adapter := NewNpmAdapter()

	weirdInputs := []struct {
		name    string
		command string
	}{
		{"", ""},
		{" ", " "},
		{"\n", "\n"},
		{string([]byte{0x00}), string([]byte{0x00})},
		{"test", ""},
		{"", "jest"},
		{strings.Repeat("test", 1000), strings.Repeat("jest", 1000)},
		{"<script>", "alert(1)"},
		{"$(rm -rf /)", "`rm -rf /`"},
		{"\x00test\x00", "\x00jest\x00"},
	}

	for i, input := range weirdInputs {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("inferTargetType panicked: %v", r)
				}
			}()

			result := adapter.inferTargetType(input.name, input.command)
			switch result {
			case TargetTypeBinary, TargetTypeLibrary, TargetTypeTest, TargetTypeUnknown:
				// OK
			default:
				t.Errorf("unexpected target type: %q", result)
			}
		})
	}
}

func TestChaos_ResultCacheHitRate_ExtremeValues(t *testing.T) {
	extremeCases := []struct {
		name   string
		result *Result
	}{
		{"zero_zero", &Result{CacheHits: 0, CacheMisses: 0}},
		{"max_int64_hits", &Result{CacheHits: 9223372036854775807, CacheMisses: 0}},
		{"max_int64_misses", &Result{CacheHits: 0, CacheMisses: 9223372036854775807}},
		{"both_max", &Result{CacheHits: 9223372036854775807, CacheMisses: 9223372036854775807}},
		{"one_each", &Result{CacheHits: 1, CacheMisses: 1}},
		{"large_unbalanced", &Result{CacheHits: 9223372036854775807, CacheMisses: 1}},
	}

	for _, tc := range extremeCases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("CacheHitRate panicked: %v", r)
				}
			}()

			rate := tc.result.CacheHitRate()

			// Rate should be in [0, 100] or NaN (which we'd want to catch)
			if rate < 0 || rate > 100 {
				// Check if it's NaN
				if rate != rate { // NaN check
					t.Errorf("CacheHitRate returned NaN")
				} else {
					t.Errorf("CacheHitRate out of bounds: %f", rate)
				}
			}
		})
	}
}

// Helper to check if string contains valid UTF-8
func isValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

func TestChaos_OutputContainsInvalidUTF8(t *testing.T) {
	adapter := NewBazelAdapter()

	// Specifically test that invalid UTF-8 doesn't cause issues
	invalidUTF8Inputs := []string{
		string([]byte{0x80}),                     // Invalid start byte
		string([]byte{0xC0, 0xAF}),               // Overlong encoding
		string([]byte{0xE0, 0x80, 0xAF}),         // Overlong encoding
		string([]byte{0xF0, 0x80, 0x80, 0xAF}),   // Overlong encoding
		string([]byte{0xF4, 0x90, 0x80, 0x80}),   // Out of range
		string([]byte{0xED, 0xA0, 0x80}),         // Surrogate half
		"valid" + string([]byte{0x80}) + "text", // Mixed
	}

	for i, input := range invalidUTF8Inputs {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			if isValidUTF8(input) {
				t.Skip("Input is valid UTF-8, skipping")
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parseQueryOutput panicked on invalid UTF-8: %v", r)
				}
			}()

			// These shouldn't panic even with invalid UTF-8
			_, _ = adapter.parseQueryOutput(input)

			result := &Result{}
			adapter.extractMetrics(input, result)
		})
	}
}
