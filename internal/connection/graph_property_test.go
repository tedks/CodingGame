package connection

import (
	"math/rand"
	"testing"
)

// TestGraphInvariantsAfterRandomOps verifies graph invariants hold after random operations.
func TestGraphInvariantsAfterRandomOps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property test in short mode")
	}

	rng := rand.New(rand.NewSource(42)) // Deterministic for reproducibility
	g := NewGraph()

	files := []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go", "h.go"}
	types := []Type{TypeImport, TypeCall, TypeComposition, TypeInheritance}

	// Track what we've added for verification
	addedIDs := make(map[string]bool)

	const iterations = 1000
	for i := 0; i < iterations; i++ {
		// Random operation: 70% add, 20% remove, 10% clear
		op := rng.Intn(100)

		switch {
		case op < 70: // Add
			from := files[rng.Intn(len(files))]
			to := files[rng.Intn(len(files))]
			typ := types[rng.Intn(len(types))]
			c := g.AddNew(from, to, typ)
			addedIDs[c.ID()] = true

		case op < 90: // Remove
			if len(addedIDs) > 0 {
				// Pick a random ID to remove
				var id string
				for id = range addedIDs {
					break
				}
				g.Remove(id)
				delete(addedIDs, id)
			}

		default: // Clear
			g.Clear()
			addedIDs = make(map[string]bool)
		}

		// Verify invariants after each operation
		verifyGraphInvariants(t, g, i)
	}
}

// verifyGraphInvariants checks that all graph invariants hold.
func verifyGraphInvariants(t *testing.T, g *Graph, iteration int) {
	t.Helper()

	// Invariant 1: Count() == len(All())
	count := g.Count()
	all := g.All()
	if count != len(all) {
		t.Errorf("Iteration %d: Count()=%d != len(All())=%d", iteration, count, len(all))
	}

	// Invariant 2: FanIn(x) == len(FromFile(x)) for all files
	// Invariant 3: FanOut(x) == len(ToFile(x)) for all files
	seen := make(map[string]bool)
	for _, c := range all {
		seen[c.From()] = true
		seen[c.To()] = true
	}

	for file := range seen {
		fanIn := g.FanIn(file)
		fromFile := g.FromFile(file)
		if fanIn != len(fromFile) {
			t.Errorf("Iteration %d: FanIn(%q)=%d != len(FromFile())=%d",
				iteration, file, fanIn, len(fromFile))
		}

		fanOut := g.FanOut(file)
		toFile := g.ToFile(file)
		if fanOut != len(toFile) {
			t.Errorf("Iteration %d: FanOut(%q)=%d != len(ToFile())=%d",
				iteration, file, fanOut, len(toFile))
		}
	}

	// Invariant 4: Get(c.ID()) == c for all connections in All()
	for _, c := range all {
		got := g.Get(c.ID())
		if got != c {
			t.Errorf("Iteration %d: Get(%q) returned different connection", iteration, c.ID())
		}
	}

	// Invariant 5: GetByEndpoints matches Get for all connections
	for _, c := range all {
		got := g.GetByEndpoints(c.From(), c.To(), c.Type())
		if got != c {
			t.Errorf("Iteration %d: GetByEndpoints(%q, %q, %v) != Get(%q)",
				iteration, c.From(), c.To(), c.Type(), c.ID())
		}
	}

	// Invariant 6: FilesWithConnections contains all from/to files
	filesWithConns := make(map[string]bool)
	for _, f := range g.FilesWithConnections() {
		filesWithConns[f] = true
	}
	for file := range seen {
		if !filesWithConns[file] {
			t.Errorf("Iteration %d: file %q has connections but not in FilesWithConnections()",
				iteration, file)
		}
	}
}

// TestAddThenRemoveLeavesGraphUnchanged verifies add-then-remove is a no-op.
func TestAddThenRemoveLeavesGraphUnchanged(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping property test in short mode")
	}

	// Start with a graph with some connections
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("b.go", "c.go", TypeImport)
	g.AddNew("c.go", "d.go", TypeCall)

	// Capture initial state
	initialCount := g.Count()
	initialIDs := make(map[string]bool)
	for _, c := range g.All() {
		initialIDs[c.ID()] = true
	}

	// Add a new connection and immediately remove it
	newConn := g.AddNew("x.go", "y.go", TypeComposition)
	newID := newConn.ID()

	// Verify it was added
	if g.Count() != initialCount+1 {
		t.Error("Connection was not added")
	}
	if g.Get(newID) == nil {
		t.Error("New connection not found by Get")
	}

	// Remove it
	if !g.Remove(newID) {
		t.Error("Remove returned false for just-added connection")
	}

	// Verify graph is back to initial state
	if g.Count() != initialCount {
		t.Errorf("Count changed: initial=%d, after add+remove=%d", initialCount, g.Count())
	}

	// Verify same connections exist
	for _, c := range g.All() {
		if !initialIDs[c.ID()] {
			t.Errorf("Unexpected connection after add+remove: %s", c.ID())
		}
	}
	for id := range initialIDs {
		if g.Get(id) == nil {
			t.Errorf("Original connection missing after add+remove: %s", id)
		}
	}

	// Indexes should be clean
	if g.FromFile("x.go") != nil {
		t.Error("FromFile index not cleaned up after remove")
	}
	if g.ToFile("y.go") != nil {
		t.Error("ToFile index not cleaned up after remove")
	}
}

// TestClearThenRepopulate verifies clear followed by re-add works correctly.
func TestClearThenRepopulate(t *testing.T) {
	g := NewGraph()

	// Build initial graph
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("b.go", "c.go", TypeImport)
	g.AddNew("c.go", "a.go", TypeImport) // Creates cycle

	g.DetectCircular()
	if !g.HasCircularDependencies() {
		t.Fatal("Expected circular dependencies")
	}

	// Clear
	g.Clear()

	if g.Count() != 0 {
		t.Errorf("Count after clear = %d, want 0", g.Count())
	}
	if g.HasCircularDependencies() {
		t.Error("HasCircularDependencies should be false after clear")
	}

	// Repopulate with different structure (no cycle)
	g.AddNew("x.go", "y.go", TypeImport)
	g.AddNew("y.go", "z.go", TypeImport)

	g.DetectCircular()
	if g.HasCircularDependencies() {
		t.Error("New linear graph should not have circular dependencies")
	}

	// Verify graph is correctly populated
	if g.Count() != 2 {
		t.Errorf("Count after repopulate = %d, want 2", g.Count())
	}
	if g.Get(g.GetByEndpoints("x.go", "y.go", TypeImport).ID()) == nil {
		t.Error("New connection not found")
	}
}

// TestDuplicateAddIsIdempotent verifies adding same connection twice is no-op.
func TestDuplicateAddIsIdempotent(t *testing.T) {
	g := NewGraph()

	// Add connection
	c1 := g.AddNew("a.go", "b.go", TypeImport)
	count1 := g.Count()
	fanIn1 := g.FanIn("a.go")
	fanOut1 := g.FanOut("b.go")

	// Add same connection again (different Connection object, same ID)
	c2 := g.AddNew("a.go", "b.go", TypeImport)

	// Should return original connection
	if c1 != c2 {
		t.Error("Duplicate add should return original connection")
	}

	// Count should not change
	if g.Count() != count1 {
		t.Errorf("Count changed after duplicate add: %d -> %d", count1, g.Count())
	}

	// FanIn/FanOut should not change
	if g.FanIn("a.go") != fanIn1 {
		t.Error("FanIn changed after duplicate add")
	}
	if g.FanOut("b.go") != fanOut1 {
		t.Error("FanOut changed after duplicate add")
	}

	// All() should return same connections
	all := g.All()
	if len(all) != 1 {
		t.Errorf("All() returned %d connections, want 1", len(all))
	}
}

// TestRemoveNonexistentIsNoOp verifies removing non-existent ID is safe.
func TestRemoveNonexistentIsNoOp(t *testing.T) {
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)

	initialCount := g.Count()

	// Remove non-existent
	removed := g.Remove("nonexistent-id")
	if removed {
		t.Error("Remove should return false for non-existent ID")
	}

	// Graph unchanged
	if g.Count() != initialCount {
		t.Error("Count changed after removing non-existent ID")
	}

	// Multiple removes of non-existent are safe
	for i := 0; i < 100; i++ {
		g.Remove("nonexistent-" + string(rune('0'+i)))
	}
	if g.Count() != initialCount {
		t.Error("Count changed after multiple non-existent removes")
	}
}
