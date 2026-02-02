package claude

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// =============================================================================
// FUZZ TESTS
// =============================================================================

// FuzzInferEventType fuzzes the event type inference with random JSON data.
func FuzzInferEventType(f *testing.F) {
	// Seed corpus with known inputs
	seeds := []string{
		`{"tool": "Read"}`,
		`{"tool": "Write"}`,
		`{"tool": "Edit"}`,
		`{"tool": "Bash", "command": "go build"}`,
		`{"tool": "Bash", "command": "go test"}`,
		`{"tool": "Task"}`,
		`{"text": "hello"}`,
		`{}`,
		`{"tool": 123}`,
		`{"tool": null}`,
		`{"tool": "Read", "extra": {"nested": "value"}}`,
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	interceptor := New()

	f.Fuzz(func(t *testing.T, input string) {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(input), &data); err != nil {
			// Invalid JSON is fine - just skip it
			return
		}

		// Should not panic regardless of input
		_ = interceptor.inferEventType(data)
	})
}

// FuzzParseOutput fuzzes the JSON line parsing with random input.
func FuzzParseOutput(f *testing.F) {
	// Seed corpus
	seeds := []string{
		`{"tool": "Read", "file_path": "/test.go"}` + "\n",
		`{"tool": "Write"}` + "\n" + `{"tool": "Edit"}` + "\n",
		`not json at all` + "\n",
		`{"incomplete":` + "\n",
		`{}` + "\n",
		string(make([]byte, 1000)) + "\n", // Large line
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		interceptor := New()

		// Must have valid UTF-8
		if !utf8.ValidString(input) {
			return
		}

		received := make(chan *Event, 100)
		interceptor.AddHandler(func(e *Event) {
			select {
			case received <- e:
			default:
			}
		})

		if err := interceptor.Start(); err != nil {
			t.Skip("failed to start")
		}
		defer interceptor.Stop()

		// parseOutput should handle any input without panicking
		done := make(chan struct{})
		go func() {
			interceptor.parseOutput(strings.NewReader(input))
			close(done)
		}()

		select {
		case <-done:
			// parseOutput completed
		case <-time.After(time.Second):
			// Timeout is acceptable for very long inputs
		}
	})
}

// =============================================================================
// TARGETED FUZZ-LIKE TESTS (for environments without native fuzz support)
// =============================================================================

// TestFuzzMalformedJSON tests various malformed JSON patterns.
func TestFuzzMalformedJSON(t *testing.T) {
	interceptor := New()

	malformedInputs := []string{
		// Truncated JSON
		`{"tool": "Read"`,
		`{"tool":`,
		`{`,
		`{"tool": "Read", "file_path": "/test.go`,

		// Unbalanced braces
		`{"tool": "Read"}}`,
		`{{"tool": "Read"}`,
		`{"tool": ["Read"]]`,

		// Invalid escapes
		`{"tool": "\x00"}`,
		`{"tool": "\u000"}`,
		`{"tool": "Read\"}`,

		// Type mismatches
		`{"tool": true}`,
		`{"tool": false}`,
		`{"tool": 3.14}`,
		`{"tool": -1}`,

		// Empty and whitespace
		``,
		`   `,
		`\n\n\n`,

		// Deeply nested
		`{"a":{"b":{"c":{"d":{"e":{"f":"Read"}}}}}}`,

		// Arrays where objects expected
		`[]`,
		`[{"tool": "Read"}]`,

		// Unicode edge cases
		`{"tool": "\u0000Read"}`,
		`{"tool": "Read\uD800"}`,      // Unpaired surrogate
		`{"tool": "\uFFFE"}`,          // Non-character
		`{"tool": "Read\u202E"}`,      // Right-to-left override

		// Very long strings
		`{"tool": "` + strings.Repeat("a", 10000) + `"}`,

		// Control characters
		`{"tool": "Read` + "\x00" + `"}`,
		`{"tool": "Read` + "\x1F" + `"}`,

		// Comments (not valid JSON but sometimes attempted)
		`{"tool": "Read"} // comment`,
		`/* comment */ {"tool": "Read"}`,
	}

	for i, input := range malformedInputs {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			var data map[string]interface{}
			err := json.Unmarshal([]byte(input), &data)
			if err != nil {
				// Invalid JSON - verify parseOutput handles it gracefully
				reader := strings.NewReader(input + "\n")
				done := make(chan struct{})
				go func() {
					interceptor.parseOutput(reader)
					close(done)
				}()

				select {
				case <-done:
					// Good - completed without panic
				case <-time.After(time.Second):
					// Timeout is acceptable
				}
				return
			}

			// Valid JSON - inferEventType should not panic
			_ = interceptor.inferEventType(data)
		})
	}
}

// TestFuzzLargeInputs tests behavior with very large inputs.
func TestFuzzLargeInputs(t *testing.T) {
	interceptor := New()

	received := make(chan *Event, 10)
	interceptor.AddHandler(func(e *Event) {
		select {
		case received <- e:
		default:
		}
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	// Test various large input scenarios
	testCases := []struct {
		name    string
		genLine func() []byte
	}{
		{
			name: "large_valid_json",
			genLine: func() []byte {
				data := map[string]interface{}{
					"tool":      "Read",
					"file_path": "/" + strings.Repeat("a", 100000),
				}
				b, _ := json.Marshal(data)
				return append(b, '\n')
			},
		},
		{
			name: "large_content_field",
			genLine: func() []byte {
				data := map[string]interface{}{
					"tool":    "Read",
					"content": strings.Repeat("x", 500000), // 500KB
				}
				b, _ := json.Marshal(data)
				return append(b, '\n')
			},
		},
		{
			name: "many_fields",
			genLine: func() []byte {
				data := make(map[string]interface{})
				data["tool"] = "Read"
				for i := 0; i < 1000; i++ {
					data[strings.Repeat("f", i%50+1)+string(rune(i))] = i
				}
				b, _ := json.Marshal(data)
				return append(b, '\n')
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line := tc.genLine()
			reader := bytes.NewReader(line)

			done := make(chan struct{})
			go func() {
				interceptor.parseOutput(reader)
				close(done)
			}()

			select {
			case <-done:
				// Completed
			case <-time.After(5 * time.Second):
				t.Error("timeout processing large input")
			}
		})
	}
}

// TestFuzzRapidEmit tests rapid event emission for memory/goroutine issues.
func TestFuzzRapidEmit(t *testing.T) {
	interceptor := New()

	var count int
	interceptor.AddHandler(func(e *Event) {
		count++
	})

	if err := interceptor.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}

	// Rapid fire events
	for i := 0; i < 10000; i++ {
		interceptor.emitEvent(&Event{
			Type:      EventFileRead,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"i": i},
		})
	}

	// Give time to process
	time.Sleep(100 * time.Millisecond)

	if err := interceptor.Stop(); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	// Not all events may be processed (channel may block) but should not panic
	if count == 0 {
		t.Error("no events were processed")
	}
}
