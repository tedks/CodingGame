package capability

// Discoverer is the interface for capability discovery implementations.
// Different discoverers find capabilities from different sources.
type Discoverer interface {
	// Name returns a human-readable name for this discoverer.
	Name() string

	// Discover finds and returns all capabilities from this source.
	// Returns an empty slice if no capabilities are found.
	Discover() ([]*Capability, error)

	// WatchPaths returns file paths that should be monitored for changes.
	// Returns nil if this discoverer doesn't support watching.
	WatchPaths() []string
}

// BuiltinToolDiscoverer discovers built-in Claude Code tools.
type BuiltinToolDiscoverer struct{}

// NewBuiltinToolDiscoverer creates a new built-in tool discoverer.
func NewBuiltinToolDiscoverer() *BuiltinToolDiscoverer {
	return &BuiltinToolDiscoverer{}
}

// Name implements Discoverer.
func (d *BuiltinToolDiscoverer) Name() string {
	return "builtin-tools"
}

// Discover implements Discoverer.
func (d *BuiltinToolDiscoverer) Discover() ([]*Capability, error) {
	return builtinTools(), nil
}

// WatchPaths implements Discoverer.
// Built-in tools don't change, so nothing to watch.
func (d *BuiltinToolDiscoverer) WatchPaths() []string {
	return nil
}

// builtinTools returns the list of built-in Claude Code tools.
func builtinTools() []*Capability {
	return []*Capability{
		// Core domain - file operations
		NewCapability("read", "Read", TypeTool, DomainCore).
			WithDescription("Read file contents").
			WithSource("builtin"),
		NewCapability("write", "Write", TypeTool, DomainCore).
			WithDescription("Write file contents").
			WithSource("builtin"),
		NewCapability("edit", "Edit", TypeTool, DomainCore).
			WithDescription("Edit files with precise replacements").
			WithSource("builtin"),
		NewCapability("glob", "Glob", TypeTool, DomainCore).
			WithDescription("Find files by pattern").
			WithSource("builtin"),

		// Core domain - execution and web
		NewCapability("bash", "Bash", TypeTool, DomainCore).
			WithDescription("Execute shell commands").
			WithSource("builtin"),
		NewCapability("task", "Task", TypeTool, DomainCore).
			WithDescription("Spawn subagent for complex tasks").
			WithSource("builtin"),
		NewCapability("webfetch", "WebFetch", TypeTool, DomainCore).
			WithDescription("Fetch and process web content").
			WithSource("builtin"),
		NewCapability("websearch", "WebSearch", TypeTool, DomainCore).
			WithDescription("Search the web").
			WithSource("builtin"),

		// Analysis domain
		NewCapability("grep", "Grep", TypeTool, DomainAnalysis).
			WithDescription("Search file contents with regex").
			WithSource("builtin"),

		// Version control domain (via bash, but conceptually here)
		NewCapability("notebook_edit", "NotebookEdit", TypeTool, DomainCore).
			WithDescription("Edit Jupyter notebooks").
			WithSource("builtin"),
	}
}
