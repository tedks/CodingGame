package capability

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Renderer draws the capability inventory (tech tree view).
type Renderer struct {
	// Colors for different domains
	coreColor        color.RGBA
	buildColor       color.RGBA
	versionCtrlColor color.RGBA
	deploymentColor  color.RGBA
	analysisColor    color.RGBA

	// Colors for different types (node border accents)
	toolColor        color.RGBA
	mcpColor         color.RGBA
	commandColor     color.RGBA
	integrationColor color.RGBA

	// Background color
	backgroundColor color.RGBA

	// Layout parameters
	columnWidth  int
	nodeHeight   int
	nodeMargin   int
	headerHeight int
	padding      int
}

// NewRenderer creates a new capability renderer with default settings.
func NewRenderer() *Renderer {
	return &Renderer{
		// Domain colors (column headers)
		coreColor:        color.RGBA{60, 80, 100, 255},
		buildColor:       color.RGBA{80, 100, 60, 255},
		versionCtrlColor: color.RGBA{100, 80, 60, 255},
		deploymentColor:  color.RGBA{80, 60, 100, 255},
		analysisColor:    color.RGBA{60, 100, 80, 255},

		// Type colors (node border accents)
		toolColor:        color.RGBA{100, 180, 255, 255},
		mcpColor:         color.RGBA{255, 180, 100, 255},
		commandColor:     color.RGBA{180, 100, 255, 255},
		integrationColor: color.RGBA{100, 255, 180, 255},

		// Background
		backgroundColor: color.RGBA{20, 25, 30, 255},

		// Layout
		columnWidth:  200,
		nodeHeight:   40,
		nodeMargin:   8,
		headerHeight: 30,
		padding:      16,
	}
}

// Draw renders the capability inventory.
//
// Parameters:
//   - screen: The ebiten image to draw on
//   - capabilities: The capabilities to display
//   - x, y: Top-left position to start drawing
//   - width, height: Available space for rendering
func (r *Renderer) Draw(screen *ebiten.Image, capabilities []*Capability, x, y, width, height int) {
	// Fill background
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(height),
		r.backgroundColor,
		false,
	)

	// Group capabilities by domain
	byDomain := make(map[Domain][]*Capability)
	for _, cap := range capabilities {
		byDomain[cap.Domain] = append(byDomain[cap.Domain], cap)
	}

	// Draw title
	ebitenutil.DebugPrintAt(screen, "Capability Inventory", x+r.padding, y+r.padding)

	// Calculate column positions
	domains := AllDomains()
	startYOffset, actualColumnWidth, columnHeight, ok := r.columnLayout(width, height, domains)
	if !ok {
		return
	}
	startY := y + startYOffset
	contentWidth := actualColumnWidth - r.nodeMargin

	// Draw each domain column
	for i, domain := range domains {
		colX := x + r.padding + i*actualColumnWidth
		caps := byDomain[domain]
		r.drawDomainColumn(screen, domain, caps, colX, startY, contentWidth, columnHeight)
	}

	// Draw legend at bottom
	r.drawLegend(screen, x+r.padding, y+height-40, width-2*r.padding)
}

func (r *Renderer) columnLayout(width, height int, domains []Domain) (startYOffset, columnWidth, columnHeight int, ok bool) {
	if len(domains) == 0 {
		return 0, 0, 0, false
	}

	availableWidth := width - 2*r.padding
	if availableWidth <= 0 {
		return 0, 0, 0, false
	}

	actualColumnWidth := availableWidth / len(domains)
	if actualColumnWidth <= 0 {
		return 0, 0, 0, false
	}
	if actualColumnWidth > r.columnWidth {
		actualColumnWidth = r.columnWidth
	}
	if actualColumnWidth-r.nodeMargin <= 0 {
		return 0, 0, 0, false
	}

	startYOffset = r.padding + 20 + r.padding
	columnHeight = height - startYOffset - r.padding
	if columnHeight <= 0 {
		return 0, 0, 0, false
	}

	return startYOffset, actualColumnWidth, columnHeight, true
}

// drawDomainColumn draws a single domain column.
func (r *Renderer) drawDomainColumn(screen *ebiten.Image, domain Domain, capabilities []*Capability, x, y, width, maxHeight int) {
	// Draw domain header background
	headerBg := r.getDomainColor(domain)
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(r.headerHeight),
		headerBg,
		false,
	)

	// Draw domain name centered in header
	domainName := domain.String()
	textX := x + (width-len(domainName)*7)/2 // Approximate centering (7px per char)
	textY := y + (r.headerHeight-14)/2       // 14px line height
	ebitenutil.DebugPrintAt(screen, domainName, textX, textY)

	// Draw capability nodes
	nodeY := y + r.headerHeight + r.nodeMargin
	for _, cap := range capabilities {
		if nodeY+r.nodeHeight > y+maxHeight {
			// No more space, draw overflow indicator
			ebitenutil.DebugPrintAt(screen, "...", x+width/2-10, nodeY)
			break
		}
		r.drawCapabilityNode(screen, cap, x, nodeY, width)
		nodeY += r.nodeHeight + r.nodeMargin
	}

	// Show count if empty
	if len(capabilities) == 0 {
		ebitenutil.DebugPrintAt(screen, "(none)", x+width/2-20, y+r.headerHeight+r.nodeMargin)
	}
}

// drawCapabilityNode draws a single capability.
func (r *Renderer) drawCapabilityNode(screen *ebiten.Image, cap *Capability, x, y, width int) {
	// Background
	bgColor := color.RGBA{40, 45, 50, 255}
	if !cap.Enabled {
		bgColor = color.RGBA{30, 30, 30, 200}
	}
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		float32(width), float32(r.nodeHeight),
		bgColor,
		false,
	)

	// Left border accent by type
	accentColor := r.getTypeColor(cap.Type)
	vector.DrawFilledRect(
		screen,
		float32(x), float32(y),
		4, float32(r.nodeHeight),
		accentColor,
		false,
	)

	// Name (top line)
	nameY := y + 6
	ebitenutil.DebugPrintAt(screen, cap.Name, x+r.nodeMargin+4, nameY)

	// Type indicator (smaller, below name)
	typeText := cap.Type.String()
	typeY := y + 22
	ebitenutil.DebugPrintAt(screen, typeText, x+r.nodeMargin+4, typeY)
}

// drawLegend draws the type color legend.
func (r *Renderer) drawLegend(screen *ebiten.Image, x, y, _ int) {
	legendItems := []struct {
		name  string
		color color.RGBA
	}{
		{"Tool", r.toolColor},
		{"MCP", r.mcpColor},
		{"Command", r.commandColor},
		{"Integration", r.integrationColor},
	}

	currentX := x
	for _, item := range legendItems {
		// Color swatch
		vector.DrawFilledRect(
			screen,
			float32(currentX), float32(y+2),
			10, 10,
			item.color,
			false,
		)
		// Label
		ebitenutil.DebugPrintAt(screen, item.name, currentX+14, y)
		currentX += len(item.name)*7 + 30
	}
}

// getDomainColor returns the background color for a domain.
func (r *Renderer) getDomainColor(domain Domain) color.RGBA {
	switch domain {
	case DomainCore:
		return r.coreColor
	case DomainBuild:
		return r.buildColor
	case DomainVersionCtrl:
		return r.versionCtrlColor
	case DomainDeployment:
		return r.deploymentColor
	case DomainAnalysis:
		return r.analysisColor
	default:
		return r.coreColor
	}
}

// getTypeColor returns the accent color for a capability type.
func (r *Renderer) getTypeColor(capType CapabilityType) color.RGBA {
	switch capType {
	case TypeTool:
		return r.toolColor
	case TypeMCP:
		return r.mcpColor
	case TypeCommand:
		return r.commandColor
	case TypeIntegration:
		return r.integrationColor
	default:
		return r.toolColor
	}
}

// SetDomainColors configures the background colors for each domain.
func (r *Renderer) SetDomainColors(core, build, versionCtrl, deployment, analysis color.RGBA) {
	r.coreColor = core
	r.buildColor = build
	r.versionCtrlColor = versionCtrl
	r.deploymentColor = deployment
	r.analysisColor = analysis
}

// SetTypeColors configures the accent colors for each capability type.
func (r *Renderer) SetTypeColors(tool, mcp, command, integration color.RGBA) {
	r.toolColor = tool
	r.mcpColor = mcp
	r.commandColor = command
	r.integrationColor = integration
}

// SetLayout configures the layout parameters.
func (r *Renderer) SetLayout(columnWidth, nodeHeight, nodeMargin, headerHeight, padding int) {
	r.columnWidth = columnWidth
	r.nodeHeight = nodeHeight
	r.nodeMargin = nodeMargin
	r.headerHeight = headerHeight
	r.padding = padding
}
