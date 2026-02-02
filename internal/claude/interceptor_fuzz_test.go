package claude

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzInferEventType fuzzes the event type inference with random JSON data.
func FuzzInferEventType(f *testing.F) {
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
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	interceptor := New()

	f.Fuzz(func(t *testing.T, input string) {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(input), &data); err != nil {
			return
		}
		_ = interceptor.inferEventType(data)
	})
}

// FuzzParseOutput fuzzes the JSON line parsing with random input.
func FuzzParseOutput(f *testing.F) {
	seeds := []string{
		`{"tool": "Read", "file_path": "/test.go"}` + "\n",
		`{"tool": "Write"}` + "\n" + `{"tool": "Edit"}` + "\n",
		`not json at all` + "\n",
		`{"incomplete":` + "\n",
		`{}` + "\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		if !utf8.ValidString(input) {
			return
		}

		interceptor := New()

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

		done := make(chan struct{})
		go func() {
			interceptor.parseOutput(strings.NewReader(input))
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
		}
	})
}

// TestFuzzMalformedJSON tests various malformed JSON patterns.
func TestFuzzMalformedJSON(t *testing.T) {
	interceptor := New()

	malformedInputs := []string{
		`{"tool": "Read"`,
		`{"tool":`,
		`{`,
		`{"tool": "Read"}}`,
		`{"tool": true}`,
		`{"tool": false}`,
		`{"tool": 3.14}`,
		``,
		`   `,
		`[]`,
		`[{"tool": "Read"}]`,
		`{"tool": "` + strings.Repeat("a", 10000) + `"}`,
	}

	for i, input := range malformedInputs {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			var data map[string]interface{}
			err := json.Unmarshal([]byte(input), &data)
			if err != nil {
				reader := strings.NewReader(input + "\n")
				done := make(chan struct{})
				go func() {
					interceptor.parseOutput(reader)
					close(done)
				}()

				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return
			}
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
					"content": strings.Repeat("x", 500000),
				}
				b, _ := json.Marshal(data)
				return append(b, '\n')
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			line := tc.genLine()
			reader := strings.NewReader(string(line))

			done := make(chan struct{})
			go func() {
				interceptor.parseOutput(reader)
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Error("timeout processing large input")
			}
		})
	}
}
