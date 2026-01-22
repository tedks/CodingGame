package capability

import (
	"image/color"
	"testing"
)

func TestNewRenderer(t *testing.T) {
	r := NewRenderer()
	if r == nil {
		t.Fatal("NewRenderer returned nil")
	}

	// Verify default values are sensible
	if r.columnWidth <= 0 {
		t.Error("columnWidth should be positive")
	}
	if r.nodeHeight <= 0 {
		t.Error("nodeHeight should be positive")
	}
	if r.headerHeight <= 0 {
		t.Error("headerHeight should be positive")
	}
}

func TestRendererGetDomainColor(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		domain   Domain
		expected color.RGBA
	}{
		{DomainCore, r.coreColor},
		{DomainBuild, r.buildColor},
		{DomainVersionCtrl, r.versionCtrlColor},
		{DomainDeployment, r.deploymentColor},
		{DomainAnalysis, r.analysisColor},
		{Domain("unknown"), r.coreColor}, // Default fallback
	}

	for _, tc := range tests {
		got := r.getDomainColor(tc.domain)
		if got != tc.expected {
			t.Errorf("getDomainColor(%v) = %v, expected %v", tc.domain, got, tc.expected)
		}
	}
}

func TestRendererGetTypeColor(t *testing.T) {
	r := NewRenderer()

	tests := []struct {
		capType  CapabilityType
		expected color.RGBA
	}{
		{TypeTool, r.toolColor},
		{TypeMCP, r.mcpColor},
		{TypeCommand, r.commandColor},
		{TypeIntegration, r.integrationColor},
		{CapabilityType("unknown"), r.toolColor}, // Default fallback
	}

	for _, tc := range tests {
		got := r.getTypeColor(tc.capType)
		if got != tc.expected {
			t.Errorf("getTypeColor(%v) = %v, expected %v", tc.capType, got, tc.expected)
		}
	}
}

func TestRendererSetDomainColors(t *testing.T) {
	r := NewRenderer()

	newCore := color.RGBA{255, 0, 0, 255}
	newBuild := color.RGBA{0, 255, 0, 255}
	newVC := color.RGBA{0, 0, 255, 255}
	newDeploy := color.RGBA{255, 255, 0, 255}
	newAnalysis := color.RGBA{255, 0, 255, 255}

	r.SetDomainColors(newCore, newBuild, newVC, newDeploy, newAnalysis)

	if r.getDomainColor(DomainCore) != newCore {
		t.Error("SetDomainColors did not update core color")
	}
	if r.getDomainColor(DomainBuild) != newBuild {
		t.Error("SetDomainColors did not update build color")
	}
	if r.getDomainColor(DomainVersionCtrl) != newVC {
		t.Error("SetDomainColors did not update version control color")
	}
	if r.getDomainColor(DomainDeployment) != newDeploy {
		t.Error("SetDomainColors did not update deployment color")
	}
	if r.getDomainColor(DomainAnalysis) != newAnalysis {
		t.Error("SetDomainColors did not update analysis color")
	}
}

func TestRendererSetTypeColors(t *testing.T) {
	r := NewRenderer()

	newTool := color.RGBA{100, 100, 100, 255}
	newMCP := color.RGBA{150, 150, 150, 255}
	newCommand := color.RGBA{200, 200, 200, 255}
	newIntegration := color.RGBA{50, 50, 50, 255}

	r.SetTypeColors(newTool, newMCP, newCommand, newIntegration)

	if r.getTypeColor(TypeTool) != newTool {
		t.Error("SetTypeColors did not update tool color")
	}
	if r.getTypeColor(TypeMCP) != newMCP {
		t.Error("SetTypeColors did not update MCP color")
	}
	if r.getTypeColor(TypeCommand) != newCommand {
		t.Error("SetTypeColors did not update command color")
	}
	if r.getTypeColor(TypeIntegration) != newIntegration {
		t.Error("SetTypeColors did not update integration color")
	}
}

func TestRendererSetLayout(t *testing.T) {
	r := NewRenderer()

	r.SetLayout(300, 50, 10, 40, 20)

	if r.columnWidth != 300 {
		t.Errorf("columnWidth = %d, expected 300", r.columnWidth)
	}
	if r.nodeHeight != 50 {
		t.Errorf("nodeHeight = %d, expected 50", r.nodeHeight)
	}
	if r.nodeMargin != 10 {
		t.Errorf("nodeMargin = %d, expected 10", r.nodeMargin)
	}
	if r.headerHeight != 40 {
		t.Errorf("headerHeight = %d, expected 40", r.headerHeight)
	}
	if r.padding != 20 {
		t.Errorf("padding = %d, expected 20", r.padding)
	}
}

func TestRendererColorsDifferByDomain(t *testing.T) {
	r := NewRenderer()

	// All domain colors should be different
	domains := AllDomains()
	colors := make(map[color.RGBA]Domain)

	for _, d := range domains {
		c := r.getDomainColor(d)
		if existing, found := colors[c]; found {
			t.Errorf("domains %v and %v have the same color", existing, d)
		}
		colors[c] = d
	}
}

func TestRendererColorsDifferByType(t *testing.T) {
	r := NewRenderer()

	// All type colors should be different
	types := []CapabilityType{TypeTool, TypeMCP, TypeCommand, TypeIntegration}
	colors := make(map[color.RGBA]CapabilityType)

	for _, ct := range types {
		c := r.getTypeColor(ct)
		if existing, found := colors[c]; found {
			t.Errorf("types %v and %v have the same color", existing, ct)
		}
		colors[c] = ct
	}
}
