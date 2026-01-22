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

	currentRow := 0
	l.layoutChildren(l.root.Children, 0, &currentRow)
}

// layoutChildren lays out a slice of children at a given depth.
// Children are placed horizontally, wrapping to new rows as needed.
// Directories get their own row, files flow horizontally.
func (l *TreeLayout) layoutChildren(children []*LayoutNode, depth int, currentRow *int) {
	if len(children) == 0 {
		return
	}

	// Separate directories and files
	var dirs, files []*LayoutNode
	for _, child := range children {
		if child.Tile.IsDirectory() {
			dirs = append(dirs, child)
		} else {
			files = append(files, child)
		}
	}

	// Layout directories first - each directory on its own row with its contents below
	for _, dir := range dirs {
		// Place directory tile
		dir.Row = *currentRow
		dir.Col = depth
		dir.X = float64(depth) * l.tileWidth
		dir.Y = float64(*currentRow) * l.tileHeight
		dir.Width = l.tileWidth
		dir.Height = l.tileHeight
		dir.Depth = depth
		*currentRow++

		// Layout directory's children recursively
		l.layoutChildren(dir.Children, depth+1, currentRow)
	}

	// Layout files horizontally at current depth
	if len(files) > 0 {
		col := depth
		maxCol := l.maxTilesPerRow
		for _, file := range files {
			// Wrap to next row if we exceed max columns
			if col >= maxCol {
				col = depth
				*currentRow++
			}

			file.Row = *currentRow
			file.Col = col
			file.X = float64(col) * l.tileWidth
			file.Y = float64(*currentRow) * l.tileHeight
			file.Width = l.tileWidth
			file.Height = l.tileHeight
			file.Depth = depth

			col++
		}
		*currentRow++
	}
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
