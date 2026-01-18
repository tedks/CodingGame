// Package connection manages dependency relationships between files in the codebase.
// These connections are visualized as Factorio-style "belts" showing data flow.
//
// Direction Convention:
// - From = source/provider (the file being imported)
// - To = consumer/importer (the file doing the importing)
// - Data flows FROM source TO consumer
//
// Example: if foo.go imports bar.go, the connection is:
//
//	From: bar.go (provides symbols)
//	To:   foo.go (consumes/imports)
//
// This matches Factorio semantics where items flow from producers to consumers.
package connection

import (
	"sync"
	"time"
)

// Type categorizes the nature of a dependency relationship.
type Type int

const (
	// TypeImport represents a direct import/include statement.
	// E.g., `import "foo"` or `#include "foo.h"`
	TypeImport Type = iota

	// TypeInheritance represents class/type inheritance.
	// E.g., `class Foo extends Bar`
	TypeInheritance

	// TypeComposition represents composition/aggregation.
	// E.g., struct field of another type, function using types from another file
	TypeComposition

	// TypeCall represents a function/method call relationship.
	// E.g., file A calls function defined in file B
	TypeCall

	// TypeUnknown is used when the relationship type can't be determined.
	TypeUnknown
)

// String returns a human-readable name for the connection type.
func (t Type) String() string {
	switch t {
	case TypeImport:
		return "import"
	case TypeInheritance:
		return "inheritance"
	case TypeComposition:
		return "composition"
	case TypeCall:
		return "call"
	default:
		return "unknown"
	}
}

// Connection represents a dependency relationship between two files.
// It is the data model; visual representation is handled by the belt renderer.
//
// Thread-safe: All methods are safe for concurrent access.
type Connection struct {
	mu sync.RWMutex

	// Identity - these fields are immutable after creation
	id   string // Unique identifier (from:to:type)
	from string // Source file path (provider) - relative to project root
	to   string // Destination file path (consumer) - relative to project root
	typ  Type   // Type of relationship

	// Metrics - may be updated over time
	strength      int       // Coupling strength (number of symbols used across this connection)
	exerciseCount int       // How many times this connection has been traversed (for animation speed)
	lastExercised time.Time // When this connection was last traversed

	// Analysis state
	isCircular bool // Part of a circular dependency chain
	isExternal bool // Target is outside the project (e.g., standard library)
}

// NewConnection creates a new connection between two files.
//
// Parameters:
//   - from: relative path of the source file (provider)
//   - to: relative path of the destination file (consumer)
//   - typ: the type of dependency relationship
//
// The connection ID is derived from the paths and type, making it stable
// for lookups and deduplication.
//
// Edge cases:
//   - from == to (self-reference): allowed but flagged
//   - empty paths: will result in invalid connection, caller should validate
func NewConnection(from, to string, typ Type) *Connection {
	// Normalize paths to use forward slashes for consistency
	// (filepath.ToSlash is a no-op on Unix, so we do explicit replacement)
	from = normalizePath(from)
	to = normalizePath(to)

	return &Connection{
		id:       makeID(from, to, typ),
		from:     from,
		to:       to,
		typ:      typ,
		strength: 1, // Default minimum strength
	}
}

// makeID creates a stable unique identifier for a connection.
func makeID(from, to string, typ Type) string {
	return from + ":" + to + ":" + typ.String()
}

// normalizePath converts backslashes to forward slashes for cross-platform consistency.
// This is needed because filepath.ToSlash is a no-op on Unix systems.
func normalizePath(p string) string {
	// Simple byte-by-byte replacement; strings.ReplaceAll would work too
	// but this avoids an import for a trivial operation
	result := make([]byte, len(p))
	for i := 0; i < len(p); i++ {
		if p[i] == '\\' {
			result[i] = '/'
		} else {
			result[i] = p[i]
		}
	}
	return string(result)
}

// ID returns the unique identifier for this connection.
func (c *Connection) ID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.id
}

// From returns the source file path (provider).
func (c *Connection) From() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.from
}

// To returns the destination file path (consumer).
func (c *Connection) To() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.to
}

// Type returns the connection type.
func (c *Connection) Type() Type {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.typ
}

// Strength returns the coupling strength (number of symbols used).
func (c *Connection) Strength() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.strength
}

// SetStrength updates the coupling strength.
//
// Assumptions:
//   - strength >= 0; negative values are clamped to 0
//   - 0 strength means minimal coupling (1 symbol)
func (c *Connection) SetStrength(strength int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strength < 0 {
		strength = 0
	}
	c.strength = strength
}

// ExerciseCount returns how many times this connection has been traversed.
func (c *Connection) ExerciseCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.exerciseCount
}

// Exercise marks this connection as having been traversed.
// This is called when code execution passes through this dependency path.
func (c *Connection) Exercise() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.exerciseCount++
	c.lastExercised = time.Now()
}

// LastExercised returns when this connection was last traversed.
func (c *Connection) LastExercised() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastExercised
}

// IsCircular returns true if this connection is part of a circular dependency.
func (c *Connection) IsCircular() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCircular
}

// SetCircular marks this connection as part of a circular dependency.
func (c *Connection) SetCircular(circular bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isCircular = circular
}

// IsExternal returns true if the target is outside the project.
func (c *Connection) IsExternal() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isExternal
}

// SetExternal marks this connection as pointing to an external dependency.
func (c *Connection) SetExternal(external bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isExternal = external
}

// IsSelfReference returns true if from and to are the same file.
func (c *Connection) IsSelfReference() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.from == c.to
}

// Endpoints returns both file paths as a convenience.
func (c *Connection) Endpoints() (from, to string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.from, c.to
}
