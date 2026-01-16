package tile

import (
	"path/filepath"
	"sync"
	"time"
)

// FogState represents the fog of war state for a tile
type FogState int

const (
	FogFull     FogState = iota // Completely fogged (not read by Claude)
	FogStale                    // Previously read but may be outdated (context summarized)
	FogRevealed                 // Currently in Claude's context
)

// Tile represents a file or directory in the codebase
type Tile struct {
	mu sync.RWMutex

	// Identity
	path      string // Absolute path
	relPath   string // Relative path from project root
	isDir     bool
	name      string
	extension string

	// Fog of war state
	fogState     FogState
	lastRevealed time.Time
	revealCount  int // How many times Claude has read this file
	lastModified time.Time
	sizeBytes    int64

	// Animation state
	highlightUntil time.Time // For temporary highlights on edits
}

// New creates a new tile
func New(path, relPath string, isDir bool) *Tile {
	name := filepath.Base(path)
	ext := ""
	if !isDir {
		ext = filepath.Ext(name)
	}

	return &Tile{
		path:      path,
		relPath:   relPath,
		isDir:     isDir,
		name:      name,
		extension: ext,
		fogState:  FogFull,
	}
}

// Path returns the absolute path
func (t *Tile) Path() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.path
}

// RelPath returns the relative path
func (t *Tile) RelPath() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.relPath
}

// IsDirectory returns true if this tile represents a directory
func (t *Tile) IsDirectory() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isDir
}

// Name returns the file/directory name
func (t *Tile) Name() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.name
}

// Extension returns the file extension (empty for directories)
func (t *Tile) Extension() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.extension
}

// FogState returns the current fog of war state
func (t *Tile) FogState() FogState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.fogState
}

// IsRevealed returns true if the tile is currently revealed (not fogged)
func (t *Tile) IsRevealed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.fogState == FogRevealed
}

// IsStale returns true if the tile was revealed but is now stale
func (t *Tile) IsStale() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.fogState == FogStale
}

// Reveal marks the tile as revealed (fog cleared)
// Called when Claude reads this file
func (t *Tile) Reveal() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fogState = FogRevealed
	t.lastRevealed = time.Now()
	t.revealCount++
}

// MarkStale marks the tile as stale (was revealed but now outdated)
// Called when context summarization occurs
func (t *Tile) MarkStale() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fogState == FogRevealed {
		t.fogState = FogStale
	}
}

// ResetFog marks the tile as completely fogged again
func (t *Tile) ResetFog() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.fogState = FogFull
}

// Highlight temporarily highlights the tile (for edits/changes)
func (t *Tile) Highlight(duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.highlightUntil = time.Now().Add(duration)
}

// IsHighlighted returns true if the tile should be highlighted
func (t *Tile) IsHighlighted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Now().Before(t.highlightUntil)
}

// LastRevealed returns when the tile was last revealed
func (t *Tile) LastRevealed() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastRevealed
}

// RevealCount returns how many times this tile has been revealed
func (t *Tile) RevealCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.revealCount
}

// UpdateMetadata updates tile metadata (size, modified time)
func (t *Tile) UpdateMetadata(sizeBytes int64, modTime time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.sizeBytes = sizeBytes
	t.lastModified = modTime
}

// SizeBytes returns the file size in bytes
func (t *Tile) SizeBytes() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sizeBytes
}

// LastModified returns when the file was last modified
func (t *Tile) LastModified() time.Time {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.lastModified
}
