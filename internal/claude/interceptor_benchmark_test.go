package claude

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkEventDispatch measures the overhead of dispatching a single event.
func BenchmarkEventDispatch(b *testing.B) {
	interceptor := New()

	var count int64
	interceptor.AddHandler(func(e *Event) {
		atomic.AddInt64(&count, 1)
	})

	if err := interceptor.Start(); err != nil {
		b.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	event := &Event{
		Type:      EventFileRead,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"tool": "Read", "file_path": "/test.go"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.emitEvent(event)
	}
	b.StopTimer()

	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt64(&count) < int64(b.N) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
}

// BenchmarkMultiHandler measures dispatch overhead with multiple handlers.
func BenchmarkMultiHandler(b *testing.B) {
	for _, numHandlers := range []int{1, 5, 10, 20} {
		b.Run(fmt.Sprintf("handlers=%d", numHandlers), func(b *testing.B) {
			interceptor := New()

			var count int64
			for i := 0; i < numHandlers; i++ {
				interceptor.AddHandler(func(e *Event) {
					atomic.AddInt64(&count, 1)
				})
			}

			if err := interceptor.Start(); err != nil {
				b.Fatalf("failed to start: %v", err)
			}
			defer interceptor.Stop()

			event := &Event{
				Type:      EventFileRead,
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"tool": "Read"},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				interceptor.emitEvent(event)
			}
			b.StopTimer()

			expected := int64(b.N * numHandlers)
			deadline := time.Now().Add(10 * time.Second)
			for atomic.LoadInt64(&count) < expected && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
		})
	}
}

// BenchmarkInferEventType measures the speed of event type inference.
func BenchmarkInferEventType(b *testing.B) {
	interceptor := New()

	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{"read", map[string]interface{}{"tool": "Read", "file_path": "/test.go"}},
		{"write", map[string]interface{}{"tool": "Write", "file_path": "/test.go"}},
		{"bash_build", map[string]interface{}{"tool": "Bash", "command": "go build ./..."}},
		{"bash_test", map[string]interface{}{"tool": "Bash", "command": "go test ./..."}},
		{"bash_other", map[string]interface{}{"tool": "Bash", "command": "ls -la"}},
		{"text", map[string]interface{}{"text": "some content"}},
		{"unknown", map[string]interface{}{"unknown": "field"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				interceptor.inferEventType(tc.data)
			}
		})
	}
}

// BenchmarkHighVolume measures sustained throughput.
func BenchmarkHighVolume(b *testing.B) {
	interceptor := New()

	var count int64
	interceptor.AddHandler(func(e *Event) {
		atomic.AddInt64(&count, 1)
	})

	if err := interceptor.Start(); err != nil {
		b.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	events := make([]*Event, 1000)
	for i := range events {
		events[i] = &Event{
			Type:      EventFileRead,
			Timestamp: time.Now(),
			Data:      map[string]interface{}{"tool": "Read", "file_path": fmt.Sprintf("/file%d.go", i)},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		interceptor.emitEvent(events[i%len(events)])
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkStartStop measures the overhead of lifecycle operations.
func BenchmarkStartStop(b *testing.B) {
	for i := 0; i < b.N; i++ {
		interceptor := New()
		interceptor.Start()
		interceptor.Stop()
	}
}

// BenchmarkContainsAny measures the string matching performance.
func BenchmarkContainsAny(b *testing.B) {
	testCases := []struct {
		name    string
		command string
		terms   []string
	}{
		{"short_match", "go build ./...", []string{"build", "test"}},
		{"short_nomatch", "ls -la", []string{"build", "test"}},
		{"long_match", "bazel build //... && bazel test //... && echo done", []string{"build", "test"}},
		{"many_terms", "go build ./...", []string{"build", "test", "cargo", "make", "npm", "yarn", "gradle", "maven"}},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				containsAny(tc.command, tc.terms)
			}
		})
	}
}

// BenchmarkParseOutput measures the full parsing pipeline.
func BenchmarkParseOutput(b *testing.B) {
	interceptor := New()

	var count int64
	interceptor.AddHandler(func(e *Event) {
		atomic.AddInt64(&count, 1)
	})

	if err := interceptor.Start(); err != nil {
		b.Fatalf("failed to start: %v", err)
	}
	defer interceptor.Stop()

	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString(`{"tool": "Read", "file_path": "/test.go"}`)
		sb.WriteString("\n")
	}
	input := sb.String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		atomic.StoreInt64(&count, 0)
		interceptor.parseOutput(strings.NewReader(input))
	}
	b.StopTimer()

	b.ReportMetric(float64(b.N*100)/b.Elapsed().Seconds(), "lines/sec")
}
