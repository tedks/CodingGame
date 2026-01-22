// Package mapview provides layout algorithms for the map visualization.
package mapview

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/tedks/CodingGame/internal/tile"
)

// LayoutNode represents a node in the file tree with layout information.
type LayoutNode struct {
	Tile     *tile.Tile
	Children []*LayoutNode
	Parent   *LayoutNode
	Depth    int

	// Layout position (computed by layout algorithm)
	Row    int     // Which row this node is on
	Col    int     // Column offset (based on depth indentation)
	X      float64 // Pixel X position
	Y      float64 // Pixel Y position
	Width  float64 // Tile width
	Height float64 // Tile height
}

// TreeLayout manages the tree-style layout of tiles.
type TreeLayout struct {
	root     *LayoutNode
	nodeMap  map[string]*LayoutNode // path -> node
	allNodes []*LayoutNode          // flat list for iteration

	// Layout parameters
	tileWidth       float64
	tileHeight      float64
	indentWidth     float64 // Pixels per depth level
	verticalGap     float64 // Gap between directory groups
	horizontalGap   float64 // Gap between tiles horizontally
	maxTilesPerRow  int     // Max tiles per row for horizontal layout
	viewportWidth   float64 // Current viewport width for wrapping
}

// NewTreeLayout creates a new tree layout from tiles.
func NewTreeLayout(tiles []*tile.Tile, tileSize float64) *TreeLayout {
	layout := &TreeLayout{
		nodeMap:        make(map[string]*LayoutNode),
		tileWidth:      tileSize,
		tileHeight:     tileSize,         // Square tiles for grid alignment
		indentWidth:    tileSize,         // Indent = 1 tile width
		verticalGap:    0,                // No extra gaps (grid aligned)
		horizontalGap:  0,                // No extra gaps (grid aligned)
		maxTilesPerRow: 10,               // Default max tiles per row
		viewportWidth:  tileSize * 10,    // Default viewport
	}

	layout.buildTree(tiles)
	layout.computeLayout()

	return layout
}

// buildTree constructs the tree structure from flat tile list.
func (l *TreeLayout) buildTree(tiles []*tile.Tile) {
	// Create nodes for all tiles
	for _, t := range tiles {
		node := &LayoutNode{
			Tile: t,
		}
		l.nodeMap[t.RelPath()] = node
		l.allNodes = append(l.allNodes, node)
	}

	// Build parent-child relationships
	for _, node := range l.allNodes {
		relPath := node.Tile.RelPath()
		parentPath := filepath.Dir(relPath)

		if parentPath == "." || parentPath == relPath {
			// This is a root-level item
			if l.root == nil {
				// Create virtual root if needed
				l.root = &LayoutNode{
					Depth: -1,
				}
			}
			node.Parent = l.root
			node.Depth = 0
			l.root.Children = append(l.root.Children, node)
		} else if parentNode, exists := l.nodeMap[parentPath]; exists {
			node.Parent = parentNode
			node.Depth = parentNode.Depth + 1
			parentNode.Children = append(parentNode.Children, node)
		} else {
			// Parent not in our tile list, treat as root level
			if l.root == nil {
				l.root = &LayoutNode{
					Depth: -1,
				}
			}
			node.Parent = l.root
			node.Depth = 0
			l.root.Children = append(l.root.Children, node)
		}
	}

	// Sort children at each level (directories first, then alphabetically)
	l.sortChildren(l.root)
}

// sortChildren recursively sorts children of a node.
func (l *TreeLayout) sortChildren(node *LayoutNode) {
	if node == nil || len(node.Children) == 0 {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		ti := node.Children[i].Tile
		tj := node.Children[j].Tile

		// Directories come first
		if ti.IsDirectory() != tj.IsDirectory() {
			return ti.IsDirectory()
		}

		// Then sort alphabetically
		return strings.ToLower(ti.Name()) < strings.ToLower(tj.Name())
	})

	// Recurse into children
	for _, child := range node.Children {
		l.sortChildren(child)
	}
}

// computeLayout calculates X, Y positions for all nodes using grid-aligned horizontal flow.
// Sibling directories are placed side-by-side when they fit.
func (l *TreeLayout) computeLayout() {
	if l.root == nil {
		return
	}

	// Calculate how many tiles fit per row based on viewport
	tilesPerRow := int(l.viewportWidth / l.tileWidth)
	if tilesPerRow < 3 {
		tilesPerRow = 3
	}
	l.maxTilesPerRow = tilesPerRow

	// First pass: calculate subtree widths
	l.calcSubtreeWidth(l.root)

	// Second pass: layout with horizontal placement
	l.layoutSubtree(l.root, 0, 0)
}

// SubtreeWidth is stored in the Col field temporarily during width calculation
// (we'll overwrite it during actual layout)

// calcSubtreeWidth calculates how many columns each subtree needs.
// Returns the width in columns.
func (l *TreeLayout) calcSubtreeWidth(node *LayoutNode) int {
	if node == nil {
		return 0
	}

	// Separate directories and files
	var dirs, files []*LayoutNode
	for _, child := range node.Children {
		if child.Tile != nil && child.Tile.IsDirectory() {
			dirs = append(dirs, child)
		} else if child.Tile != nil {
			files = append(files, child)
		}
	}

	// Calculate width needed for directories (placed side by side)
	dirWidth := 0
	for _, dir := range dirs {
		childWidth := l.calcSubtreeWidth(dir)
		// Directory needs at least 1 column for itself
		if childWidth < 1 {
			childWidth = 1
		}
		dirWidth += childWidth
	}

	// Calculate width needed for files (horizontal row)
	fileWidth := len(files)

	// Total width is max of: 1 (for this node), dir children width, file children width
	width := 1
	if dirWidth > width {
		width = dirWidth
	}
	if fileWidth > width {
		width = fileWidth
	}

	// Store width temporarily (will be overwritten during layout)
	node.Col = width
	return width
}

// layoutSubtree places a node and its children starting at (startCol, startRow).
// Returns the row after this subtree.
func (l *TreeLayout) layoutSubtree(node *LayoutNode, startCol, startRow int) int {
	if node == nil {
		return startRow
	}

	currentRow := startRow

	// Place this node (skip virtual root)
	if node.Tile != nil {
		node.Row = currentRow
		node.Col = startCol
		node.X = float64(startCol) * l.tileWidth
		node.Y = float64(currentRow) * l.tileHeight
		node.Width = l.tileWidth
		node.Height = l.tileHeight
		currentRow++
	}

	// Separate directories and files
	var dirs, files []*LayoutNode
	for _, child := range node.Children {
		if child.Tile != nil && child.Tile.IsDirectory() {
			dirs = append(dirs, child)
		} else if child.Tile != nil {
			files = append(files, child)
		}
	}

	// Layout directories side-by-side
	if len(dirs) > 0 {
		dirRow := currentRow
		maxChildRow := currentRow

		col := startCol
		if node.Tile != nil {
			col = startCol // Children start at same column as parent (indented by being in subtree)
		}

		for _, dir := range dirs {
			// Get the pre-calculated width for this subtree
			subtreeWidth := l.calcSubtreeWidth(dir)
			if subtreeWidth < 1 {
				subtreeWidth = 1
			}

			// Layout this directory subtree
			endRow := l.layoutSubtree(dir, col, dirRow)
			if endRow > maxChildRow {
				maxChildRow = endRow
			}

			// Move to next column position
			col += subtreeWidth
		}

		currentRow = maxChildRow
	}

	// Layout files horizontally
	if len(files) > 0 {
		col := startCol
		if node.Tile != nil {
			col = startCol // Files at same indent as parent's children position
		}

		for _, file := range files {
			// Wrap if we exceed viewport
			if col >= l.maxTilesPerRow {
				col = startCol
				currentRow++
			}

			file.Row = currentRow
			file.Col = col
			file.X = float64(col) * l.tileWidth
			file.Y = float64(currentRow) * l.tileHeight
			file.Width = l.tileWidth
			file.Height = l.tileHeight

			col++
		}
		currentRow++
	}

	return currentRow
}

// Nodes returns all layout nodes for rendering.
func (l *TreeLayout) Nodes() []*LayoutNode {
	return l.allNodes
}

// NodeAt returns the node at the given path, if any.
func (l *TreeLayout) NodeAt(relPath string) *LayoutNode {
	return l.nodeMap[relPath]
}

// Root returns the root node.
func (l *TreeLayout) Root() *LayoutNode {
	return l.root
}

// TotalHeight returns the total height of the layout in pixels.
func (l *TreeLayout) TotalHeight() float64 {
	maxRow := 0
	for _, node := range l.allNodes {
		if node.Row > maxRow {
			maxRow = node.Row
		}
	}
	return float64(maxRow+1) * l.tileHeight
}

// TotalWidth returns the total width needed for the layout.
func (l *TreeLayout) TotalWidth() float64 {
	maxCol := 0
	for _, node := range l.allNodes {
		if node.Col > maxCol {
			maxCol = node.Col
		}
	}
	return float64(maxCol+1) * l.tileWidth
}

// UpdateTileSize recalculates layout with a new tile size.
func (l *TreeLayout) UpdateTileSize(tileSize float64) {
	l.tileWidth = tileSize
	l.tileHeight = tileSize
	l.indentWidth = tileSize

	l.computeLayout()
}

// SetViewportWidth updates the viewport width for calculating wrapping.
func (l *TreeLayout) SetViewportWidth(width float64) {
	l.viewportWidth = width
	l.computeLayout()
}

// VisibleNodes returns nodes that would be visible in the given viewport.
func (l *TreeLayout) VisibleNodes(viewX, viewY, viewWidth, viewHeight float64) []*LayoutNode {
	var visible []*LayoutNode

	for _, node := range l.allNodes {
		if node.Tile == nil {
			continue
		}

		// Check if node intersects viewport (grid-aligned check)
		nodeRight := node.X + l.tileWidth
		nodeBottom := node.Y + l.tileHeight

		if nodeRight >= viewX && node.X <= viewX+viewWidth &&
			nodeBottom >= viewY && node.Y <= viewY+viewHeight {
			visible = append(visible, node)
		}
	}

	return visible
}
