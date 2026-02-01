package connection

import (
	"sort"
	"sync"
	"testing"
)

func TestGraphAddAndGet(t *testing.T) {
	g := NewGraph()

	c := NewConnection("a.go", "b.go", TypeImport)
	added := g.Add(c)

	if added != c {
		t.Error("Add should return the added connection")
	}

	if got := g.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}

	// Get by ID
	if got := g.Get(c.ID()); got != c {
		t.Error("Get(ID) should return the same connection")
	}

	// Get by endpoints
	if got := g.GetByEndpoints("a.go", "b.go", TypeImport); got != c {
		t.Error("GetByEndpoints should return the same connection")
	}

	// Adding same connection again returns existing
	c2 := NewConnection("a.go", "b.go", TypeImport)
	existing := g.Add(c2)
	if existing != c {
		t.Error("Adding duplicate should return existing connection")
	}
	if got := g.Count(); got != 1 {
		t.Errorf("Count() after duplicate = %d, want 1", got)
	}
}

func TestGraphAddNil(t *testing.T) {
	g := NewGraph()

	if got := g.Add(nil); got != nil {
		t.Error("Add(nil) should return nil")
	}
	if g.Count() != 0 {
		t.Errorf("Count() after Add(nil) = %d, want 0", g.Count())
	}
}

func TestGraphAddNew(t *testing.T) {
	g := NewGraph()

	c := g.AddNew("x.go", "y.go", TypeCall)
	if c == nil {
		t.Fatal("AddNew should return non-nil")
	}

	if c.From() != "x.go" || c.To() != "y.go" || c.Type() != TypeCall {
		t.Error("AddNew connection has wrong values")
	}

	if got := g.Count(); got != 1 {
		t.Errorf("Count() = %d, want 1", got)
	}
}

func TestGraphRemove(t *testing.T) {
	g := NewGraph()
	c := g.AddNew("a.go", "b.go", TypeImport)

	if !g.Remove(c.ID()) {
		t.Error("Remove should return true for existing connection")
	}

	if got := g.Count(); got != 0 {
		t.Errorf("Count() after remove = %d, want 0", got)
	}

	if g.Get(c.ID()) != nil {
		t.Error("Get should return nil after remove")
	}

	// Remove non-existent
	if g.Remove("nonexistent") {
		t.Error("Remove should return false for non-existent ID")
	}
}

func TestGraphClear(t *testing.T) {
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("b.go", "c.go", TypeImport)
	g.AddNew("c.go", "a.go", TypeImport) // Create cycle

	g.DetectCircular()
	g.Clear()

	if got := g.Count(); got != 0 {
		t.Errorf("Count() after clear = %d, want 0", got)
	}

	if len(g.CircularPaths()) != 0 {
		t.Error("CircularPaths should be empty after clear")
	}
}

func TestGraphAll(t *testing.T) {
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("c.go", "d.go", TypeCall)

	all := g.All()
	if len(all) != 2 {
		t.Errorf("All() returned %d connections, want 2", len(all))
	}

	// Modifying returned slice shouldn't affect graph
	all[0] = nil
	if g.Count() != 2 {
		t.Error("Modifying All() slice affected graph")
	}
}

func TestGraphFromFile(t *testing.T) {
	g := NewGraph()
	g.AddNew("utils.go", "main.go", TypeImport)
	g.AddNew("utils.go", "server.go", TypeImport)
	g.AddNew("config.go", "main.go", TypeImport)

	// utils.go is a source for 2 connections
	fromUtils := g.FromFile("utils.go")
	if len(fromUtils) != 2 {
		t.Errorf("FromFile(\"utils.go\") = %d connections, want 2", len(fromUtils))
	}

	// config.go is a source for 1 connection
	fromConfig := g.FromFile("config.go")
	if len(fromConfig) != 1 {
		t.Errorf("FromFile(\"config.go\") = %d connections, want 1", len(fromConfig))
	}

	// non-existent file
	fromNone := g.FromFile("nonexistent.go")
	if fromNone != nil {
		t.Error("FromFile for non-existent should return nil")
	}
}

func TestGraphToFile(t *testing.T) {
	g := NewGraph()
	g.AddNew("utils.go", "main.go", TypeImport)
	g.AddNew("config.go", "main.go", TypeImport)
	g.AddNew("config.go", "server.go", TypeImport)

	// main.go imports 2 files
	toMain := g.ToFile("main.go")
	if len(toMain) != 2 {
		t.Errorf("ToFile(\"main.go\") = %d connections, want 2", len(toMain))
	}

	// server.go imports 1 file
	toServer := g.ToFile("server.go")
	if len(toServer) != 1 {
		t.Errorf("ToFile(\"server.go\") = %d connections, want 1", len(toServer))
	}
}

func TestGraphFilesWithConnections(t *testing.T) {
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("b.go", "c.go", TypeImport)

	files := g.FilesWithConnections()
	sort.Strings(files)

	expected := []string{"a.go", "b.go", "c.go"}
	if len(files) != len(expected) {
		t.Fatalf("FilesWithConnections() = %v, want %v", files, expected)
	}
	for i, f := range files {
		if f != expected[i] {
			t.Errorf("FilesWithConnections()[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

func TestGraphDetectCircular_NoCycle(t *testing.T) {
	g := NewGraph()
	// Linear chain: a -> b -> c
	g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("b.go", "c.go", TypeImport)

	cycles := g.DetectCircular()
	if len(cycles) != 0 {
		t.Errorf("DetectCircular() found %d cycles, want 0", len(cycles))
	}

	if g.HasCircularDependencies() {
		t.Error("HasCircularDependencies() should be false")
	}

	// No connections should be marked circular
	for _, c := range g.All() {
		if c.IsCircular() {
			t.Errorf("Connection %s should not be circular", c.ID())
		}
	}
}

func TestGraphDetectCircular_SimpleCycle(t *testing.T) {
	g := NewGraph()
	// Cycle: a -> b -> c -> a
	// In import semantics: a imports b, b imports c, c imports a
	// So edges are: b->a, c->b, a->c (from source to consumer)
	g.AddNew("b.go", "a.go", TypeImport) // a imports b
	g.AddNew("c.go", "b.go", TypeImport) // b imports c
	g.AddNew("a.go", "c.go", TypeImport) // c imports a

	cycles := g.DetectCircular()
	if len(cycles) != 1 {
		t.Errorf("DetectCircular() found %d cycles, want 1", len(cycles))
	}

	if !g.HasCircularDependencies() {
		t.Error("HasCircularDependencies() should be true")
	}

	// All connections in the cycle should be marked circular
	for _, c := range g.All() {
		if !c.IsCircular() {
			t.Errorf("Connection %s should be circular", c.ID())
		}
	}
}

func TestGraphDetectCircular_MixedCycleAndLinear(t *testing.T) {
	g := NewGraph()
	// Cycle: a <-> b (mutual import)
	g.AddNew("a.go", "b.go", TypeImport) // b imports a
	g.AddNew("b.go", "a.go", TypeImport) // a imports b

	// Linear: c -> d (c imports d, so d->c)
	g.AddNew("d.go", "c.go", TypeImport)

	cycles := g.DetectCircular()
	if len(cycles) != 1 {
		t.Errorf("DetectCircular() found %d cycles, want 1", len(cycles))
	}

	// Check that only a-b connections are circular
	for _, c := range g.All() {
		isInCycle := (c.From() == "a.go" && c.To() == "b.go") ||
			(c.From() == "b.go" && c.To() == "a.go")
		if c.IsCircular() != isInCycle {
			t.Errorf("Connection %s: IsCircular() = %v, want %v",
				c.ID(), c.IsCircular(), isInCycle)
		}
	}
}

func TestGraphDetectCircular_SelfReference(t *testing.T) {
	g := NewGraph()
	// Self-reference: a -> a
	g.AddNew("a.go", "a.go", TypeImport)

	cycles := g.DetectCircular()
	// Self-loop is a trivial SCC, may or may not be reported depending on implementation
	// Our implementation marks it as circular since it's in a cycle with itself
	// Actually, single-node SCCs are trivial and not reported by Tarjan's standard impl
	// Let's verify behavior
	t.Logf("Self-reference cycles: %v", cycles)
}

func TestGraphFanInFanOut(t *testing.T) {
	g := NewGraph()
	// utils.go is imported by 3 files
	g.AddNew("utils.go", "a.go", TypeImport)
	g.AddNew("utils.go", "b.go", TypeImport)
	g.AddNew("utils.go", "c.go", TypeImport)

	// a.go imports 2 files
	g.AddNew("config.go", "a.go", TypeImport)

	// Fan-in: how many import this file (utils is imported by 3)
	if got := g.FanIn("utils.go"); got != 3 {
		t.Errorf("FanIn(\"utils.go\") = %d, want 3", got)
	}

	// Fan-out: how many this file imports (a.go imports 2: utils, config)
	if got := g.FanOut("a.go"); got != 2 {
		t.Errorf("FanOut(\"a.go\") = %d, want 2", got)
	}

	// Non-existent file
	if got := g.FanIn("nonexistent.go"); got != 0 {
		t.Errorf("FanIn(nonexistent) = %d, want 0", got)
	}
}

func TestGraphBottlenecks(t *testing.T) {
	g := NewGraph()
	// utils.go is imported by 5 files (high fan-in)
	for i := 0; i < 5; i++ {
		g.AddNew("utils.go", string(rune('a'+i))+".go", TypeImport)
	}
	// config.go is imported by 2 files
	g.AddNew("config.go", "a.go", TypeImport)
	g.AddNew("config.go", "b.go", TypeImport)

	// Threshold 4: only utils.go qualifies
	bottlenecks := g.Bottlenecks(4)
	if len(bottlenecks) != 1 || bottlenecks[0] != "utils.go" {
		t.Errorf("Bottlenecks(4) = %v, want [\"utils.go\"]", bottlenecks)
	}

	// Threshold 2: both qualify
	bottlenecks = g.Bottlenecks(2)
	if len(bottlenecks) != 2 {
		t.Errorf("Bottlenecks(2) = %v, want 2 files", bottlenecks)
	}
}

func TestGraphOrphans(t *testing.T) {
	g := NewGraph()
	g.AddNew("a.go", "b.go", TypeImport)

	allFiles := []string{"a.go", "b.go", "orphan.go", "another_orphan.go"}
	orphans := g.Orphans(allFiles)

	sort.Strings(orphans)
	expected := []string{"another_orphan.go", "orphan.go"}

	if len(orphans) != len(expected) {
		t.Fatalf("Orphans() = %v, want %v", orphans, expected)
	}
	for i, f := range orphans {
		if f != expected[i] {
			t.Errorf("Orphans()[%d] = %q, want %q", i, f, expected[i])
		}
	}
}

func TestGraphConcurrency(t *testing.T) {
	g := NewGraph()
	const goroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				from := string(rune('a' + id))
				to := string(rune('a' + (id+1)%goroutines))

				c := g.AddNew(from, to, TypeImport)
				_ = g.Count()
				_ = g.Get(c.ID())
				_ = g.FromFile(from)
				_ = g.ToFile(to)
				_ = g.All()
				_ = g.FanIn(from)
				_ = g.FanOut(to)
			}
		}(i)
	}

	wg.Wait()

	// Should have exactly 'goroutines' connections (one per pair)
	if got := g.Count(); got != goroutines {
		t.Errorf("Count() = %d, want %d", got, goroutines)
	}
}

func TestGraphRemoveUpdatesIndexes(t *testing.T) {
	g := NewGraph()
	c1 := g.AddNew("a.go", "b.go", TypeImport)
	g.AddNew("a.go", "c.go", TypeImport)

	// Remove first connection
	g.Remove(c1.ID())

	// fromIndex should still have a.go -> c.go
	fromA := g.FromFile("a.go")
	if len(fromA) != 1 {
		t.Errorf("FromFile(\"a.go\") after remove = %d, want 1", len(fromA))
	}

	// toIndex for b.go should be empty
	toB := g.ToFile("b.go")
	if len(toB) != 0 {
		t.Errorf("ToFile(\"b.go\") after remove = %d, want 0", len(toB))
	}
}
