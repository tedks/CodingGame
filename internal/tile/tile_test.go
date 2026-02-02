package tile

import (
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (f *fakeClock) Now() time.Time {
	return f.now
}

func (f *fakeClock) Advance(d time.Duration) {
	f.now = f.now.Add(d)
}

func TestNew(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	if tile.Path() != "/path/to/file.go" {
		t.Errorf("expected path /path/to/file.go, got %s", tile.Path())
	}
	if tile.RelPath() != "file.go" {
		t.Errorf("expected relPath file.go, got %s", tile.RelPath())
	}
	if tile.IsDirectory() {
		t.Error("expected IsDirectory to be false")
	}
	if tile.Name() != "file.go" {
		t.Errorf("expected name file.go, got %s", tile.Name())
	}
	if tile.Extension() != ".go" {
		t.Errorf("expected extension .go, got %s", tile.Extension())
	}
}

func TestFogStates(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	// Should start fogged
	if tile.FogState() != FogFull {
		t.Errorf("expected FogFull, got %v", tile.FogState())
	}
	if tile.IsRevealed() {
		t.Error("expected tile to not be revealed initially")
	}

	// Reveal the tile
	tile.Reveal()
	if tile.FogState() != FogRevealed {
		t.Errorf("expected FogRevealed after Reveal(), got %v", tile.FogState())
	}
	if !tile.IsRevealed() {
		t.Error("expected tile to be revealed after Reveal()")
	}
	if tile.RevealCount() != 1 {
		t.Errorf("expected reveal count 1, got %d", tile.RevealCount())
	}

	// Mark as stale
	tile.MarkStale()
	if tile.FogState() != FogStale {
		t.Errorf("expected FogStale after MarkStale(), got %v", tile.FogState())
	}
	if !tile.IsStale() {
		t.Error("expected tile to be stale after MarkStale()")
	}

	// Reset fog
	tile.ResetFog()
	if tile.FogState() != FogFull {
		t.Errorf("expected FogFull after ResetFog(), got %v", tile.FogState())
	}
}

func TestHighlight(t *testing.T) {
	clock := &fakeClock{now: time.Unix(0, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// Should not be highlighted initially
	if tile.IsHighlighted() {
		t.Error("expected tile to not be highlighted initially")
	}

	// Highlight for 100ms
	tile.Highlight(100 * time.Millisecond)
	if !tile.IsHighlighted() {
		t.Error("expected tile to be highlighted after Highlight()")
	}

	// Wait for highlight to expire
	clock.Advance(150 * time.Millisecond)
	if tile.IsHighlighted() {
		t.Error("expected highlight to expire after duration")
	}
}

func TestRevealCount(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	if tile.RevealCount() != 0 {
		t.Errorf("expected initial reveal count 0, got %d", tile.RevealCount())
	}

	// Reveal multiple times
	tile.Reveal()
	tile.Reveal()
	tile.Reveal()

	if tile.RevealCount() != 3 {
		t.Errorf("expected reveal count 3, got %d", tile.RevealCount())
	}
}

func TestDirectoryTile(t *testing.T) {
	tile := New("/path/to/dir", "dir", true)

	if !tile.IsDirectory() {
		t.Error("expected IsDirectory to be true")
	}
	if tile.Extension() != "" {
		t.Errorf("expected empty extension for directory, got %s", tile.Extension())
	}
}

func TestUpdateMetadata(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	modTime := time.Now()
	tile.UpdateMetadata(1024, modTime)

	if tile.SizeBytes() != 1024 {
		t.Errorf("expected size 1024, got %d", tile.SizeBytes())
	}
	if !tile.LastModified().Equal(modTime) {
		t.Errorf("expected modTime %v, got %v", modTime, tile.LastModified())
	}
}

func TestConcurrentAccess(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	// Test concurrent reads and writes
	done := make(chan bool)

	// Reader goroutine
	go func() {
		for i := 0; i < 100; i++ {
			_ = tile.IsRevealed()
			_ = tile.FogState()
			_ = tile.Path()
		}
		done <- true
	}()

	// Writer goroutine
	go func() {
		for i := 0; i < 100; i++ {
			tile.Reveal()
			tile.MarkStale()
			tile.ResetFog()
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// If we get here without deadlock or data race, test passes
}

// =============================================================================
// State Transition Edge Cases (Task 1)
// =============================================================================

func TestMarkStaleFromNonRevealed(t *testing.T) {
	t.Run("MarkStale from FogFull is no-op", func(t *testing.T) {
		tile := New("/path/to/file.go", "file.go", false)

		// Initial state should be FogFull
		if tile.FogState() != FogFull {
			t.Fatalf("precondition failed: expected FogFull, got %v", tile.FogState())
		}

		// MarkStale from FogFull should be a no-op
		tile.MarkStale()

		if tile.FogState() != FogFull {
			t.Errorf("MarkStale() from FogFull should be no-op, got %v", tile.FogState())
		}
	})

	t.Run("MarkStale from FogStale is idempotent", func(t *testing.T) {
		tile := New("/path/to/file.go", "file.go", false)

		// Get to FogStale state
		tile.Reveal()
		tile.MarkStale()
		if tile.FogState() != FogStale {
			t.Fatalf("precondition failed: expected FogStale, got %v", tile.FogState())
		}

		// MarkStale again should be idempotent
		tile.MarkStale()

		if tile.FogState() != FogStale {
			t.Errorf("MarkStale() from FogStale should be idempotent, got %v", tile.FogState())
		}
	})
}

func TestResetFogFromAllStates(t *testing.T) {
	t.Run("ResetFog from FogFull is idempotent", func(t *testing.T) {
		tile := New("/path/to/file.go", "file.go", false)

		// Should start at FogFull
		if tile.FogState() != FogFull {
			t.Fatalf("precondition failed: expected FogFull, got %v", tile.FogState())
		}

		// ResetFog should be idempotent
		tile.ResetFog()

		if tile.FogState() != FogFull {
			t.Errorf("ResetFog() from FogFull should be idempotent, got %v", tile.FogState())
		}
	})

	t.Run("ResetFog from FogStale transitions to FogFull", func(t *testing.T) {
		tile := New("/path/to/file.go", "file.go", false)

		// Get to FogStale state
		tile.Reveal()
		tile.MarkStale()
		if tile.FogState() != FogStale {
			t.Fatalf("precondition failed: expected FogStale, got %v", tile.FogState())
		}

		// ResetFog should go to FogFull
		tile.ResetFog()

		if tile.FogState() != FogFull {
			t.Errorf("ResetFog() from FogStale should transition to FogFull, got %v", tile.FogState())
		}
	})

	t.Run("ResetFog from FogRevealed transitions to FogFull", func(t *testing.T) {
		tile := New("/path/to/file.go", "file.go", false)

		// Get to FogRevealed state
		tile.Reveal()
		if tile.FogState() != FogRevealed {
			t.Fatalf("precondition failed: expected FogRevealed, got %v", tile.FogState())
		}

		// ResetFog should go to FogFull
		tile.ResetFog()

		if tile.FogState() != FogFull {
			t.Errorf("ResetFog() from FogRevealed should transition to FogFull, got %v", tile.FogState())
		}
	})
}

func TestRevealFromStale(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// Get to FogStale state
	tile.Reveal()
	tile.MarkStale()
	if tile.FogState() != FogStale {
		t.Fatalf("precondition failed: expected FogStale, got %v", tile.FogState())
	}
	initialCount := tile.RevealCount()

	// Advance time and reveal again
	clock.Advance(time.Hour)
	tile.Reveal()

	if tile.FogState() != FogRevealed {
		t.Errorf("Reveal() from FogStale should transition to FogRevealed, got %v", tile.FogState())
	}
	if tile.RevealCount() != initialCount+1 {
		t.Errorf("Reveal() should increment revealCount, expected %d, got %d", initialCount+1, tile.RevealCount())
	}
}

// =============================================================================
// Double-Reveal Behavior Documentation (Task 2)
// =============================================================================

// TestDoubleRevealIsIntentional verifies that calling Reveal() on an already-revealed
// tile increments the revealCount. This is INTENTIONAL behavior - every Reveal() call
// represents a distinct read event by Claude. The game uses revealCount to track
// "how many times has Claude looked at this file", which is valuable for:
//   - Heat maps of frequently-read files
//   - Identifying "hot spots" in the codebase
//   - Understanding context usage patterns
//
// This is NOT a bug. Each call to Reveal() means Claude read the file again.
func TestDoubleRevealIsIntentional(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// First reveal
	tile.Reveal()
	firstRevealTime := tile.LastRevealed()
	if tile.FogState() != FogRevealed {
		t.Fatalf("precondition failed: expected FogRevealed, got %v", tile.FogState())
	}
	if tile.RevealCount() != 1 {
		t.Fatalf("precondition failed: expected revealCount 1, got %d", tile.RevealCount())
	}

	// Advance time and reveal again while already revealed
	clock.Advance(time.Minute)
	tile.Reveal()

	// State should remain FogRevealed
	if tile.FogState() != FogRevealed {
		t.Errorf("double Reveal() should remain FogRevealed, got %v", tile.FogState())
	}

	// Count should increment (intentional - each Reveal() is a read event)
	if tile.RevealCount() != 2 {
		t.Errorf("double Reveal() should increment count: expected 2, got %d", tile.RevealCount())
	}

	// LastRevealed should update
	if !tile.LastRevealed().After(firstRevealTime) {
		t.Errorf("double Reveal() should update lastRevealed: first=%v, second=%v",
			firstRevealTime, tile.LastRevealed())
	}
}

// =============================================================================
// Reset Semantics: "Hide But Remember" (Task 3)
// =============================================================================

// TestResetFogPreservesHistory verifies that ResetFog() implements "hide but remember"
// semantics. When fog is reset, the tile becomes invisible again, but the history of
// how many times it was revealed AND when it was last revealed are preserved.
func TestResetFogPreservesHistory(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// Reveal the tile
	tile.Reveal()
	revealTime := tile.LastRevealed()
	if tile.RevealCount() != 1 {
		t.Fatalf("precondition failed: expected revealCount 1, got %d", tile.RevealCount())
	}

	// Reset fog
	clock.Advance(time.Hour)
	tile.ResetFog()

	// Fog state should be reset
	if tile.FogState() != FogFull {
		t.Errorf("ResetFog() should set state to FogFull, got %v", tile.FogState())
	}

	// RevealCount should persist (history preserved)
	if tile.RevealCount() != 1 {
		t.Errorf("ResetFog() should preserve revealCount: expected 1, got %d", tile.RevealCount())
	}

	// LastRevealed should persist (history preserved)
	if !tile.LastRevealed().Equal(revealTime) {
		t.Errorf("ResetFog() should preserve lastRevealed: expected %v, got %v",
			revealTime, tile.LastRevealed())
	}
}

// =============================================================================
// Full State Cycle (Task 4)
// =============================================================================

func TestFullStateCycle(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// Initial state: FogFull
	if tile.FogState() != FogFull {
		t.Fatalf("initial state should be FogFull, got %v", tile.FogState())
	}
	if tile.RevealCount() != 0 {
		t.Fatalf("initial revealCount should be 0, got %d", tile.RevealCount())
	}

	// Cycle 1: FogFull -> Reveal -> FogRevealed
	clock.Advance(time.Minute)
	tile.Reveal()
	if tile.FogState() != FogRevealed {
		t.Errorf("after first Reveal(): expected FogRevealed, got %v", tile.FogState())
	}
	if tile.RevealCount() != 1 {
		t.Errorf("after first Reveal(): expected revealCount 1, got %d", tile.RevealCount())
	}

	// Cycle 1: FogRevealed -> MarkStale -> FogStale
	clock.Advance(time.Minute)
	tile.MarkStale()
	if tile.FogState() != FogStale {
		t.Errorf("after MarkStale(): expected FogStale, got %v", tile.FogState())
	}

	// Cycle 1: FogStale -> ResetFog -> FogFull
	clock.Advance(time.Minute)
	tile.ResetFog()
	if tile.FogState() != FogFull {
		t.Errorf("after ResetFog(): expected FogFull, got %v", tile.FogState())
	}

	// Cycle 2: Repeat the full cycle
	clock.Advance(time.Minute)
	tile.Reveal()
	if tile.FogState() != FogRevealed {
		t.Errorf("cycle 2 Reveal(): expected FogRevealed, got %v", tile.FogState())
	}

	clock.Advance(time.Minute)
	tile.MarkStale()
	if tile.FogState() != FogStale {
		t.Errorf("cycle 2 MarkStale(): expected FogStale, got %v", tile.FogState())
	}

	clock.Advance(time.Minute)
	tile.ResetFog()
	if tile.FogState() != FogFull {
		t.Errorf("cycle 2 ResetFog(): expected FogFull, got %v", tile.FogState())
	}

	// After 2 cycles, revealCount should be 2
	if tile.RevealCount() != 2 {
		t.Errorf("after 2 cycles, revealCount should be 2, got %d", tile.RevealCount())
	}
}

// =============================================================================
// Path Edge Cases (Task 5)
// =============================================================================

func TestPathEdgeCases(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		tile := New("", "", false)

		// filepath.Base("") returns "."
		if tile.Name() != "." {
			t.Errorf("expected name '.' for empty path, got %q", tile.Name())
		}
		if tile.Path() != "" {
			t.Errorf("expected empty path, got %q", tile.Path())
		}
		if tile.RelPath() != "" {
			t.Errorf("expected empty relPath, got %q", tile.RelPath())
		}
	})

	t.Run("dotfiles", func(t *testing.T) {
		tile := New("/path/.gitignore", ".gitignore", false)

		if tile.Name() != ".gitignore" {
			t.Errorf("expected name '.gitignore', got %q", tile.Name())
		}
		// filepath.Ext(".gitignore") returns ".gitignore" - Go considers the whole
		// filename after the leading dot as the extension for pure dotfiles
		if tile.Extension() != ".gitignore" {
			t.Errorf("expected extension '.gitignore' for dotfile, got %q", tile.Extension())
		}
	})

	t.Run("dotfiles with extension", func(t *testing.T) {
		tile := New("/path/.eslintrc.json", ".eslintrc.json", false)

		if tile.Name() != ".eslintrc.json" {
			t.Errorf("expected name '.eslintrc.json', got %q", tile.Name())
		}
		// filepath.Ext(".eslintrc.json") returns ".json"
		if tile.Extension() != ".json" {
			t.Errorf("expected extension '.json', got %q", tile.Extension())
		}
	})

	t.Run("multiple dots in filename", func(t *testing.T) {
		tile := New("/path/file.test.go", "file.test.go", false)

		if tile.Name() != "file.test.go" {
			t.Errorf("expected name 'file.test.go', got %q", tile.Name())
		}
		// filepath.Ext returns last extension only
		if tile.Extension() != ".go" {
			t.Errorf("expected extension '.go' for multiple dots, got %q", tile.Extension())
		}
	})

	t.Run("trailing slash for directory", func(t *testing.T) {
		tile := New("/path/dir/", "dir/", true)

		// filepath.Base("/path/dir/") returns "dir" (trailing slash is stripped)
		if tile.Name() != "dir" {
			t.Errorf("expected name 'dir' for trailing slash, got %q", tile.Name())
		}
		if !tile.IsDirectory() {
			t.Error("expected IsDirectory to be true")
		}
		// Directories should have no extension
		if tile.Extension() != "" {
			t.Errorf("expected empty extension for directory, got %q", tile.Extension())
		}
	})

	t.Run("file with no extension", func(t *testing.T) {
		tile := New("/path/Makefile", "Makefile", false)

		if tile.Name() != "Makefile" {
			t.Errorf("expected name 'Makefile', got %q", tile.Name())
		}
		if tile.Extension() != "" {
			t.Errorf("expected empty extension, got %q", tile.Extension())
		}
	})

	t.Run("deeply nested path", func(t *testing.T) {
		tile := New("/a/b/c/d/e/f/g.go", "b/c/d/e/f/g.go", false)

		if tile.Name() != "g.go" {
			t.Errorf("expected name 'g.go', got %q", tile.Name())
		}
		if tile.Path() != "/a/b/c/d/e/f/g.go" {
			t.Errorf("expected path '/a/b/c/d/e/f/g.go', got %q", tile.Path())
		}
		if tile.RelPath() != "b/c/d/e/f/g.go" {
			t.Errorf("expected relPath 'b/c/d/e/f/g.go', got %q", tile.RelPath())
		}
	})
}

// =============================================================================
// Metadata/State Interaction (Task 6)
// =============================================================================

// TestMetadataAfterReveal documents the relationship between lastRevealed and
// lastModified timestamps. Callers must compare these timestamps themselves to
// determine if content is stale.
func TestMetadataAfterReveal(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	// Reveal at T1
	tile.Reveal()
	revealTime := tile.LastRevealed()

	// Update metadata with modification time before reveal
	pastModTime := time.Unix(500, 0)
	tile.UpdateMetadata(1024, pastModTime)

	// Content is NOT stale (modified before revealed)
	if tile.LastModified().After(revealTime) {
		t.Errorf("file modified before reveal should not appear stale")
	}

	// Update metadata with modification time after reveal
	clock.Advance(time.Hour)
	futureModTime := clock.Now()
	tile.UpdateMetadata(2048, futureModTime)

	// Content IS stale (modified after revealed)
	if !tile.LastModified().After(revealTime) {
		t.Errorf("file modified after reveal should appear stale: lastModified=%v, lastRevealed=%v",
			tile.LastModified(), revealTime)
	}

	// Verify that fog state and metadata are independent
	if tile.FogState() != FogRevealed {
		t.Errorf("UpdateMetadata should not affect fog state, got %v", tile.FogState())
	}
}

func TestMetadataInitialValues(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	// Initial values should be zero values
	if tile.SizeBytes() != 0 {
		t.Errorf("expected initial size 0, got %d", tile.SizeBytes())
	}
	if !tile.LastModified().IsZero() {
		t.Errorf("expected zero lastModified, got %v", tile.LastModified())
	}
	if !tile.LastRevealed().IsZero() {
		t.Errorf("expected zero lastRevealed, got %v", tile.LastRevealed())
	}
}

func TestMetadataDoesNotAffectFogState(t *testing.T) {
	tests := []struct {
		name     string
		expected FogState
	}{
		{"FogFull", FogFull},
		{"FogRevealed", FogRevealed},
		{"FogStale", FogStale},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tile := New("/path/to/file.go", "file.go", false)

			// Set up initial state
			switch tt.expected {
			case FogFull:
				// Default state
			case FogRevealed:
				tile.Reveal()
			case FogStale:
				tile.Reveal()
				tile.MarkStale()
			}

			// Verify we're in the expected state
			if tile.FogState() != tt.expected {
				t.Fatalf("setup failed: expected %v, got %v", tt.expected, tile.FogState())
			}

			// Update metadata multiple times
			for i := 0; i < 10; i++ {
				tile.UpdateMetadata(int64(i*1000), time.Now())
			}

			// State should be unchanged
			if tile.FogState() != tt.expected {
				t.Errorf("UpdateMetadata should not change fog state from %v, got %v",
					tt.expected, tile.FogState())
			}
		})
	}
}

// =============================================================================
// Enhanced Concurrent Tests (Task 7)
// =============================================================================

func TestConcurrentStateConsistency(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	const numGoroutines = 10
	const opsPerGoroutine = 100

	done := make(chan struct{})
	started := make(chan struct{})

	// Spawn multiple writer goroutines doing different operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			<-started // Wait for all goroutines to be ready
			for j := 0; j < opsPerGoroutine; j++ {
				switch id % 3 {
				case 0:
					tile.Reveal()
				case 1:
					tile.MarkStale()
				case 2:
					tile.ResetFog()
				}
			}
			done <- struct{}{}
		}(i)
	}

	// Spawn reader goroutines
	for i := 0; i < numGoroutines; i++ {
		go func() {
			<-started // Wait for all goroutines to be ready
			for j := 0; j < opsPerGoroutine; j++ {
				state := tile.FogState()
				// Verify state is valid
				if state != FogFull && state != FogRevealed && state != FogStale {
					t.Errorf("invalid fog state: %v", state)
				}
				_ = tile.IsRevealed()
				_ = tile.IsStale()
				_ = tile.RevealCount()
				_ = tile.LastRevealed()
			}
			done <- struct{}{}
		}()
	}

	// Start all goroutines simultaneously
	close(started)

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// Final state should be valid
	finalState := tile.FogState()
	if finalState != FogFull && finalState != FogRevealed && finalState != FogStale {
		t.Errorf("final state should be valid, got %v", finalState)
	}

	// RevealCount should be non-negative
	if tile.RevealCount() < 0 {
		t.Errorf("revealCount should be non-negative, got %d", tile.RevealCount())
	}
}

func TestConcurrentMetadataUpdates(t *testing.T) {
	tile := New("/path/to/file.go", "file.go", false)

	const numGoroutines = 10
	const opsPerGoroutine = 100

	done := make(chan struct{})
	started := make(chan struct{})

	// Writers updating metadata
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			<-started
			for j := 0; j < opsPerGoroutine; j++ {
				tile.UpdateMetadata(int64(id*1000+j), time.Now())
			}
			done <- struct{}{}
		}(i)
	}

	// Readers checking metadata
	for i := 0; i < numGoroutines; i++ {
		go func() {
			<-started
			for j := 0; j < opsPerGoroutine; j++ {
				size := tile.SizeBytes()
				if size < 0 {
					t.Errorf("size should be non-negative, got %d", size)
				}
				_ = tile.LastModified()
			}
			done <- struct{}{}
		}()
	}

	close(started)

	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}

	// Final size should be non-negative
	if tile.SizeBytes() < 0 {
		t.Errorf("final size should be non-negative, got %d", tile.SizeBytes())
	}
}

func TestConcurrentHighlight(t *testing.T) {
	clock := &fakeClock{now: time.Unix(1000, 0)}
	tile := newWithClock("/path/to/file.go", "file.go", false, clock)

	const numGoroutines = 10
	const opsPerGoroutine = 50

	done := make(chan struct{})
	started := make(chan struct{})

	// Writers setting highlights
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			<-started
			for j := 0; j < opsPerGoroutine; j++ {
				tile.Highlight(time.Duration(id*100+j) * time.Millisecond)
			}
			done <- struct{}{}
		}(i)
	}

	// Readers checking highlight status
	for i := 0; i < numGoroutines; i++ {
		go func() {
			<-started
			for j := 0; j < opsPerGoroutine; j++ {
				// Just verify we can read without panicking
				_ = tile.IsHighlighted()
			}
			done <- struct{}{}
		}()
	}

	close(started)

	for i := 0; i < numGoroutines*2; i++ {
		<-done
	}
}

// =============================================================================
// Nil Clock Safety
// =============================================================================

func TestNilClockFallback(t *testing.T) {
	// Use newWithClock with nil clock
	tile := newWithClock("/path/to/file.go", "file.go", false, nil)

	// Operations should not panic with nil clock
	tile.Reveal()
	tile.Highlight(100 * time.Millisecond)

	// Verify operations completed
	if tile.RevealCount() != 1 {
		t.Errorf("expected revealCount 1 with nil clock, got %d", tile.RevealCount())
	}
	if tile.LastRevealed().IsZero() {
		t.Error("expected non-zero lastRevealed with nil clock fallback")
	}
}
