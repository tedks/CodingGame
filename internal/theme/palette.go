package theme

import "image/color"

// Shared palette for UI renderers. Treat these as read-only values.
var (
	HeaderBackground   = color.RGBA{R: 0x1A, G: 0x1A, B: 0x2E, A: 0xFF}
	CardBackground     = color.RGBA{R: 0x2A, G: 0x2A, B: 0x3E, A: 0xFF}
	UsageBarBackground = color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF}

	StatusNeutral = color.RGBA{R: 0x75, G: 0x75, B: 0x75, A: 0xFF}
	StatusSuccess = color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF}
	StatusWarning = color.RGBA{R: 0xF5, G: 0x7F, B: 0x17, A: 0xFF}
	StatusInfo    = color.RGBA{R: 0x19, G: 0x76, B: 0xD2, A: 0xFF}
	StatusError   = color.RGBA{R: 0xC6, G: 0x28, B: 0x28, A: 0xFF}

	ResourcePanelBackground = color.RGBA{R: 30, G: 30, B: 40, A: 255}
	ResourceBarBackground   = color.RGBA{R: 50, G: 50, B: 60, A: 255}
	ResourceText            = color.RGBA{R: 200, G: 200, B: 220, A: 255}
	ResourceContext         = color.RGBA{R: 100, G: 150, B: 255, A: 255}
	ResourceCost            = color.RGBA{R: 255, G: 200, B: 100, A: 255}
	ResourceCoverage        = color.RGBA{R: 100, G: 255, B: 150, A: 255}

	CapabilityBackground     = color.RGBA{R: 20, G: 25, B: 30, A: 255}
	CapabilityNodeBackground = color.RGBA{R: 40, G: 45, B: 50, A: 255}
	CapabilityNodeDisabled   = color.RGBA{R: 30, G: 30, B: 30, A: 200}

	CapabilityDomainCore        = color.RGBA{R: 60, G: 80, B: 100, A: 255}
	CapabilityDomainBuild       = color.RGBA{R: 80, G: 100, B: 60, A: 255}
	CapabilityDomainVersionCtrl = color.RGBA{R: 100, G: 80, B: 60, A: 255}
	CapabilityDomainDeployment  = color.RGBA{R: 80, G: 60, B: 100, A: 255}
	CapabilityDomainAnalysis    = color.RGBA{R: 60, G: 100, B: 80, A: 255}

	CapabilityTypeTool        = color.RGBA{R: 100, G: 180, B: 255, A: 255}
	CapabilityTypeMCP         = color.RGBA{R: 255, G: 180, B: 100, A: 255}
	CapabilityTypeCommand     = color.RGBA{R: 180, G: 100, B: 255, A: 255}
	CapabilityTypeIntegration = color.RGBA{R: 100, G: 255, B: 180, A: 255}
)
