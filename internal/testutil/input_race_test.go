package testutil

import (
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestTestInputSource_ConcurrentQueueAndAdvance tests that concurrent calls to
// QueueKeyPress and AdvanceFrame don't cause races or panics.
//
// Run with: bazel test --@io_bazel_rules_go//go/config:race //internal/testutil:testutil_test
func TestTestInputSource_ConcurrentQueueAndAdvance(t *testing.T) {
	source := NewTestInputSource()

	const numWriters = 4
	const numReaders = 4
	const iterations = 100

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Start writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			key := ebiten.Key(writerID % 10) // Use different keys
			for j := 0; j < iterations; j++ {
				select {
				case <-done:
					return
				default:
					source.QueueKeyPress(key)
				}
			}
		}(i)
	}

	// Start reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				select {
				case <-done:
					return
				default:
					source.AdvanceFrame()
					_ = source.JustPressedKeys()
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(done)

	// If we get here without panic or race detector error, the test passes
	t.Log("Concurrent queue and advance completed without race")
}

// TestTestInputSource_ConcurrentReads tests that concurrent read operations
// don't race with each other or with writes.
func TestTestInputSource_ConcurrentReads(t *testing.T) {
	source := NewTestInputSource()

	// Set up some initial state
	source.QueueKeyPress(ebiten.KeyA)
	source.QueueMouseMove(100, 200)
	source.QueueMouseClick(ebiten.MouseButtonLeft)
	source.AdvanceFrame()

	const numReaders = 8
	const iterations = 100

	var wg sync.WaitGroup

	// Multiple goroutines reading state simultaneously
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = source.IsKeyPressed(ebiten.KeyA)
				_ = source.IsKeyJustPressed(ebiten.KeyA)
				_ = source.JustPressedKeys()
				_, _ = source.CursorPosition()
				_ = source.IsMouseButtonPressed(ebiten.MouseButtonLeft)
				_ = source.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
				_, _ = source.Wheel()
				_ = source.AppendInputChars(nil)
				_ = source.HasPendingEvents()
			}
		}()
	}

	wg.Wait()
	t.Log("Concurrent reads completed without race")
}

// TestTestInputSource_ConcurrentMixedOperations tests a realistic scenario
// where events are queued, frames advance, and state is read concurrently.
func TestTestInputSource_ConcurrentMixedOperations(t *testing.T) {
	source := NewTestInputSource()

	const iterations = 50

	var wg sync.WaitGroup
	done := make(chan struct{})

	// Writer: queues various events
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			select {
			case <-done:
				return
			default:
				source.QueueKeyPress(ebiten.Key(i % 26)) // A-Z
				source.QueueMouseMove(i, i*2)
				source.QueueCharInput(rune('a' + (i % 26)))
				source.QueueMouseWheel(float64(i%3), float64(i%3))
			}
		}
	}()

	// Advancer: processes frames
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			select {
			case <-done:
				return
			default:
				source.AdvanceFrame()
			}
		}
	}()

	// Reader: queries state
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			select {
			case <-done:
				return
			default:
				_ = source.JustPressedKeys()
				_ = source.IsKeyPressed(ebiten.KeyA)
				_, _ = source.CursorPosition()
				_ = source.AppendInputChars(nil)
				_, _ = source.Wheel()
			}
		}
	}()

	// Clearer: occasionally clears state
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations/10; i++ {
			select {
			case <-done:
				return
			default:
				source.Clear()
			}
		}
	}()

	wg.Wait()
	close(done)
	t.Log("Concurrent mixed operations completed without race")
}

// TestTestInputSource_ConcurrentKeyHold tests that key hold operations
// are thread-safe during concurrent access.
func TestTestInputSource_ConcurrentKeyHold(t *testing.T) {
	source := NewTestInputSource()

	const numGoroutines = 4
	const iterations = 50

	var wg sync.WaitGroup

	// Multiple goroutines queuing key holds
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := ebiten.Key(id % 10)
			for j := 0; j < iterations; j++ {
				source.QueueKeyHold(key, 3)
				source.AdvanceFrame()
				_ = source.IsKeyPressed(key)
			}
		}(i)
	}

	wg.Wait()
	t.Log("Concurrent key hold operations completed without race")
}

// TestTestInputSource_ConcurrentCharInput tests that character input
// operations are thread-safe.
func TestTestInputSource_ConcurrentCharInput(t *testing.T) {
	source := NewTestInputSource()

	const numWriters = 4
	const iterations = 50

	var wg sync.WaitGroup

	// Writers: queue characters
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				source.QueueTextInput("hello")
				source.QueueCharInput('a', 'b', 'c')
			}
		}(i)
	}

	// Reader: consume characters
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < iterations*numWriters; j++ {
			source.AdvanceFrame()
			_ = source.AppendInputChars(nil)
		}
	}()

	wg.Wait()
	t.Log("Concurrent char input completed without race")
}
