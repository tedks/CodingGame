package production

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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
	prodPadding      = 20
	prodCityWidth    = 180
	prodCityHeight   = 100
	prodCitySpacing  = 30
	prodCitiesPerRow = 4
	prodHeaderHeight = 60
)

// Color scheme for health statuses
var healthColors = map[HealthStatus]color.RGBA{
	HealthHealthy:   {R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}, // Green
	HealthDegraded:  {R: 0xF5, G: 0x7F, B: 0x17, A: 0xFF}, // Orange
	HealthUnhealthy: {R: 0xC6, G: 0x28, B: 0x28, A: 0xFF}, // Red
	HealthUnknown:   {R: 0x75, G: 0x75, B: 0x75, A: 0xFF}, // Gray
}

// Weather symbols (ASCII art style)
var weatherSymbols = map[Weather]string{
	WeatherClear:   "sun",
	WeatherCloudy:  "cloud",
	WeatherStorm:   "storm",
	WeatherDrought: "dry",
	WeatherFlood:   "flood",
}

// Draw renders the production view to the screen.
// Parameters:
//   - screen: the Ebiten image to draw to
//   - services: the services to display
//   - x, y: the top-left position to start drawing
//   - width, height: the available drawing area
func (r *Renderer) Draw(screen *ebiten.Image, services []*Service, x, y, width, height int) {
	// Draw header with summary
	r.drawHeader(screen, services, x, y, width)

	// If no services, show empty state
	if len(services) == 0 {
		r.drawEmptyState(screen, x, y+prodHeaderHeight, width, height-prodHeaderHeight)
		return
	}

	// Draw services as cities
	startY := y + prodHeaderHeight + prodPadding
	for i, svc := range services {
		row := i / prodCitiesPerRow
		col := i % prodCitiesPerRow

		cityX := x + prodPadding + col*(prodCityWidth+prodCitySpacing)
		cityY := startY + row*(prodCityHeight+prodCitySpacing)

		// Check if we're still in bounds
		if cityY+prodCityHeight > y+height {
			break
		}

		r.drawCity(screen, svc, cityX, cityY)
	}
}

// drawHeader renders the summary header with weather overview.
func (r *Renderer) drawHeader(screen *ebiten.Image, services []*Service, x, y, width int) {
	// Draw background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(prodHeaderHeight),
		color.RGBA{R: 0x1A, G: 0x1A, B: 0x2E, A: 0xFF}, false)

	// Draw title
	ebitenutil.DebugPrintAt(screen, "PRODUCTION REALM", x+prodPadding, y+10)

	// Calculate health summary
	healthCounts := make(map[HealthStatus]int)
	weatherCounts := make(map[Weather]int)
	for _, svc := range services {
		healthCounts[svc.Health]++
		weatherCounts[svc.Weather]++
	}

	// Draw health summary
	summaryY := y + 30
	summaryX := x + prodPadding
	healthy := healthCounts[HealthHealthy]
	degraded := healthCounts[HealthDegraded]
	unhealthy := healthCounts[HealthUnhealthy]

	summary := fmt.Sprintf("Services: %d | Healthy: %d | Degraded: %d | Unhealthy: %d",
		len(services), healthy, degraded, unhealthy)
	ebitenutil.DebugPrintAt(screen, summary, summaryX, summaryY)

	// Draw weather summary
	weatherX := x + width - 300
	weatherSummary := fmt.Sprintf("Clear: %d | Cloudy: %d | Storm: %d",
		weatherCounts[WeatherClear], weatherCounts[WeatherCloudy], weatherCounts[WeatherStorm])
	ebitenutil.DebugPrintAt(screen, weatherSummary, weatherX, summaryY)
}

// drawEmptyState renders a message when no services are configured.
func (r *Renderer) drawEmptyState(screen *ebiten.Image, x, y, width, height int) {
	msg := "No production services configured"
	hint := "Add a .production.json file to monitor services"

	// Center the messages
	msgX := x + width/2 - len(msg)*3
	msgY := y + height/2 - 20
	hintX := x + width/2 - len(hint)*3
	hintY := y + height/2

	ebitenutil.DebugPrintAt(screen, msg, msgX, msgY)
	ebitenutil.DebugPrintAt(screen, hint, hintX, hintY)
}

// drawCity renders a single service as a "city" card.
func (r *Renderer) drawCity(screen *ebiten.Image, svc *Service, x, y int) {
	// Get color based on health
	healthColor := healthColors[svc.Health]

	// Draw city background
	bgColor := color.RGBA{R: 0x2A, G: 0x2A, B: 0x3E, A: 0xFF}
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(prodCityWidth), float32(prodCityHeight),
		bgColor, false)

	// Draw health indicator border
	vector.StrokeRect(screen, float32(x), float32(y), float32(prodCityWidth), float32(prodCityHeight),
		2, healthColor, false)

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
