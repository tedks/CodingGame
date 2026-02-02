package delve

import (
	"encoding/json"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockRPCServer simulates a delve RPC server for testing.
type mockRPCServer struct {
	listener   net.Listener
	conns      []net.Conn
	mu         sync.Mutex
	responseID atomic.Int64

	// Configurable behaviors for chaos testing
	delayResponse  time.Duration
	corruptID      bool // Return wrong response ID
	extraData      bool // Add extra buffered data after response
	closeOnConnect bool
}

func newMockRPCServer(t *testing.T) *mockRPCServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create mock server: %v", err)
	}

	server := &mockRPCServer{
		listener: listener,
	}

	go server.acceptLoop()
	return server
}

func (s *mockRPCServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}

		s.mu.Lock()
		s.conns = append(s.conns, conn)
		s.mu.Unlock()

		if s.closeOnConnect {
			conn.Close()
			continue
		}

		go s.handleConn(conn)
	}
}

func (s *mockRPCServer) handleConn(conn net.Conn) {
	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var request struct {
			JSONRPC string        `json:"jsonrpc"`
			ID      int64         `json:"id"`
			Method  string        `json:"method"`
			Params  []interface{} `json:"params"`
		}

		if err := decoder.Decode(&request); err != nil {
			return
		}

		if s.delayResponse > 0 {
			time.Sleep(s.delayResponse)
		}

		responseID := request.ID
		if s.corruptID {
			responseID = request.ID + 999 // Wrong ID
		}

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      responseID,
			"result": map[string]interface{}{
				"State": map[string]interface{}{
					"Running": false,
				},
			},
		}

		if err := encoder.Encode(response); err != nil {
			return
		}

		// Simulate extra buffered data (stale response problem)
		if s.extraData {
			// Write garbage that would confuse a new decoder
			conn.Write([]byte(`{"jsonrpc":"2.0","id":0,"result":{}}`))
		}
	}
}

func (s *mockRPCServer) Addr() string {
	return s.listener.Addr().String()
}

func (s *mockRPCServer) Close() {
	s.listener.Close()
	s.mu.Lock()
	for _, conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()
}

// TestRPCDecoderReuse verifies that the persistent decoder handles
// multiple sequential RPC calls correctly.
func TestRPCDecoderReuse(t *testing.T) {
	server := newMockRPCServer(t)
	defer server.Close()

	// Connect directly to mock server
	conn, err := net.Dial("tcp", server.Addr())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	state := &sessionState{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}

	adapter := NewAdapter()

	// Make multiple sequential RPC calls
	for i := 0; i < 10; i++ {
		_, err := adapter.callRPC(state, "State", map[string]interface{}{})
		if err != nil {
			t.Errorf("RPC call %d failed: %v", i, err)
		}
	}
}

// TestRPCResponseIDMismatch verifies that response ID validation catches
// mismatched responses.
func TestRPCResponseIDMismatch(t *testing.T) {
	server := newMockRPCServer(t)
	server.corruptID = true // Enable ID corruption
	defer server.Close()

	conn, err := net.Dial("tcp", server.Addr())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	state := &sessionState{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}

	adapter := NewAdapter()

	_, err = adapter.callRPC(state, "State", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for ID mismatch, got nil")
	} else if !containsIDMismatch(err.Error()) {
		t.Errorf("Expected ID mismatch error, got: %v", err)
	}
}

func containsIDMismatch(s string) bool {
	return len(s) > 0 && (contains(s, "mismatch") || contains(s, "ID"))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestRPCConcurrentCalls verifies that concurrent RPC calls are properly
// serialized and don't interfere with each other.
func TestRPCConcurrentCalls(t *testing.T) {
	server := newMockRPCServer(t)
	server.delayResponse = 10 * time.Millisecond // Add delay to increase interleaving
	defer server.Close()

	conn, err := net.Dial("tcp", server.Addr())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	state := &sessionState{
		conn:    conn,
		encoder: json.NewEncoder(conn),
		decoder: json.NewDecoder(conn),
	}

	adapter := NewAdapter()

	var wg sync.WaitGroup
	numCalls := 20
	errors := make(chan error, numCalls)

	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			_, err := adapter.callRPC(state, "State", map[string]interface{}{
				"call": n,
			})
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent RPC call failed: %v", err)
	}
}

// TestParallelSessions verifies that multiple independent sessions can
// run in parallel without interfering with each other.
func TestParallelSessions(t *testing.T) {
	adapter := NewAdapter()

	// Create multiple mock servers to simulate independent sessions
	numSessions := 5
	servers := make([]*mockRPCServer, numSessions)
	states := make([]*sessionState, numSessions)

	for i := 0; i < numSessions; i++ {
		servers[i] = newMockRPCServer(t)
		defer servers[i].Close()

		conn, err := net.Dial("tcp", servers[i].Addr())
		if err != nil {
			t.Fatalf("Failed to connect to server %d: %v", i, err)
		}
		defer conn.Close()

		states[i] = &sessionState{
			conn:    conn,
			encoder: json.NewEncoder(conn),
			decoder: json.NewDecoder(conn),
		}
	}

	var wg sync.WaitGroup
	numCallsPerSession := 10
	errors := make(chan error, numSessions*numCallsPerSession)

	for i := 0; i < numSessions; i++ {
		for j := 0; j < numCallsPerSession; j++ {
			wg.Add(1)
			go func(sessionIdx, callIdx int) {
				defer wg.Done()

				_, err := adapter.callRPC(states[sessionIdx], "State", map[string]interface{}{
					"session": sessionIdx,
					"call":    callIdx,
				})
				if err != nil {
					errors <- err
				}
			}(i, j)
		}
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Session error: %v", err)
	}

	if errorCount > 0 {
		t.Errorf("Had %d errors across parallel sessions", errorCount)
	}
}

// TestSessionStateRaceCondition verifies that the sessionState struct
// handles concurrent access correctly.
func TestSessionStateRaceCondition(t *testing.T) {
	// This test is primarily for the race detector to catch issues
	state := &sessionState{}

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Concurrent atomic operations
			state.rpcID.Add(1)
			_ = state.rpcID.Load()

			// Lock/unlock cycles
			state.rpcMu.Lock()
			// Simulate work
			time.Sleep(time.Microsecond)
			state.rpcMu.Unlock()
		}()
	}

	wg.Wait()

	// Verify ID was incremented correctly
	finalID := state.rpcID.Load()
	if finalID != int64(numGoroutines) {
		t.Errorf("Expected rpcID to be %d, got %d", numGoroutines, finalID)
	}
}

// TestAdapterSessionsMap verifies that the adapter's sessions map
// is accessed safely under concurrent load.
func TestAdapterSessionsMapConcurrency(t *testing.T) {
	adapter := NewAdapter()

	var wg sync.WaitGroup
	numGoroutines := 100

	// Simulate concurrent session lookups (getState calls)
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()

			// These will return nil (no session exists) but exercise the locking
			state := adapter.getState(nil)
			if state != nil {
				t.Error("Expected nil state for nil session")
			}

			// Interleave with small delays
			time.Sleep(time.Microsecond)
		}(i)
	}

	wg.Wait()
}
