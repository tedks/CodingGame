package production

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/tedks/CodingGame/internal/ui"
)

// Renderer draws the production environment visualization.
// Services are displayed as "cities" on a world map with weather conditions.
type Renderer struct{}

// NewRenderer creates a new production renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Layout constants
const (
	prodCityWidth    = 180
	prodCityHeight   = 100
	prodCitySpacing  = 30
	prodCitiesPerRow = 4
)

// Color scheme for health statuses
var healthColors = map[HealthStatus]color.RGBA{
	HealthHealthy:   {R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}, // Green
	HealthDegraded:  {R: 0xF5, G: 0x7F, B: 0x17, A: 0xFF}, // Orange
	HealthUnhealthy: {R: 0xC6, G: 0x28, B: 0x28, A: 0xFF}, // Red
	HealthUnknown:   {R: 0x75, G: 0x75, B: 0x75, A: 0xFF}, // Gray
}

// Draw renders the production view to the screen.
// Parameters:
//   - screen: the Ebiten image to draw to
//   - services: the services to display
//   - x, y: the top-left position to start drawing
//   - width, height: the available drawing area
func (r *Renderer) Draw(screen *ebiten.Image, services []*Service, x, y, width, height int) {
	headerLayout := ui.DefaultHeaderLayout()

	// Draw header with summary
	r.drawHeader(screen, services, x, y, width, headerLayout)

	// If no services, show empty state
	if len(services) == 0 {
		ui.DrawEmptyState(screen, "No production services configured",
			"Add a .production.json file to monitor services",
			x, y+headerLayout.Height, width, height-headerLayout.Height)
		return
	}

	// Draw services as cities
	startY := y + headerLayout.Height + headerLayout.Padding
	for i, svc := range services {
		row := i / prodCitiesPerRow
		col := i % prodCitiesPerRow

		cityX := x + headerLayout.Padding + col*(prodCityWidth+prodCitySpacing)
		cityY := startY + row*(prodCityHeight+prodCitySpacing)

		// Check if we're still in bounds
		if cityY+prodCityHeight > y+height {
			break
		}

		r.drawCity(screen, svc, cityX, cityY)
	}
}

func weatherSummaryX(x, width int) (int, bool) {
	layout := ui.DefaultHeaderLayout()
	return ui.RightAlignedSummaryX(x, width, 300, layout.Padding)
}

// drawHeader renders the summary header with weather overview.
func (r *Renderer) drawHeader(screen *ebiten.Image, services []*Service, x, y, width int, layout ui.HeaderLayout) {
	ui.DrawHeader(screen, "PRODUCTION REALM", x, y, width, layout)

	// Calculate health summary
	healthCounts := make(map[HealthStatus]int)
	weatherCounts := make(map[Weather]int)
	for _, svc := range services {
		healthCounts[svc.Health]++
		weatherCounts[svc.Weather]++
	}

	// Draw health summary
	summaryY := y + layout.SummaryOffsetY
	healthy := healthCounts[HealthHealthy]
	degraded := healthCounts[HealthDegraded]
	unhealthy := healthCounts[HealthUnhealthy]

	summary := fmt.Sprintf("Services: %d | Healthy: %d | Degraded: %d | Unhealthy: %d",
		len(services), healthy, degraded, unhealthy)
	ui.DrawHeaderSummary(screen, summary, x, y, layout)

	// Draw weather summary
	weatherX, ok := weatherSummaryX(x, width)
	if ok {
		weatherSummary := fmt.Sprintf("Clear: %d | Cloudy: %d | Storm: %d",
			weatherCounts[WeatherClear], weatherCounts[WeatherCloudy], weatherCounts[WeatherStorm])
		ebitenutil.DebugPrintAt(screen, weatherSummary, weatherX, summaryY)
	}
}

// drawCity renders a single service as a "city" card.
func (r *Renderer) drawCity(screen *ebiten.Image, svc *Service, x, y int) {
	// Get color based on health
	healthColor := healthColors[svc.Health]

	// Draw city background
	ui.DrawCard(screen, x, y, prodCityWidth, prodCityHeight, ui.DefaultCardStyle(healthColor))

	// Draw service name
	nameY := y + 8
	ebitenutil.DebugPrintAt(screen, svc.Name, x+8, nameY)

	// Draw service type
	typeY := y + 24
	typeStr := fmt.Sprintf("[%s]", svc.Type)
	ebitenutil.DebugPrintAt(screen, typeStr, x+8, typeY)

	// Draw weather indicator
	weatherY := y + 40
	weatherStr := r.weatherDisplay(svc.Weather)
	ebitenutil.DebugPrintAt(screen, weatherStr, x+8, weatherY)

	// Draw health status
	healthY := y + 56
	healthStr := fmt.Sprintf("Status: %s", svc.Health)
	ebitenutil.DebugPrintAt(screen, healthStr, x+8, healthY)

	// Draw traffic if available
	if svc.Metrics.RequestsPerSecond > 0 {
		trafficY := y + 72
		trafficStr := fmt.Sprintf("%.1f req/s", svc.Metrics.RequestsPerSecond)
		ebitenutil.DebugPrintAt(screen, trafficStr, x+8, trafficY)
	}

	// Draw dependency count
	if len(svc.Dependencies) > 0 {
		depY := y + 72
		depStr := fmt.Sprintf("Deps: %d", len(svc.Dependencies))
		ebitenutil.DebugPrintAt(screen, depStr, x+prodCityWidth-60, depY)
	}
}

// weatherDisplay returns a text representation of weather.
func (r *Renderer) weatherDisplay(w Weather) string {
	switch w {
	case WeatherClear:
		return "Weather: Clear [*]"
	case WeatherCloudy:
		return "Weather: Cloudy [~]"
	case WeatherStorm:
		return "Weather: STORM [!]"
	case WeatherDrought:
		return "Weather: Drought [-]"
	case WeatherFlood:
		return "Weather: FLOOD [^]"
	default:
		return "Weather: Unknown [?]"
	}
}

// GetHealthColor returns the color associated with a health status.
// Exported for use by other renderers or tests.
func GetHealthColor(h HealthStatus) color.RGBA {
	if c, ok := healthColors[h]; ok {
		return c
	}
	return healthColors[HealthUnknown]
}
