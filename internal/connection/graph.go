package connection

import (
	"sync"
)

// Graph manages a collection of connections representing the dependency graph.
//
// Thread-safe: All methods are safe for concurrent access.
//
// The graph provides:
// - Fast lookup by connection ID, from path, or to path
// - Circular dependency detection
// - Graph traversal utilities
type Graph struct {
	mu sync.RWMutex

	// All connections indexed by ID
	connections map[string]*Connection

	// Indexes for fast lookup
	byFrom map[string][]*Connection // Connections where this file is the source
	byTo   map[string][]*Connection // Connections where this file is the consumer

	// Cached analysis results
	circularPaths [][]string // Detected circular dependency chains
	analysisValid bool       // Whether cached analysis is current
}

// NewGraph creates an empty dependency graph.
func NewGraph() *Graph {
	return &Graph{
		connections: make(map[string]*Connection),
		byFrom:      make(map[string][]*Connection),
		byTo:        make(map[string][]*Connection),
	}
}

// Add adds a connection to the graph.
//
// If a connection with the same ID already exists, this is a no-op.
// To update an existing connection, use Get and modify it directly.
//
// Returns the connection (existing or new).
func (g *Graph) Add(c *Connection) *Connection {
	g.mu.Lock()
	defer g.mu.Unlock()

	id := c.ID()
	if existing, ok := g.connections[id]; ok {
		return existing
	}

	g.connections[id] = c
	g.byFrom[c.from] = append(g.byFrom[c.from], c)
	g.byTo[c.to] = append(g.byTo[c.to], c)
	g.analysisValid = false

	return c
}

// AddNew creates and adds a new connection.
// Convenience method combining NewConnection and Add.
func (g *Graph) AddNew(from, to string, typ Type) *Connection {
	c := NewConnection(from, to, typ)
	return g.Add(c)
}

// Get returns a connection by ID, or nil if not found.
func (g *Graph) Get(id string) *Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.connections[id]
}

// GetByEndpoints returns a connection by its endpoints and type.
func (g *Graph) GetByEndpoints(from, to string, typ Type) *Connection {
	id := makeID(from, to, typ)
	return g.Get(id)
}

// Remove removes a connection from the graph.
// Returns true if the connection existed and was removed.
func (g *Graph) Remove(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	c, ok := g.connections[id]
	if !ok {
		return false
	}

	delete(g.connections, id)
	g.removeFromIndex(g.byFrom, c.from, c)
	g.removeFromIndex(g.byTo, c.to, c)
	g.analysisValid = false

	return true
}

// removeFromIndex removes a connection from an index slice.
func (g *Graph) removeFromIndex(index map[string][]*Connection, key string, c *Connection) {
	conns := index[key]
	if len(conns) == 0 {
		return // Shouldn't happen, but be defensive
	}
	for i, conn := range conns {
		if conn.ID() == c.ID() {
			// Remove by swapping with last and truncating
			conns[i] = conns[len(conns)-1]
			index[key] = conns[:len(conns)-1]
			if len(index[key]) == 0 {
				delete(index, key)
			}
			return
		}
	}
}

// Clear removes all connections from the graph.
func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.connections = make(map[string]*Connection)
	g.byFrom = make(map[string][]*Connection)
	g.byTo = make(map[string][]*Connection)
	g.circularPaths = nil
	g.analysisValid = false
}

// Count returns the number of connections in the graph.
func (g *Graph) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.connections)
}

// All returns all connections as a slice.
// The returned slice is a copy; modifying it doesn't affect the graph.
func (g *Graph) All() []*Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make([]*Connection, 0, len(g.connections))
	for _, c := range g.connections {
		result = append(result, c)
	}
	return result
}

// FromFile returns all connections where the given file is the source (provider).
// These are the files that consume/import from this file.
func (g *Graph) FromFile(path string) []*Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	conns := g.byFrom[path]
	if conns == nil {
		return nil
	}

	// Return a copy
	result := make([]*Connection, len(conns))
	copy(result, conns)
	return result
}

// ToFile returns all connections where the given file is the destination (consumer).
// These are the files that this file imports/depends on.
func (g *Graph) ToFile(path string) []*Connection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	conns := g.byTo[path]
	if conns == nil {
		return nil
	}

	// Return a copy
	result := make([]*Connection, len(conns))
	copy(result, conns)
	return result
}

// FilesWithConnections returns all file paths that have at least one connection.
func (g *Graph) FilesWithConnections() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	seen := make(map[string]bool)
	for path := range g.byFrom {
		seen[path] = true
	}
	for path := range g.byTo {
		seen[path] = true
	}

	result := make([]string, 0, len(seen))
	for path := range seen {
		result = append(result, path)
	}
	return result
}

// DetectCircular performs cycle detection and marks connections that are part
// of circular dependency chains.
//
// This uses Tarjan's algorithm for finding strongly connected components.
// Any connection within a non-trivial SCC is marked as circular.
//
// Returns the list of circular paths found.
func (g *Graph) DetectCircular() [][]string {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Build adjacency list from connections
	adjacency := make(map[string][]string)
	for _, c := range g.connections {
		// Direction: from (source) -> to (consumer)
		// For cycle detection, we follow the import direction: to imports from
		// So the edge is: to -> from (consumer depends on source)
		adjacency[c.to] = append(adjacency[c.to], c.from)
	}

	// Reset all connections to non-circular first
	for _, c := range g.connections {
		c.SetCircular(false)
	}

	// Find all strongly connected components using Tarjan's algorithm
	sccs := tarjanSCC(adjacency)

	// Mark connections in non-trivial SCCs as circular
	var circularPaths [][]string
	for _, scc := range sccs {
		if len(scc) > 1 {
			// Non-trivial SCC = circular dependency
			circularPaths = append(circularPaths, scc)

			// Create set for fast lookup
			inSCC := make(map[string]bool, len(scc))
			for _, node := range scc {
				inSCC[node] = true
			}

			// Mark connections between nodes in this SCC as circular
			for _, c := range g.connections {
				if inSCC[c.from] && inSCC[c.to] {
					c.SetCircular(true)
				}
			}
		}
	}

	g.circularPaths = circularPaths
	g.analysisValid = true
	return circularPaths
}

// CircularPaths returns cached circular dependency paths.
// Call DetectCircular() first to populate this.
func (g *Graph) CircularPaths() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.circularPaths
}

// HasCircularDependencies returns true if the graph contains any circular dependencies.
// Call DetectCircular() first.
func (g *Graph) HasCircularDependencies() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.circularPaths) > 0
}

// tarjanSCC implements Tarjan's strongly connected components algorithm.
// Returns a list of SCCs, where each SCC is a list of node names.
func tarjanSCC(adjacency map[string][]string) [][]string {
	var (
		index    = 0
		stack    []string
		onStack  = make(map[string]bool)
		indices  = make(map[string]int)
		lowlinks = make(map[string]int)
		sccs     [][]string
	)

	var strongConnect func(v string)
	strongConnect = func(v string) {
		indices[v] = index
		lowlinks[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range adjacency[v] {
			if _, visited := indices[w]; !visited {
				strongConnect(w)
				if lowlinks[w] < lowlinks[v] {
					lowlinks[v] = lowlinks[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlinks[v] {
					lowlinks[v] = indices[w]
				}
			}
		}

		// If v is a root node, pop the stack and generate an SCC
		if lowlinks[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	// Visit all nodes
	for v := range adjacency {
		if _, visited := indices[v]; !visited {
			strongConnect(v)
		}
	}

	// Also visit nodes that only appear as destinations
	allNodes := make(map[string]bool)
	for v := range adjacency {
		allNodes[v] = true
		for _, w := range adjacency[v] {
			allNodes[w] = true
		}
	}
	for v := range allNodes {
		if _, visited := indices[v]; !visited {
			strongConnect(v)
		}
	}

	return sccs
}

// FanIn returns the number of files that depend on (import) the given file.
func (g *Graph) FanIn(path string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.byFrom[path])
}

// FanOut returns the number of files that the given file depends on (imports).
func (g *Graph) FanOut(path string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.byTo[path])
}

// Bottlenecks returns files with high fan-in (many dependents).
// threshold specifies the minimum fan-in to be considered a bottleneck.
func (g *Graph) Bottlenecks(threshold int) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var bottlenecks []string
	for path, conns := range g.byFrom {
		if len(conns) >= threshold {
			bottlenecks = append(bottlenecks, path)
		}
	}
	return bottlenecks
}

// Orphans returns files with no incoming or outgoing connections.
// These are isolated modules with no dependencies.
func (g *Graph) Orphans(allFiles []string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	connected := make(map[string]bool)
	for path := range g.byFrom {
		connected[path] = true
	}
	for path := range g.byTo {
		connected[path] = true
	}

	var orphans []string
	for _, file := range allFiles {
		if !connected[file] {
			orphans = append(orphans, file)
		}
	}
	return orphans
}
