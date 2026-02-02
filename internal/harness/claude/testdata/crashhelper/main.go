// Package main provides a test binary that simulates various subprocess behaviors
// for testing harness crash handling. It is used by crash_test.go to verify that
// monitorProcess() correctly handles process crashes, hangs, and abnormal exits.
//
// Usage:
//
//	crashhelper --mode=<mode>
//
// Modes:
//   - exit0: Exit immediately with status 0
//   - exit1: Exit immediately with status 1
//   - sigkill: Exit via SIGKILL (simulates killed process)
//   - hang: Never exit (for testing timeout handling)
//   - output-then-crash: Output some JSON lines, then exit with status 1
//   - output-only: Output some JSON lines, then exit with status 0
//   - large-output: Output a very large JSON message (tests scanner buffer)
//   - slow-output: Output JSON lines with delays between them
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var mode = flag.String("mode", "exit0", "crash simulation mode")

func main() {
	flag.Parse()

	switch *mode {
	case "exit0":
		// Clean exit
		os.Exit(0)

	case "exit1":
		// Error exit
		os.Exit(1)

	case "sigkill":
		// Kill ourselves with SIGKILL to simulate being killed externally
		syscall.Kill(syscall.Getpid(), syscall.SIGKILL)
		// Should not reach here
		os.Exit(1)

	case "hang":
		// Block forever until killed - useful for testing timeout handling
		// Set up signal handler to allow graceful cleanup in tests
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
		<-sigCh
		os.Exit(0)

	case "output-then-crash":
		// Output some valid JSON lines, then crash
		fmt.Println(`{"type":"message_start"}`)
		fmt.Println(`{"type":"text","text":"Starting analysis..."}`)
		fmt.Println(`{"type":"text","text":"Processing file..."}`)
		// Flush stdout before crashing
		os.Stdout.Sync()
		os.Exit(1)

	case "output-only":
		// Output some valid JSON lines and exit cleanly
		fmt.Println(`{"type":"message_start"}`)
		fmt.Println(`{"type":"text","text":"Hello from test harness"}`)
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	case "large-output":
		// Output a very large JSON message to test scanner buffer handling
		// Default max is 1MB, so we'll output something just under that
		largeText := strings.Repeat("x", 900*1024) // ~900KB of text
		fmt.Printf(`{"type":"text","text":"%s"}`+"\n", largeText)
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	case "slow-output":
		// Output JSON lines with delays - tests reading behavior
		fmt.Println(`{"type":"message_start"}`)
		time.Sleep(100 * time.Millisecond)
		fmt.Println(`{"type":"text","text":"First message"}`)
		time.Sleep(100 * time.Millisecond)
		fmt.Println(`{"type":"text","text":"Second message"}`)
		time.Sleep(100 * time.Millisecond)
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	case "stderr-output":
		// Output to stderr (tests warning event generation)
		fmt.Fprintln(os.Stderr, "Warning: something happened")
		fmt.Fprintln(os.Stderr, "Error: something bad happened")
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	case "mixed-output":
		// Interleaved stdout and stderr
		fmt.Println(`{"type":"message_start"}`)
		fmt.Fprintln(os.Stderr, "Debug: starting process")
		fmt.Println(`{"type":"text","text":"Working..."}`)
		fmt.Fprintln(os.Stderr, "Debug: completed step 1")
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	case "malformed-json":
		// Output malformed JSON to test parser error handling
		fmt.Println(`{"type":"message_start"}`)
		fmt.Println(`not valid json at all`)
		fmt.Println(`{"truncated": true`)
		fmt.Println(`{"type":"message_stop","stop_reason":"end_turn"}`)
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", *mode)
		os.Exit(2)
	}
}
