// Package belt provides rendering for dependency connections as Factorio-style belts.
// Belts are animated lines that flow from source files to consumer files,
// visualizing the dependency graph.
//
// Visual properties:
// - Color: Based on connection type (import, inheritance, etc.)
// - Width: Based on coupling strength (more symbols = thicker)
// - Animation: UV-scroll effect showing data flow direction
// - Special: Circular dependencies pulse red
package belt

import (
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/connection"
)

// Renderer draws belts (dependency connections) between tiles.
type Renderer struct {
	// Colors for different connection types
	importColor      color.RGBA
	inheritanceColor color.RGBA
	compositionColor color.RGBA
	callColor        color.RGBA
	circularColor    color.RGBA

	// Animation state
	startTime  time.Time
	frameCount int64 // Frame counter to avoid floating-point precision issues

	// Rendering parameters
	minWidth    float32 // Minimum belt width
	maxWidth    float32 // Maximum belt width
	maxStrength int     // Strength value that maps to maxWidth
	animSpeed   float64 // Animation speed (pixels per second)
	dashLength  float32 // Length of dashes for animation
	dashGap     float32 // Gap between dashes
}

// NewRenderer creates a new belt renderer with default settings.
func NewRenderer() *Renderer {
	return &Renderer{
		importColor:      color.RGBA{100, 150, 200, 200}, // Blue
		inheritanceColor: color.RGBA{200, 150, 100, 200}, // Orange
		compositionColor: color.RGBA{150, 200, 100, 200}, // Green
		callColor:        color.RGBA{200, 100, 200, 200}, // Purple
		circularColor:    color.RGBA{255, 80, 80, 255},   // Red

		startTime: time.Now(),

		minWidth:    1.0,
		maxWidth:    6.0,
		maxStrength: 10, // Strength >= 10 gets maximum width
		animSpeed:   30.0,
		dashLength:  8.0,
		dashGap:     4.0,
	}
}

// TilePosition represents the screen position and size of a tile.
type TilePosition struct {
	X, Y   float32 // Top-left corner
	Width  float32
	Height float32
}

// Center returns the center point of the tile.
func (tp TilePosition) Center() (float32, float32) {
	return tp.X + tp.Width/2, tp.Y + tp.Height/2
}

// Draw renders all connections in the graph as belts.
//
// Parameters:
//   - screen: The ebiten image to draw on
//   - graph: The connection graph to visualize
//   - tilePositions: Map from file path to screen position
//   - offsetX, offsetY: Screen offset for the map view
//
// Connections whose endpoints are not in tilePositions are skipped.
func (r *Renderer) Draw(
	screen *ebiten.Image,
	graph *connection.Graph,
	tilePositions map[string]TilePosition,
	offsetX, offsetY int,
) {
	if graph == nil {
		return
	}

	// Calculate animation offset based on time
	// Use modulo on elapsed time to prevent unbounded growth
	elapsed := time.Since(r.startTime).Seconds()
	dashPeriod := float64(r.dashLength + r.dashGap)
	// Reset elapsed periodically to avoid floating-point precision degradation
	elapsed = math.Mod(elapsed, dashPeriod*1000) // Reset every 1000 periods
	animOffset := float32(math.Mod(elapsed*r.animSpeed, dashPeriod))

	// Draw all connections
	for _, conn := range graph.All() {
		from, to := conn.Endpoints()

		// Get positions for both endpoints
		fromPos, fromExists := tilePositions[from]
		toPos, toExists := tilePositions[to]

		if !fromExists || !toExists {
			// Skip connections where we can't find the tiles
			continue
		}

		// Skip self-references (see CodingGame-dmv for future rendering)
		if conn.IsSelfReference() {
			continue
		}

		// Get center points
		fromX, fromY := fromPos.Center()
		toX, toY := toPos.Center()

		// Apply offset
		fromX += float32(offsetX)
		fromY += float32(offsetY)
		toX += float32(offsetX)
		toY += float32(offsetY)

		// Determine color and width
		beltColor := r.getColor(conn)
		beltWidth := r.getWidth(conn)

		// Draw the belt
		r.drawBelt(screen, fromX, fromY, toX, toY, beltColor, beltWidth, animOffset)
	}
}

// getColor returns the appropriate color for a connection.
func (r *Renderer) getColor(conn *connection.Connection) color.RGBA {
	// Circular dependencies override type color
	if conn.IsCircular() {
		return r.circularColor
	}

	switch conn.Type() {
	case connection.TypeImport:
		return r.importColor
	case connection.TypeInheritance:
		return r.inheritanceColor
	case connection.TypeComposition:
		return r.compositionColor
	case connection.TypeCall:
		return r.callColor
	default:
		return r.importColor
	}
}

// getWidth returns the belt width based on coupling strength.
func (r *Renderer) getWidth(conn *connection.Connection) float32 {
	strength := conn.Strength()
	if strength <= 0 {
		return r.minWidth
	}
	if strength >= r.maxStrength {
		return r.maxWidth
	}

	// Linear interpolation between min and max
	t := float32(strength) / float32(r.maxStrength)
	return r.minWidth + t*(r.maxWidth-r.minWidth)
}

// drawBelt draws a single belt between two points with animation.
func (r *Renderer) drawBelt(
	screen *ebiten.Image,
	fromX, fromY, toX, toY float32,
	col color.RGBA,
	width float32,
	animOffset float32,
) {
	// Calculate the total length
	dx := toX - fromX
	dy := toY - fromY
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if length < 1 {
		return // Too short to draw
	}

	// Draw as dashed line for animation effect
	r.drawDashedLine(screen, fromX, fromY, toX, toY, col, width, animOffset, length)

	// Draw arrowhead at the destination
	r.drawArrowhead(screen, fromX, fromY, toX, toY, col, width)
}

// drawDashedLine draws a dashed line with animation offset.
func (r *Renderer) drawDashedLine(
	screen *ebiten.Image,
	fromX, fromY, toX, toY float32,
	col color.RGBA,
	width float32,
	animOffset float32,
	length float32,
) {
	// Calculate direction vector
	dx := (toX - fromX) / length
	dy := (toY - fromY) / length

	// Draw dashes
	dashPeriod := r.dashLength + r.dashGap
	start := -animOffset // Start offset for animation

	for start < length {
		// Calculate dash segment
		dashStart := start
		dashEnd := start + r.dashLength

		// Clamp to valid range
		if dashStart < 0 {
			dashStart = 0
		}
		if dashEnd > length {
			dashEnd = length
		}

		// Only draw if we have a valid segment
		if dashEnd > dashStart {
			x1 := fromX + dx*dashStart
			y1 := fromY + dy*dashStart
			x2 := fromX + dx*dashEnd
			y2 := fromY + dy*dashEnd

			vector.StrokeLine(screen, x1, y1, x2, y2, width, col, false)
		}

		start += dashPeriod
	}
}

// drawArrowhead draws an arrowhead at the destination point.
func (r *Renderer) drawArrowhead(
	screen *ebiten.Image,
	fromX, fromY, toX, toY float32,
	col color.RGBA,
	width float32,
) {
	// Calculate direction
	dx := toX - fromX
	dy := toY - fromY
	length := float32(math.Sqrt(float64(dx*dx + dy*dy)))

	if length < 1 {
		return
	}

	// Normalize direction
	dx /= length
	dy /= length

	// Arrowhead size proportional to belt width
	arrowLen := width * 3
	arrowWidth := width * 2

	// Arrowhead tip is at destination
	tipX := toX
	tipY := toY

	// Calculate arrow back points
	// Perpendicular vector
	perpX := -dy
	perpY := dx

	// Back point
	backX := tipX - dx*arrowLen
	backY := tipY - dy*arrowLen

	// Wing points
	wing1X := backX + perpX*arrowWidth
	wing1Y := backY + perpY*arrowWidth
	wing2X := backX - perpX*arrowWidth
	wing2Y := backY - perpY*arrowWidth

	// Draw arrowhead as two lines
	vector.StrokeLine(screen, tipX, tipY, wing1X, wing1Y, width, col, false)
	vector.StrokeLine(screen, tipX, tipY, wing2X, wing2Y, width, col, false)
}

// SetColors configures the colors for different connection types.
func (r *Renderer) SetColors(
	importCol, inheritanceCol, compositionCol, callCol, circularCol color.RGBA,
) {
	r.importColor = importCol
	r.inheritanceColor = inheritanceCol
	r.compositionColor = compositionCol
	r.callColor = callCol
	r.circularColor = circularCol
}

// SetWidthRange configures the min/max width for belts.
func (r *Renderer) SetWidthRange(min, max float32, maxStrength int) {
	r.minWidth = min
	r.maxWidth = max
	r.maxStrength = maxStrength
}

// SetAnimationSpeed configures the animation speed in pixels per second.
func (r *Renderer) SetAnimationSpeed(speed float64) {
	r.animSpeed = speed
}

// SetDashPattern configures the dash pattern for belt animation.
func (r *Renderer) SetDashPattern(dashLength, dashGap float32) {
	r.dashLength = dashLength
	r.dashGap = dashGap
}
