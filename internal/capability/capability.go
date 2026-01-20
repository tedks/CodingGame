// Package capability provides discovery and visualization of Claude Code
// capabilities including tools, MCP servers, slash commands, and integrations.
//
// This is a descriptive visualization - it shows what capabilities are
// configured and available, not a progression system with unlocks.
package capability

// Domain categorizes capabilities by functional area.
type Domain string

const (
	// DomainCore contains fundamental file and search operations.
	DomainCore Domain = "core"
	// DomainBuild contains build system integrations.
	DomainBuild Domain = "build"
	// DomainVersionCtrl contains version control operations.
	DomainVersionCtrl Domain = "version_ctrl"
	// DomainDeployment contains CI/CD and cloud integrations.
	DomainDeployment Domain = "deployment"
	// DomainAnalysis contains linting, testing, and coverage tools.
	DomainAnalysis Domain = "analysis"
)

// AllDomains returns all capability domains in display order.
func AllDomains() []Domain {
	return []Domain{
		DomainCore,
		DomainBuild,
		DomainVersionCtrl,
		DomainDeployment,
		DomainAnalysis,
	}
}

// String returns the display name for a domain.
func (d Domain) String() string {
	switch d {
	case DomainCore:
		return "Core"
	case DomainBuild:
		return "Build"
	case DomainVersionCtrl:
		return "Version Control"
	case DomainDeployment:
		return "Deployment"
	case DomainAnalysis:
		return "Analysis"
	default:
		return string(d)
	}
}

// CapabilityType categorizes the kind of capability.
type CapabilityType string

const (
	// TypeTool is a built-in Claude Code tool.
	TypeTool CapabilityType = "tool"
	// TypeMCP is an MCP server integration.
	TypeMCP CapabilityType = "mcp"
	// TypeCommand is a slash command.
	TypeCommand CapabilityType = "command"
	// TypeIntegration is an external integration.
	TypeIntegration CapabilityType = "integration"
)

// String returns the display name for a capability type.
func (t CapabilityType) String() string {
	switch t {
	case TypeTool:
		return "Tool"
	case TypeMCP:
		return "MCP Server"
	case TypeCommand:
		return "Command"
	case TypeIntegration:
		return "Integration"
	default:
		return string(t)
	}
}

// Capability represents a single capability available to Claude Code.
type Capability struct {
	// ID is a unique identifier (e.g., "read", "github-mcp").
	ID string

	// Name is the human-readable display name.
	Name string

	// Type categorizes the kind of capability.
	Type CapabilityType

	// Domain categorizes the functional area.
	Domain Domain

	// Description provides a brief explanation of what this capability does.
	Description string

	// Source indicates where this capability was discovered
	// (e.g., "builtin", "~/.claude.json", ".mcp.json").
	Source string

	// Enabled indicates whether this capability is currently active.
	Enabled bool
}

// NewCapability creates a new capability with required fields.
func NewCapability(id, name string, capType CapabilityType, domain Domain) *Capability {
	return &Capability{
		ID:      id,
		Name:    name,
		Type:    capType,
		Domain:  domain,
		Enabled: true,
	}
}

// WithDescription sets the description and returns the capability for chaining.
func (c *Capability) WithDescription(desc string) *Capability {
	c.Description = desc
	return c
}

// WithSource sets the source and returns the capability for chaining.
func (c *Capability) WithSource(source string) *Capability {
	c.Source = source
	return c
}

// WithEnabled sets the enabled state and returns the capability for chaining.
func (c *Capability) WithEnabled(enabled bool) *Capability {
	c.Enabled = enabled
	return c
}
