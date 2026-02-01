package production

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/tedks/CodingGame/internal/theme"
)

// Renderer draws the production environment visualization.
// Services are displayed as "cities" on a world map with weather conditions.
type Renderer struct{}

// NewRenderer creates a new production renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

// Color scheme for health statuses
var healthColors = map[HealthStatus]color.RGBA{
	HealthHealthy:   theme.StatusSuccess,
	HealthDegraded:  theme.StatusWarning,
	HealthUnhealthy: theme.StatusError,
	HealthUnknown:   theme.StatusNeutral,
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
		r.drawEmptyState(screen, x, y+theme.PanelHeaderHeight, width, height-theme.PanelHeaderHeight)
		return
	}

	// Draw services as cities
	startY := y + theme.PanelHeaderHeight + theme.PanelPadding
	for i, svc := range services {
		row := i / theme.ProductionCitiesPerRow
		col := i % theme.ProductionCitiesPerRow

		cityX := x + theme.PanelPadding + col*(theme.ProductionCityWidth+theme.ProductionCitySpacing)
		cityY := startY + row*(theme.ProductionCityHeight+theme.ProductionCitySpacing)

		// Check if we're still in bounds
		if cityY+theme.ProductionCityHeight > y+height {
			break
		}

		r.drawCity(screen, svc, cityX, cityY)
	}
}

func weatherSummaryX(x, width int) (int, bool) {
	return theme.RightAlignedX(x, width, theme.ProductionWeatherSummaryWidth, theme.PanelPadding)
}

// drawHeader renders the summary header with weather overview.
func (r *Renderer) drawHeader(screen *ebiten.Image, services []*Service, x, y, width int) {
	// Draw background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(theme.PanelHeaderHeight),
		theme.HeaderBackground, false)

	// Draw title
	ebitenutil.DebugPrintAt(screen, "PRODUCTION REALM", x+theme.PanelPadding, y+theme.PanelHeaderTitleOffsetY)

	// Calculate health summary
	healthCounts := make(map[HealthStatus]int)
	weatherCounts := make(map[Weather]int)
	for _, svc := range services {
		healthCounts[svc.Health]++
		weatherCounts[svc.Weather]++
	}

	// Draw health summary
	summaryY := y + theme.PanelHeaderSummaryOffsetY
	summaryX := x + theme.PanelPadding
	healthy := healthCounts[HealthHealthy]
	degraded := healthCounts[HealthDegraded]
	unhealthy := healthCounts[HealthUnhealthy]

	summary := fmt.Sprintf("Services: %d | Healthy: %d | Degraded: %d | Unhealthy: %d",
		len(services), healthy, degraded, unhealthy)
	ebitenutil.DebugPrintAt(screen, summary, summaryX, summaryY)

	// Draw weather summary
	weatherX, ok := weatherSummaryX(x, width)
	if ok {
		weatherSummary := fmt.Sprintf("Clear: %d | Cloudy: %d | Storm: %d",
			weatherCounts[WeatherClear], weatherCounts[WeatherCloudy], weatherCounts[WeatherStorm])
		ebitenutil.DebugPrintAt(screen, weatherSummary, weatherX, summaryY)
	}
}

// drawEmptyState renders a message when no services are configured.
func (r *Renderer) drawEmptyState(screen *ebiten.Image, x, y, width, height int) {
	msg := "No production services configured"
	hint := "Add a .production.json file to monitor services"

	// Center the messages
	msgX := theme.CenterTextX(x, width, msg, theme.CompactTextCharWidth)
	msgY := y + height/2 - theme.EmptyStateLineOffset
	hintX := theme.CenterTextX(x, width, hint, theme.CompactTextCharWidth)
	hintY := y + height/2

	ebitenutil.DebugPrintAt(screen, msg, msgX, msgY)
	ebitenutil.DebugPrintAt(screen, hint, hintX, hintY)
}

// drawCity renders a single service as a "city" card.
func (r *Renderer) drawCity(screen *ebiten.Image, svc *Service, x, y int) {
	// Get color based on health
	healthColor := healthColors[svc.Health]

	// Draw city background
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(theme.ProductionCityWidth), float32(theme.ProductionCityHeight),
		theme.CardBackground, false)

	// Draw health indicator border
	vector.StrokeRect(screen, float32(x), float32(y), float32(theme.ProductionCityWidth), float32(theme.ProductionCityHeight),
		theme.CardBorderWidth, healthColor, false)

	// Draw service name
	nameY := y + theme.ProductionNameOffsetY
	ebitenutil.DebugPrintAt(screen, svc.Name, x+theme.CardTextPadding, nameY)

	// Draw service type
	typeY := y + theme.ProductionTypeOffsetY
	typeStr := fmt.Sprintf("[%s]", svc.Type)
	ebitenutil.DebugPrintAt(screen, typeStr, x+theme.CardTextPadding, typeY)

	// Draw weather indicator
	weatherY := y + theme.ProductionWeatherOffsetY
	weatherStr := r.weatherDisplay(svc.Weather)
	ebitenutil.DebugPrintAt(screen, weatherStr, x+theme.CardTextPadding, weatherY)

	// Draw health status
	healthY := y + theme.ProductionHealthOffsetY
	healthStr := fmt.Sprintf("Status: %s", svc.Health)
	ebitenutil.DebugPrintAt(screen, healthStr, x+theme.CardTextPadding, healthY)

	// Draw traffic if available
	if svc.Metrics.RequestsPerSecond > 0 {
		trafficY := y + theme.ProductionTrafficOffsetY
		trafficStr := fmt.Sprintf("%.1f req/s", svc.Metrics.RequestsPerSecond)
		ebitenutil.DebugPrintAt(screen, trafficStr, x+theme.CardTextPadding, trafficY)
	}

	// Draw dependency count
	if len(svc.Dependencies) > 0 {
		depY := y + theme.ProductionDependencyOffsetY
		depStr := fmt.Sprintf("Deps: %d", len(svc.Dependencies))
		ebitenutil.DebugPrintAt(screen, depStr, x+theme.ProductionCityWidth-theme.ProductionDependencyInset, depY)
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
