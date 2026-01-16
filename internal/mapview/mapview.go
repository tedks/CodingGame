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
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/tile"
)

// ZoomLevel represents the current zoom level (1-5)
type ZoomLevel int

const (
	ZoomWorld    ZoomLevel = 1 // Repository root, top-level dirs
	ZoomRegion   ZoomLevel = 2 // Package/module clusters
	ZoomCity     ZoomLevel = 3 // Individual packages with buildings
	ZoomStreet   ZoomLevel = 4 // Files within a package
	ZoomInterior ZoomLevel = 5 // Single file contents

	// Rendering constants
	TileBorderSpacing = 2  // Pixels between tiles
	BorderColorBoost  = 20 // Amount to lighten border color
)

// MapView manages the map visualization with tiles and fog of war
type MapView struct {
	projectPath string
	width       int
	height      int

	// Camera state
	panX      float64
	panY      float64
	zoomLevel ZoomLevel

	// Tile system
	tiles   []*tile.Tile
	tileMap map[string]*tile.Tile // Fast lookup by path

	// Colors
	bgColor       color.RGBA
	gridColor     color.RGBA
	fogColor      color.RGBA
	revealedColor color.RGBA
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

	return &MapView{
		projectPath:   projectPath,
		width:         width,
		height:        height,
		panX:          0,
		panY:          0,
		zoomLevel:     ZoomWorld,
		tiles:         tiles,
		tileMap:       tileMap,
		bgColor:       color.RGBA{20, 20, 30, 255},
		gridColor:     color.RGBA{60, 60, 80, 255},
		fogColor:      color.RGBA{40, 40, 50, 200},
		revealedColor: color.RGBA{100, 120, 150, 255},
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

// Update updates the map view state
func (m *MapView) Update() {
	// Future: Handle animations, fog reveal transitions, etc.
}

// Draw renders the map view
func (m *MapView) Draw(screen *ebiten.Image, offsetX, offsetY int) {
	// Create sub-image for map area
	mapArea := screen.SubImage(ebiten.NewImage(m.width, m.height)).(*ebiten.Image)

	// Fill background
	mapArea.Fill(ebiten.ColorScale{}.Scale(
		float32(m.bgColor.R)/255,
		float32(m.bgColor.G)/255,
		float32(m.bgColor.B)/255,
		1,
	).Apply())

	// Calculate tile size based on zoom level
	tileSize := m.getTileSize()
	tilesPerRow := m.width / int(tileSize)

	// Draw tiles
	for i, t := range m.tiles {
		// Calculate tile position in grid
		col := i % tilesPerRow
		row := i / tilesPerRow

		// Apply camera pan
		x := float64(col)*tileSize + m.panX + float64(offsetX)
		y := float64(row)*tileSize + m.panY + float64(offsetY)

		// Skip if outside visible area
		if x+tileSize < 0 || x > float64(m.width) || y+tileSize < 0 || y > float64(m.height) {
			continue
		}

		// Draw tile
		m.drawTile(mapArea, t, float32(x), float32(y), float32(tileSize))
	}

	// Draw grid lines
	m.drawGrid(mapArea, offsetX, offsetY, tileSize)
}

// drawTile renders a single tile
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

// drawGrid draws the grid lines
func (m *MapView) drawGrid(screen *ebiten.Image, offsetX, offsetY int, tileSize float64) {
	// Draw vertical grid lines
	for x := m.panX + float64(offsetX); x < float64(m.width); x += tileSize {
		if x >= 0 {
			vector.StrokeLine(screen, float32(x), 0, float32(x), float32(m.height), 1, m.gridColor, false)
		}
	}

	// Draw horizontal grid lines
	for y := m.panY + float64(offsetY); y < float64(m.height); y += tileSize {
		if y >= 0 {
			vector.StrokeLine(screen, 0, float32(y), float32(m.width), float32(y), 1, m.gridColor, false)
		}
	}
}

// getTileSize returns the tile size in pixels for current zoom level
func (m *MapView) getTileSize() float64 {
	switch m.zoomLevel {
	case ZoomWorld:
		return 40
	case ZoomRegion:
		return 60
	case ZoomCity:
		return 80
	case ZoomStreet:
		return 100
	case ZoomInterior:
		return 120
	default:
		return 80
	}
}

// Pan moves the camera by the given delta
func (m *MapView) Pan(dx, dy float64) {
	m.panX += dx
	m.panY += dy
}

// ZoomIn increases the zoom level
func (m *MapView) ZoomIn() {
	if m.zoomLevel < ZoomInterior {
		m.zoomLevel++
	}
}

// ZoomOut decreases the zoom level
func (m *MapView) ZoomOut() {
	if m.zoomLevel > ZoomWorld {
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
