package tile

import (
	"testing"
	"time"
)

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
	tile := New("/path/to/file.go", "file.go", false)

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
	time.Sleep(150 * time.Millisecond)
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
