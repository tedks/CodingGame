package mapview

import (
	"testing"

	"github.com/tedks/CodingGame/internal/tile"
)

// Helper function to create a test tile
func newTestTile(relPath string, isDir bool) *tile.Tile {
	return tile.New("/test/"+relPath, relPath, isDir)
}

func TestNewTreeLayout(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("src", true),
		newTestTile("src/main.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	if layout == nil {
		t.Fatal("expected non-nil layout")
	}
	if layout.root == nil {
		t.Fatal("expected non-nil root")
	}
	if len(layout.allNodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(layout.allNodes))
	}
	if layout.tileWidth != 80 {
		t.Errorf("expected tileWidth 80, got %f", layout.tileWidth)
	}
}

func TestBuildTree_SimpleHierarchy(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("src", true),
		newTestTile("src/main.go", false),
		newTestTile("src/util.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Check parent-child relationships
	srcNode := layout.NodeAt("src")
	if srcNode == nil {
		t.Fatal("expected to find src node")
	}
	if len(srcNode.Children) != 2 {
		t.Errorf("expected src to have 2 children, got %d", len(srcNode.Children))
	}

	mainNode := layout.NodeAt("src/main.go")
	if mainNode == nil {
		t.Fatal("expected to find src/main.go node")
	}
	if mainNode.Parent != srcNode {
		t.Error("expected main.go parent to be src")
	}
}

func TestBuildTree_DeepNesting(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a", true),
		newTestTile("a/b", true),
		newTestTile("a/b/c", true),
		newTestTile("a/b/c/file.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Check depths
	aNode := layout.NodeAt("a")
	if aNode.Depth != 0 {
		t.Errorf("expected depth 0 for 'a', got %d", aNode.Depth)
	}

	bNode := layout.NodeAt("a/b")
	if bNode.Depth != 1 {
		t.Errorf("expected depth 1 for 'a/b', got %d", bNode.Depth)
	}

	cNode := layout.NodeAt("a/b/c")
	if cNode.Depth != 2 {
		t.Errorf("expected depth 2 for 'a/b/c', got %d", cNode.Depth)
	}

	fileNode := layout.NodeAt("a/b/c/file.go")
	if fileNode.Depth != 3 {
		t.Errorf("expected depth 3 for file, got %d", fileNode.Depth)
	}
}

func TestBuildTree_MultipleRoots(t *testing.T) {
	// Files at root level (no common parent directory)
	tiles := []*tile.Tile{
		newTestTile("main.go", false),
		newTestTile("go.mod", false),
		newTestTile("README.md", false),
	}

	layout := NewTreeLayout(tiles, 80)

	if layout.root == nil {
		t.Fatal("expected virtual root to be created")
	}
	if len(layout.root.Children) != 3 {
		t.Errorf("expected 3 root children, got %d", len(layout.root.Children))
	}

	// All should have depth 0
	for _, n := range layout.root.Children {
		if n.Depth != 0 {
			t.Errorf("expected depth 0 for root-level item, got %d", n.Depth)
		}
	}
}

func TestBuildTree_EmptyInput(t *testing.T) {
	layout := NewTreeLayout([]*tile.Tile{}, 80)

	if layout == nil {
		t.Fatal("expected non-nil layout even for empty input")
	}
	if len(layout.allNodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(layout.allNodes))
	}
}

func TestSortChildren_DirectoriesFirst(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("file.go", false),
		newTestTile("aaa.go", false),
		newTestTile("src", true),
		newTestTile("internal", true),
	}

	layout := NewTreeLayout(tiles, 80)

	// After sorting: directories first (alphabetically), then files (alphabetically)
	children := layout.root.Children
	if len(children) != 4 {
		t.Fatalf("expected 4 children, got %d", len(children))
	}

	// First two should be directories (internal, src - alphabetical)
	if !children[0].Tile.IsDirectory() || children[0].Tile.Name() != "internal" {
		t.Errorf("expected first child to be 'internal' dir, got %s (isDir: %v)",
			children[0].Tile.Name(), children[0].Tile.IsDirectory())
	}
	if !children[1].Tile.IsDirectory() || children[1].Tile.Name() != "src" {
		t.Errorf("expected second child to be 'src' dir, got %s (isDir: %v)",
			children[1].Tile.Name(), children[1].Tile.IsDirectory())
	}

	// Last two should be files (aaa.go, file.go - alphabetical)
	if children[2].Tile.IsDirectory() || children[2].Tile.Name() != "aaa.go" {
		t.Errorf("expected third child to be 'aaa.go' file, got %s (isDir: %v)",
			children[2].Tile.Name(), children[2].Tile.IsDirectory())
	}
	if children[3].Tile.IsDirectory() || children[3].Tile.Name() != "file.go" {
		t.Errorf("expected fourth child to be 'file.go' file, got %s (isDir: %v)",
			children[3].Tile.Name(), children[3].Tile.IsDirectory())
	}
}

func TestSortChildren_CaseInsensitive(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("Zebra.go", false),
		newTestTile("apple.go", false),
		newTestTile("Banana.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	children := layout.root.Children
	expected := []string{"apple.go", "Banana.go", "Zebra.go"}

	for i, exp := range expected {
		if children[i].Tile.Name() != exp {
			t.Errorf("position %d: expected %s, got %s", i, exp, children[i].Tile.Name())
		}
	}
}

func TestComputeLayout_PositionCalculations(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("src", true),
		newTestTile("src/main.go", false),
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	srcNode := layout.NodeAt("src")
	mainNode := layout.NodeAt("src/main.go")

	// src should be at row 0, col 0
	if srcNode.X != 0 {
		t.Errorf("expected src X=0, got %f", srcNode.X)
	}
	if srcNode.Y != 0 {
		t.Errorf("expected src Y=0, got %f", srcNode.Y)
	}
	if srcNode.Width != tileSize {
		t.Errorf("expected src width=%f, got %f", tileSize, srcNode.Width)
	}

	// main.go should be indented (col=1) and on a later row
	if mainNode.Col != 1 {
		t.Errorf("expected main.go col=1, got %d", mainNode.Col)
	}
	if mainNode.X != tileSize {
		t.Errorf("expected main.go X=%f, got %f", tileSize, mainNode.X)
	}
}

func TestComputeLayout_HorizontalWrapping(t *testing.T) {
	// Create many files to trigger wrapping
	tiles := []*tile.Tile{}
	for i := 0; i < 15; i++ {
		tiles = append(tiles, newTestTile("file"+string(rune('a'+i))+".go", false))
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)
	// Default viewport is 10*tileSize, so maxTilesPerRow = 10

	// Files should wrap after 10 per row
	nodesOnRow0 := 0
	nodesOnRow1 := 0
	for _, node := range layout.allNodes {
		if node.Row == 0 {
			nodesOnRow0++
		} else if node.Row == 1 {
			nodesOnRow1++
		}
	}

	if nodesOnRow0 != 10 {
		t.Errorf("expected 10 nodes on row 0, got %d", nodesOnRow0)
	}
	if nodesOnRow1 != 5 {
		t.Errorf("expected 5 nodes on row 1, got %d", nodesOnRow1)
	}
}

func TestVisibleNodes_FullyVisible(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
		newTestTile("b.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Large viewport that contains all nodes
	visible := layout.VisibleNodes(0, 0, 1000, 1000)

	if len(visible) != 2 {
		t.Errorf("expected 2 visible nodes, got %d", len(visible))
	}
}

func TestVisibleNodes_PartiallyVisible(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
		newTestTile("b.go", false),
		newTestTile("c.go", false),
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	// Viewport smaller than one tile should still see tiles that overlap
	// Viewport exactly one tile size will see first tile fully and touch the edge of second
	visible := layout.VisibleNodes(0, 0, tileSize-1, tileSize-1)

	if len(visible) != 1 {
		t.Errorf("expected 1 visible node with viewport smaller than tile, got %d", len(visible))
	}
}

func TestVisibleNodes_NoneVisible(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
		newTestTile("b.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Viewport far away from all nodes
	visible := layout.VisibleNodes(10000, 10000, 100, 100)

	if len(visible) != 0 {
		t.Errorf("expected 0 visible nodes, got %d", len(visible))
	}
}

func TestVisibleNodes_ViewportOffset(t *testing.T) {
	// Create tiles that span multiple rows
	tiles := []*tile.Tile{
		newTestTile("dir", true),
		newTestTile("dir/a.go", false),
		newTestTile("dir/b.go", false),
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	// Viewport starting at row 1 should see the files but not the directory header
	// (depending on exact layout positioning)
	visible := layout.VisibleNodes(0, tileSize, 1000, tileSize)

	// Should have at least some nodes visible
	if len(visible) == 0 {
		t.Error("expected at least some visible nodes when offset viewport")
	}
}

func TestTotalHeight(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a", true),
		newTestTile("a/file1.go", false),
		newTestTile("a/file2.go", false),
		newTestTile("a/file3.go", false),
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	height := layout.TotalHeight()

	// Should be positive and reasonable
	if height <= 0 {
		t.Errorf("expected positive height, got %f", height)
	}
	// At minimum, should be at least 2 rows (dir + files)
	if height < tileSize*2 {
		t.Errorf("expected height >= %f, got %f", tileSize*2, height)
	}
}

func TestTotalWidth(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a", true),
		newTestTile("a/file.go", false),
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	width := layout.TotalWidth()

	// Should be positive
	if width <= 0 {
		t.Errorf("expected positive width, got %f", width)
	}
	// Files are indented, so width should be at least 2 columns
	if width < tileSize*2 {
		t.Errorf("expected width >= %f for nested structure, got %f", tileSize*2, width)
	}
}

func TestUpdateTileSize(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	initialX := layout.allNodes[0].X

	// Update tile size
	layout.UpdateTileSize(100)

	node := layout.allNodes[0]
	if node.Width != 100 {
		t.Errorf("expected width 100 after update, got %f", node.Width)
	}

	// Position should scale proportionally (both were at col 0)
	if node.X != 0 {
		t.Errorf("expected X=0 (was %f), got %f", initialX, node.X)
	}

	if layout.tileWidth != 100 {
		t.Errorf("expected layout tileWidth 100, got %f", layout.tileWidth)
	}
}

func TestSetViewportWidth(t *testing.T) {
	// Create enough tiles to test wrapping behavior
	tiles := []*tile.Tile{}
	for i := 0; i < 20; i++ {
		tiles = append(tiles, newTestTile("file"+string(rune('a'+i))+".go", false))
	}

	tileSize := 80.0
	layout := NewTreeLayout(tiles, tileSize)

	// Count rows with default viewport (10 tiles wide = 800px)
	initialMaxRow := 0
	for _, n := range layout.allNodes {
		if n.Row > initialMaxRow {
			initialMaxRow = n.Row
		}
	}

	// Set narrower viewport (4 tiles wide = 320px)
	layout.SetViewportWidth(320)

	// Should have more rows now due to wrapping
	newMaxRow := 0
	for _, n := range layout.allNodes {
		if n.Row > newMaxRow {
			newMaxRow = n.Row
		}
	}

	if newMaxRow <= initialMaxRow {
		t.Errorf("expected more rows with narrower viewport, got %d (was %d)", newMaxRow, initialMaxRow)
	}
}

func TestNodes(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
		newTestTile("b.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	nodes := layout.Nodes()

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestNodeAt(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("src/main.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Existing path
	node := layout.NodeAt("src/main.go")
	if node == nil {
		t.Error("expected to find node at src/main.go")
	}

	// Non-existing path
	nonexistent := layout.NodeAt("nonexistent.go")
	if nonexistent != nil {
		t.Error("expected nil for non-existent path")
	}
}

func TestRoot(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("a.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	root := layout.Root()
	if root == nil {
		t.Error("expected non-nil root")
	}
	if root.Tile != nil {
		t.Error("expected virtual root to have nil Tile")
	}
	if root.Depth != -1 {
		t.Errorf("expected virtual root depth -1, got %d", root.Depth)
	}
}

func TestLayoutNode_Fields(t *testing.T) {
	tiles := []*tile.Tile{
		newTestTile("dir", true),
		newTestTile("dir/file.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	fileNode := layout.NodeAt("dir/file.go")
	dirNode := layout.NodeAt("dir")

	// Check parent relationship
	if fileNode.Parent != dirNode {
		t.Error("expected file's parent to be dir")
	}

	// Check that dir has file as child
	foundChild := false
	for _, child := range dirNode.Children {
		if child == fileNode {
			foundChild = true
			break
		}
	}
	if !foundChild {
		t.Error("expected dir to have file as child")
	}
}

func TestEmptyDirectory(t *testing.T) {
	// Empty directory with no children
	tiles := []*tile.Tile{
		newTestTile("empty_dir", true),
	}

	layout := NewTreeLayout(tiles, 80)

	node := layout.NodeAt("empty_dir")
	if node == nil {
		t.Fatal("expected to find empty_dir node")
	}
	if len(node.Children) != 0 {
		t.Errorf("expected empty dir to have 0 children, got %d", len(node.Children))
	}
}

func TestMixedStructure(t *testing.T) {
	// Complex structure with mix of dirs, files, nesting
	tiles := []*tile.Tile{
		newTestTile("README.md", false),
		newTestTile("main.go", false),
		newTestTile("cmd", true),
		newTestTile("cmd/app", true),
		newTestTile("cmd/app/main.go", false),
		newTestTile("internal", true),
		newTestTile("internal/util", true),
		newTestTile("internal/util/helper.go", false),
		newTestTile("internal/util/helper_test.go", false),
	}

	layout := NewTreeLayout(tiles, 80)

	// Verify structure
	if len(layout.allNodes) != 9 {
		t.Errorf("expected 9 nodes, got %d", len(layout.allNodes))
	}

	// Verify nesting levels
	utilHelperNode := layout.NodeAt("internal/util/helper.go")
	if utilHelperNode == nil {
		t.Fatal("expected to find internal/util/helper.go")
	}
	if utilHelperNode.Depth != 2 {
		t.Errorf("expected depth 2 for internal/util/helper.go, got %d", utilHelperNode.Depth)
	}

	// Verify root-level items are sorted correctly
	rootChildren := layout.root.Children
	if len(rootChildren) == 0 {
		t.Fatal("expected root to have children")
	}
	// First should be directories (cmd, internal), then files (main.go, README.md)
	if !rootChildren[0].Tile.IsDirectory() {
		t.Error("expected first root child to be a directory")
	}
}
