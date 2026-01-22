// Package mapview provides a zoomable, pannable 2D tile-based map visualization
// for displaying project files and directories. It implements a fog of war system
// that reveals tiles as Claude Code reads files, creating a visual representation
// of the AI's context window.
//
// The map supports 5 zoom levels from World (repository root) to Interior (file contents).
// Navigation is keyboard-driven with vim-style keys (hjkl) or arrow keys for panning,
// and +/- keys for zooming.
package mapview

import (
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/belt"
	"github.com/tedks/CodingGame/internal/connection"
	"github.com/tedks/CodingGame/internal/tile"
)

// ViewMode represents which view of the map is active.
type ViewMode int

const (
	// ViewDirectory shows the filesystem hierarchy as a tile grid.
	ViewDirectory ViewMode = iota
	// ViewDataflow shows dependencies as Factorio-style belts.
	ViewDataflow
)

// String returns a human-readable name for the view mode.
func (v ViewMode) String() string {
	switch v {
	case ViewDirectory:
		return "Directory"
	case ViewDataflow:
		return "Dataflow"
	default:
		return "Unknown"
	}
}

// ZoomLevel represents the current zoom level (0-5)
type ZoomLevel int

const (
	ZoomOverview ZoomLevel = 0 // Compact overview, small tiles
	ZoomWorld    ZoomLevel = 1 // Repository root, top-level dirs
	ZoomRegion   ZoomLevel = 2 // Package/module clusters
	ZoomCity     ZoomLevel = 3 // Individual packages with buildings
	ZoomStreet   ZoomLevel = 4 // Files within a package
	ZoomInterior ZoomLevel = 5 // Single file contents

	// Rendering constants
	TileBorderSpacing = 2  // Pixels between tiles
	BorderColorBoost  = 20 // Amount to lighten border color
)

// Layout constants
const (
	TopPadding = 1 // Empty rows at top to avoid overlapping debug text
)

// MapView manages the map visualization with tiles and fog of war
type MapView struct {
	projectPath string
	width       int
	height      int

	// View state
	viewMode ViewMode // Directory or Dataflow

	// Camera state
	panX      float64
	panY      float64
	zoomLevel ZoomLevel

	// Mouse drag state
	dragging   bool
	dragStartX int
	dragStartY int
	dragPanX   float64
	dragPanY   float64

	// Tile system
	tiles      []*tile.Tile
	tileMap    map[string]*tile.Tile // Fast lookup by path
	treeLayout *TreeLayout           // Tree-style layout for directory view

	// Colors
	bgColor       color.RGBA
	gridColor     color.RGBA
	fogColor      color.RGBA
	revealedColor color.RGBA

	// Dataflow colors (used in ViewDataflow mode)
	beltImportColor      color.RGBA
	beltInheritanceColor color.RGBA
	beltCompositionColor color.RGBA
	beltCallColor        color.RGBA
	beltCircularColor    color.RGBA

	// Belt rendering (Phase 4)
	beltRenderer    *belt.Renderer
	connectionGraph *connection.Graph
}

// New creates a new map view for the given project path
func New(projectPath string, width, height int) (*MapView, error) {
	// Scan project directory to build tile system
	tiles, err := scanProjectDirectory(projectPath)
	if err != nil {
		return nil, err
	}

	// Build fast lookup map
	tileMap := make(map[string]*tile.Tile, len(tiles))
	for _, t := range tiles {
		tileMap[t.Path()] = t
	}

	// Create tree layout for directory view
	treeLayout := NewTreeLayout(tiles, 80) // Default tile size of 80

	return &MapView{
		projectPath:   projectPath,
		width:         width,
		height:        height,
		viewMode:      ViewDirectory, // Default to directory view
		panX:          0,
		panY:          0,
		zoomLevel:     ZoomWorld,
		tiles:         tiles,
		tileMap:       tileMap,
		treeLayout:    treeLayout,
		bgColor:       color.RGBA{20, 20, 30, 255},
		gridColor:     color.RGBA{60, 60, 80, 255},
		fogColor:      color.RGBA{40, 40, 50, 200},
		revealedColor: color.RGBA{100, 120, 150, 255},
		// Belt colors for dataflow view
		beltImportColor:      color.RGBA{100, 150, 200, 255}, // Blue for imports
		beltInheritanceColor: color.RGBA{200, 150, 100, 255}, // Orange for inheritance
		beltCompositionColor: color.RGBA{150, 200, 100, 255}, // Green for composition
		beltCallColor:        color.RGBA{200, 100, 200, 255}, // Purple for calls
		beltCircularColor:    color.RGBA{255, 80, 80, 255},   // Red for circular deps

		// Belt rendering
		beltRenderer:    belt.NewRenderer(),
		connectionGraph: connection.NewGraph(),
	}, nil
}

// scanProjectDirectory scans the project directory and creates tiles
func scanProjectDirectory(projectPath string) ([]*tile.Tile, error) {
	var tiles []*tile.Tile

	// Walk the directory tree
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get base name safely
		baseName := filepath.Base(path)
		if baseName == "" || baseName == "." {
			return nil
		}

		// Skip specific hidden directories/files, but allow important dotfiles
		if baseName[0] == '.' && path != projectPath {
			// Always skip .git directory
			if baseName == ".git" {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Allow important dotfiles/directories
			allowedDotfiles := map[string]bool{
				".github":        true,
				".bazelrc":       true,
				".bazelversion":  true,
				".envrc":         true,
				".beads":         true,
				".gitignore":     true,
				".gitattributes": true,
				".editorconfig":  true,
			}

			if !allowedDotfiles[baseName] {
				// Skip other hidden files/directories
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Bazel output directories - show them but don't descend into them
		// Check if this is a symlink first using Lstat
		if linfo, lerr := os.Lstat(path); lerr == nil && linfo.Mode()&os.ModeSymlink != 0 {
			bazelDirs := map[string]bool{
				"bazel-bin":        true,
				"bazel-out":        true,
				"bazel-testlogs":   true,
				"bazel-CodingGame": true,
			}
			if bazelDirs[baseName] {
				// Create tile for the bazel symlink (as directory)
				relPath, _ := filepath.Rel(projectPath, path)
				t := tile.New(path, relPath, true)
				tiles = append(tiles, t)
				return nil // Don't use SkipDir - just don't add children (symlink won't be walked into)
			}
		}

		// Create tile for this file/directory
		relPath, _ := filepath.Rel(projectPath, path)
		if relPath == "." {
			relPath = filepath.Base(projectPath)
		}

		t := tile.New(path, relPath, info.IsDir())
		tiles = append(tiles, t)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tiles, nil
}

// Update updates the map view state and handles mouse input
func (m *MapView) Update() {
	// Handle mouse drag for panning
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if !m.dragging {
			// Start drag
			m.dragging = true
			m.dragStartX = x
			m.dragStartY = y
			m.dragPanX = m.panX
			m.dragPanY = m.panY
		} else {
			// Continue drag - update pan based on mouse movement
			dx := float64(x - m.dragStartX)
			dy := float64(y - m.dragStartY)
			m.panX = m.dragPanX + dx
			m.panY = m.dragPanY + dy
		}
	} else {
		m.dragging = false
	}
}

// Draw renders the map view based on current view mode.
func (m *MapView) Draw(screen *ebiten.Image, offsetX, offsetY int) {
	// Draw the background for the map area
	vector.DrawFilledRect(
		screen,
		float32(offsetX),
		float32(offsetY),
		float32(m.width),
		float32(m.height),
		m.bgColor,
		false,
	)

	switch m.viewMode {
	case ViewDirectory:
		m.drawDirectoryView(screen, offsetX, offsetY)
	case ViewDataflow:
		m.drawDataflowView(screen, offsetX, offsetY)
	}
}

// drawDirectoryView renders the filesystem hierarchy as a tree layout.
func (m *MapView) drawDirectoryView(screen *ebiten.Image, offsetX, offsetY int) {
	// Update tree layout with current tile size and viewport
	tileSize := m.getTileSize()
	m.treeLayout.UpdateTileSize(tileSize)
	m.treeLayout.SetViewportWidth(float64(m.width))

	// Top padding offset (one empty row to avoid overlapping debug text)
	topOffset := float64(TopPadding) * tileSize

	// Get visible nodes based on current viewport (account for top padding)
	viewX := -m.panX
	viewY := -m.panY - topOffset
	visibleNodes := m.treeLayout.VisibleNodes(viewX, viewY, float64(m.width), float64(m.height))

	// Draw grouping bars for directories with children
	groupBarColor := color.RGBA{80, 100, 120, 180}
	for _, node := range visibleNodes {
		if node.Tile == nil || !node.Tile.IsDirectory() || len(node.Children) == 0 {
			continue
		}

		// Find the Y extent of this directory's children
		minY := node.Y
		maxY := node.Y
		for _, child := range node.Children {
			if child.Y > maxY {
				maxY = child.Y
			}
		}

		// Draw vertical grouping bar to the left of children (with top padding)
		barX := node.X + m.panX + float64(offsetX) + 4
		barY1 := minY + m.panY + float64(offsetY) + topOffset + tileSize
		barY2 := maxY + m.panY + float64(offsetY) + topOffset + tileSize

		if barY2 > barY1 {
			vector.StrokeLine(screen, float32(barX), float32(barY1), float32(barX), float32(barY2), 2, groupBarColor, false)
		}
	}

	// Draw visible tiles using tree layout positions (grid-aligned, with top padding)
	for _, node := range visibleNodes {
		if node.Tile == nil {
			continue
		}

		// Apply camera pan, screen offset, and top padding
		x := node.X + m.panX + float64(offsetX)
		y := node.Y + m.panY + float64(offsetY) + topOffset

		// Draw the tile at its grid-aligned position
		m.drawTile(screen, node.Tile, float32(x), float32(y), float32(tileSize))
	}

	// Draw grid lines (aligned with tile positions)
	m.drawGrid(screen, offsetX, offsetY, tileSize)
}

// drawDataflowView renders dependencies as Factorio-style belts.
// In this view, tiles are positioned based on dependency relationships,
// and connections (belts) are drawn between them.
func (m *MapView) drawDataflowView(screen *ebiten.Image, offsetX, offsetY int) {
	// Calculate tile size based on zoom level
	tileSize := m.getTileSize()
	tilesPerRow := m.width / int(tileSize)

	// Build tile position map for belt rendering
	tilePositions := make(map[string]belt.TilePosition)
	for i, t := range m.tiles {
		col := i % tilesPerRow
		row := i / tilesPerRow

		x := float64(col)*tileSize + m.panX + float64(offsetX)
		y := float64(row)*tileSize + m.panY + float64(offsetY)

		// Store position using relative path for matching with connections
		tilePositions[t.RelPath()] = belt.TilePosition{
			X:      float32(x),
			Y:      float32(y),
			Width:  float32(tileSize),
			Height: float32(tileSize),
		}
	}

	// Draw tiles first (belts will be drawn on top)
	for i, t := range m.tiles {
		col := i % tilesPerRow
		row := i / tilesPerRow

		x := float64(col)*tileSize + m.panX + float64(offsetX)
		y := float64(row)*tileSize + m.panY + float64(offsetY)

		// Skip if outside visible area (accounting for offset)
		if x+tileSize < float64(offsetX) || x > float64(offsetX+m.width) ||
			y+tileSize < float64(offsetY) || y > float64(offsetY+m.height) {
			continue
		}

		m.drawTile(screen, t, float32(x), float32(y), float32(tileSize))
	}

	// Draw belts (connections)
	m.beltRenderer.Draw(screen, m.connectionGraph, tilePositions, offsetX, offsetY)

	// Draw grid lines with a different style to indicate dataflow mode
	m.drawDataflowGrid(screen, offsetX, offsetY, tileSize)
}

// drawDataflowGrid draws grid lines styled for dataflow view.
func (m *MapView) drawDataflowGrid(screen *ebiten.Image, offsetX, offsetY int, tileSize float64) {
	// Use a slightly different color to indicate we're in dataflow mode
	gridColor := color.RGBA{80, 60, 100, 255} // Purple tint for dataflow

	// Draw vertical grid lines (clipped to map area)
	for x := m.panX + float64(offsetX); x < float64(offsetX+m.width); x += tileSize {
		if x >= float64(offsetX) {
			vector.StrokeLine(screen, float32(x), float32(offsetY), float32(x), float32(offsetY+m.height), 1, gridColor, false)
		}
	}

	// Draw horizontal grid lines (clipped to map area)
	for y := m.panY + float64(offsetY); y < float64(offsetY+m.height); y += tileSize {
		if y >= float64(offsetY) {
			vector.StrokeLine(screen, float32(offsetX), float32(y), float32(offsetX+m.width), float32(y), 1, gridColor, false)
		}
	}
}

// drawTile renders a single tile with label
func (m *MapView) drawTile(screen *ebiten.Image, t *tile.Tile, x, y, size float32) {
	// Determine tile color based on fog state
	var tileColor color.RGBA
	if t.IsRevealed() {
		tileColor = m.revealedColor
		if t.IsDirectory() {
			// Directories are slightly darker
			tileColor = color.RGBA{80, 100, 130, 255}
		}
	} else {
		tileColor = m.fogColor
	}

	// Draw tile rectangle (with border spacing)
	vector.DrawFilledRect(screen, x, y, size-TileBorderSpacing, size-TileBorderSpacing, tileColor, false)

	// Draw border with lighter color (clamp to prevent overflow)
	borderColor := color.RGBA{
		clampUint8(int(tileColor.R) + BorderColorBoost),
		clampUint8(int(tileColor.G) + BorderColorBoost),
		clampUint8(int(tileColor.B) + BorderColorBoost),
		255,
	}
	vector.StrokeRect(screen, x, y, size-TileBorderSpacing, size-TileBorderSpacing, 1, borderColor, false)

	// Draw label (file/directory name)
	label := t.Name()
	if t.IsDirectory() {
		label += "/"
	}
	// Truncate label if too long for tile
	maxChars := int(size) / 6 // ~6 pixels per character
	if maxChars < 3 {
		maxChars = 3
	}
	if len(label) > maxChars {
		label = label[:maxChars-2] + ".."
	}
	ebitenutil.DebugPrintAt(screen, label, int(x)+2, int(y)+2)
}

// clampUint8 clamps an int value to the valid uint8 range [0, 255]
func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// drawGrid draws the grid lines bounded to the content area
func (m *MapView) drawGrid(screen *ebiten.Image, offsetX, offsetY int, tileSize float64) {
	// Get content bounds from tree layout
	contentWidth := m.treeLayout.TotalWidth()
	contentHeight := m.treeLayout.TotalHeight() + float64(TopPadding)*tileSize

	// Calculate where content starts/ends in screen coordinates
	contentStartX := m.panX + float64(offsetX)
	contentStartY := m.panY + float64(offsetY)
	contentEndX := contentStartX + contentWidth
	contentEndY := contentStartY + contentHeight

	// Clip to viewport
	viewLeft := float64(offsetX)
	viewRight := float64(offsetX + m.width)
	viewTop := float64(offsetY)
	viewBottom := float64(offsetY + m.height)

	// Draw vertical grid lines (only within content bounds)
	startX := contentStartX
	if startX < viewLeft {
		// Align to grid
		startX = viewLeft - math.Mod(viewLeft-contentStartX, tileSize)
	}
	for x := startX; x <= contentEndX && x < viewRight; x += tileSize {
		if x >= viewLeft {
			y1 := math.Max(contentStartY, viewTop)
			y2 := math.Min(contentEndY, viewBottom)
			if y2 > y1 {
				vector.StrokeLine(screen, float32(x), float32(y1), float32(x), float32(y2), 1, m.gridColor, false)
			}
		}
	}

	// Draw horizontal grid lines (only within content bounds)
	startY := contentStartY
	if startY < viewTop {
		startY = viewTop - math.Mod(viewTop-contentStartY, tileSize)
	}
	for y := startY; y <= contentEndY && y < viewBottom; y += tileSize {
		if y >= viewTop {
			x1 := math.Max(contentStartX, viewLeft)
			x2 := math.Min(contentEndX, viewRight)
			if x2 > x1 {
				vector.StrokeLine(screen, float32(x1), float32(y), float32(x2), float32(y), 1, m.gridColor, false)
			}
		}
	}
}

// getTileSize returns the tile size in pixels for current zoom level
func (m *MapView) getTileSize() float64 {
	switch m.zoomLevel {
	case ZoomOverview:
		return 40 // Compact overview
	case ZoomWorld:
		return 64 // Readable labels
	case ZoomRegion:
		return 80
	case ZoomCity:
		return 100
	case ZoomStreet:
		return 120
	case ZoomInterior:
		return 140
	default:
		return 64
	}
}

// Pan moves the camera by the given delta (inverted so arrow keys move camera, not content)
func (m *MapView) Pan(dx, dy float64) {
	m.panX -= dx
	m.panY -= dy
}

// ZoomIn increases the zoom level
func (m *MapView) ZoomIn() {
	if m.zoomLevel < ZoomInterior {
		m.zoomLevel++
	}
}

// ZoomOut decreases the zoom level
func (m *MapView) ZoomOut() {
	if m.zoomLevel > ZoomOverview {
		m.zoomLevel--
	}
}

// ZoomLevel returns the current zoom level
func (m *MapView) ZoomLevel() int {
	return int(m.zoomLevel)
}

// PanX returns the current X pan offset
func (m *MapView) PanX() float64 {
	return m.panX
}

// PanY returns the current Y pan offset
func (m *MapView) PanY() float64 {
	return m.panY
}

// RevealTile marks a tile as revealed (fog of war cleared)
func (m *MapView) RevealTile(path string) {
	if t, exists := m.tileMap[path]; exists {
		t.Reveal()
	}
}

// ViewMode returns the current view mode.
func (m *MapView) ViewMode() ViewMode {
	return m.viewMode
}

// SetViewMode sets the current view mode.
func (m *MapView) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// ToggleViewMode switches between Directory and Dataflow views.
func (m *MapView) ToggleViewMode() {
	if m.viewMode == ViewDirectory {
		m.viewMode = ViewDataflow
	} else {
		m.viewMode = ViewDirectory
	}
}

// ConnectionGraph returns the connection graph for this map view.
func (m *MapView) ConnectionGraph() *connection.Graph {
	return m.connectionGraph
}

// SetConnectionGraph sets the connection graph for belt visualization.
// The graph should contain connections using relative paths matching tile RelPath().
func (m *MapView) SetConnectionGraph(graph *connection.Graph) {
	m.connectionGraph = graph
}

// AddConnection adds a single connection to the graph.
// from and to should be relative paths matching tile RelPath().
func (m *MapView) AddConnection(from, to string, connType connection.Type) *connection.Connection {
	return m.connectionGraph.AddNew(from, to, connType)
}
