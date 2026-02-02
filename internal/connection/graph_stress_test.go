package connection

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// TestMassiveGraph tests a graph with 10K nodes and 100K edges.
func TestMassiveGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping massive graph test in short mode")
	}

	g := NewGraph()
	const nodes = 10000
	const edges = 100000

	// Create edges randomly
	rng := rand.New(rand.NewSource(42))
	for i := 0; i < edges; i++ {
		from := fmt.Sprintf("file%d.go", rng.Intn(nodes))
		to := fmt.Sprintf("file%d.go", rng.Intn(nodes))
		typ := Type(rng.Intn(4))
		g.AddNew(from, to, typ)
	}

	// Count will be less than edges due to duplicates
	count := g.Count()
	t.Logf("Created graph with %d unique connections from %d edge attempts", count, edges)

	if count == 0 {
		t.Error("Graph should have some connections")
	}

	// Verify basic operations work
	all := g.All()
	if len(all) != count {
		t.Errorf("All() returned %d, Count() returned %d", len(all), count)
	}

	// DetectCircular should complete without error
	cycles := g.DetectCircular()
	t.Logf("Found %d circular dependency cycles", len(cycles))

	// FilesWithConnections should work
	files := g.FilesWithConnections()
	t.Logf("Graph has %d unique files", len(files))
}

// TestDeepChain tests a 10K node linear chain (no stack overflow from recursion).
func TestDeepChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping deep chain test in short mode")
	}

	g := NewGraph()
	const depth = 10000

	// Create linear chain: 0 -> 1 -> 2 -> ... -> depth-1
	for i := 0; i < depth-1; i++ {
		from := fmt.Sprintf("file%d.go", i+1) // source
		to := fmt.Sprintf("file%d.go", i)     // consumer
		g.AddNew(from, to, TypeImport)
	}

	if g.Count() != depth-1 {
		t.Errorf("Chain should have %d connections, got %d", depth-1, g.Count())
	}

	// DetectCircular should complete without stack overflow
	cycles := g.DetectCircular()
	if len(cycles) != 0 {
		t.Errorf("Linear chain should have no cycles, got %d", len(cycles))
	}

	// Tarjan's algorithm should have visited all nodes
	// All connections should be non-circular
	for _, c := range g.All() {
		if c.IsCircular() {
			t.Error("Linear chain connection marked as circular")
			break
		}
	}

	// FanIn/FanOut should work on endpoints
	// Chain: file0 imports file1, file1 imports file2, ..., file{depth-2} imports file{depth-1}
	// file{depth-1} is the source (head) - imported by file{depth-2}
	// file0 is the consumer (tail) - imports file1, nobody imports file0
	tailFile := "file0.go"
	headFile := fmt.Sprintf("file%d.go", depth-1)

	// Head (file{depth-1}) should have FanIn=1 (imported by file{depth-2})
	if g.FanIn(headFile) != 1 {
		t.Errorf("Head of chain (%s) should have FanIn=1, got %d", headFile, g.FanIn(headFile))
	}
	// Tail (file0) should have FanIn=0 (nobody imports it)
	if g.FanIn(tailFile) != 0 {
		t.Errorf("Tail of chain (%s) should have FanIn=0, got %d", tailFile, g.FanIn(tailFile))
	}
	// Tail (file0) should have FanOut=1 (imports file1)
	if g.FanOut(tailFile) != 1 {
		t.Errorf("Tail of chain (%s) should have FanOut=1, got %d", tailFile, g.FanOut(tailFile))
	}
	// Head (file{depth-1}) should have FanOut=0 (imports nothing)
	if g.FanOut(headFile) != 0 {
		t.Errorf("Head of chain (%s) should have FanOut=0, got %d", headFile, g.FanOut(headFile))
	}
}

// TestHighDegreeNode tests a hub with 10K spokes.
func TestHighDegreeNode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping high degree node test in short mode")
	}

	g := NewGraph()
	const spokes = 10000

	// Hub imported by all spokes
	hub := "hub.go"
	for i := 0; i < spokes; i++ {
		spoke := fmt.Sprintf("spoke%d.go", i)
		g.AddNew(hub, spoke, TypeImport) // spoke imports hub
	}

	// Verify high fan-in
	fanIn := g.FanIn(hub)
	if fanIn != spokes {
		t.Errorf("Hub FanIn = %d, want %d", fanIn, spokes)
	}

	// FromFile should return all connections
	fromHub := g.FromFile(hub)
	if len(fromHub) != spokes {
		t.Errorf("FromFile(hub) = %d connections, want %d", len(fromHub), spokes)
	}

	// Hub should be a bottleneck
	bottlenecks := g.Bottlenecks(spokes / 2)
	found := false
	for _, b := range bottlenecks {
		if b == hub {
			found = true
			break
		}
	}
	if !found {
		t.Error("Hub should be in bottlenecks")
	}

	// No cycles in star topology
	cycles := g.DetectCircular()
	if len(cycles) != 0 {
		t.Errorf("Star topology should have no cycles, got %d", len(cycles))
	}
}

// TestAddRemoveRace tests concurrent add and remove operations.
func TestAddRemoveRace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race test in short mode")
	}

	g := NewGraph()
	const goroutines = 20
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(id)))

			for j := 0; j < iterations; j++ {
				from := fmt.Sprintf("file%d.go", rng.Intn(100))
				to := fmt.Sprintf("file%d.go", rng.Intn(100))
				typ := Type(rng.Intn(4))

				// Random operation
				switch rng.Intn(3) {
				case 0, 1: // Add (more adds than removes)
					g.AddNew(from, to, typ)
				case 2: // Remove
					// Try to remove something (might not exist)
					c := g.GetByEndpoints(from, to, typ)
					if c != nil {
						g.Remove(c.ID())
					}
				}

				// Read operations
				_ = g.Count()
				_ = g.FanIn(from)
				_ = g.FanOut(to)
				_ = g.FromFile(from)
				_ = g.ToFile(to)
			}
		}(i)
	}

	wg.Wait()

	// Should not have crashed
	t.Logf("Final graph has %d connections", g.Count())

	// DetectCircular should work after concurrent operations
	_ = g.DetectCircular()
}

// Benchmarks

func BenchmarkAddNew(b *testing.B) {
	g := NewGraph()
	for i := 0; i < b.N; i++ {
		from := fmt.Sprintf("file%d.go", i)
		to := fmt.Sprintf("file%d.go", i+1)
		g.AddNew(from, to, TypeImport)
	}
}

func BenchmarkAddNew_Duplicate(b *testing.B) {
	g := NewGraph()
	// Pre-add the connection
	g.AddNew("a.go", "b.go", TypeImport)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Adding duplicate should be fast (just lookup)
		g.AddNew("a.go", "b.go", TypeImport)
	}
}

func BenchmarkGet(b *testing.B) {
	g := NewGraph()
	c := g.AddNew("a.go", "b.go", TypeImport)
	id := c.ID()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.Get(id)
	}
}

func BenchmarkDetectCircular_Linear100(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 99; i++ {
		from := fmt.Sprintf("file%d.go", i+1)
		to := fmt.Sprintf("file%d.go", i)
		g.AddNew(from, to, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.DetectCircular()
	}
}

func BenchmarkDetectCircular_Linear1000(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 999; i++ {
		from := fmt.Sprintf("file%d.go", i+1)
		to := fmt.Sprintf("file%d.go", i)
		g.AddNew(from, to, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.DetectCircular()
	}
}

func BenchmarkDetectCircular_Cyclic100(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 100; i++ {
		from := fmt.Sprintf("file%d.go", (i+1)%100)
		to := fmt.Sprintf("file%d.go", i)
		g.AddNew(from, to, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.DetectCircular()
	}
}

func BenchmarkDetectCircular_Dense100(b *testing.B) {
	g := NewGraph()
	// Create dense graph with many connections
	for i := 0; i < 100; i++ {
		for j := 0; j < 100; j++ {
			if i != j {
				from := fmt.Sprintf("file%d.go", i)
				to := fmt.Sprintf("file%d.go", j)
				g.AddNew(from, to, TypeImport)
			}
		}
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.DetectCircular()
	}
}

func BenchmarkFanIn_Hub(b *testing.B) {
	g := NewGraph()
	hub := "hub.go"
	for i := 0; i < 1000; i++ {
		spoke := fmt.Sprintf("spoke%d.go", i)
		g.AddNew(hub, spoke, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.FanIn(hub)
	}
}

func BenchmarkFromFile_Hub(b *testing.B) {
	g := NewGraph()
	hub := "hub.go"
	for i := 0; i < 1000; i++ {
		spoke := fmt.Sprintf("spoke%d.go", i)
		g.AddNew(hub, spoke, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.FromFile(hub)
	}
}

func BenchmarkAll_1000(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 1000; i++ {
		from := fmt.Sprintf("file%d.go", i)
		to := fmt.Sprintf("file%d.go", i+1)
		g.AddNew(from, to, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.All()
	}
}

func BenchmarkFilesWithConnections_1000(b *testing.B) {
	g := NewGraph()
	for i := 0; i < 1000; i++ {
		from := fmt.Sprintf("file%d.go", i)
		to := fmt.Sprintf("file%d.go", i+1)
		g.AddNew(from, to, TypeImport)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.FilesWithConnections()
	}
}
