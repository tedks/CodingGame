package harness

import (
	"fmt"
	"os/exec"
	"sync"
)

// Registry manages available harness implementations
type Registry struct {
	mu        sync.RWMutex
	factories map[string]HarnessFactory
	defs      map[string]HarnessDefinition
}

// NewRegistry creates a new harness registry
func NewRegistry() *Registry {
	r := &Registry{
		factories: make(map[string]HarnessFactory),
		defs:      make(map[string]HarnessDefinition),
	}

	// Load default definitions
	for _, def := range DefaultHarnessDefinitions() {
		r.defs[def.Name] = def
	}

	return r
}

// Register adds a harness factory to the registry
func (r *Registry) Register(name string, factory HarnessFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// RegisterWithDefinition adds a harness factory with its definition
func (r *Registry) RegisterWithDefinition(def HarnessDefinition, factory HarnessFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[def.Name] = factory
	r.defs[def.Name] = def
}

// Create instantiates a harness by name
func (r *Registry) Create(name string) (Harness, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown harness: %s", name)
	}

	return factory(), nil
}

// Available returns the names of all registered harnesses
func (r *Registry) Available() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// Defined returns the names of all defined harnesses (including unregistered)
func (r *Registry) Defined() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.defs))
	for name := range r.defs {
		names = append(names, name)
	}
	return names
}

// Definition returns the definition for a harness
func (r *Registry) Definition(name string) (HarnessDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, ok := r.defs[name]
	return def, ok
}

// IsRegistered checks if a harness factory is registered
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.factories[name]
	return ok
}

// IsInstalled checks if the harness CLI is installed on the system
func (r *Registry) IsInstalled(name string) bool {
	r.mu.RLock()
	def, ok := r.defs[name]
	r.mu.RUnlock()

	if !ok {
		return false
	}

	// Check if command exists in PATH
	_, err := exec.LookPath(def.Command)
	return err == nil
}

// InstalledHarnesses returns all harnesses that are both registered and installed
func (r *Registry) InstalledHarnesses() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	installed := make([]string, 0)
	for name := range r.factories {
		def, ok := r.defs[name]
		if !ok {
			continue
		}
		if _, err := exec.LookPath(def.Command); err == nil {
			installed = append(installed, name)
		}
	}
	return installed
}

// Models returns the available models for a harness
func (r *Registry) Models(name string) []Model {
	r.mu.RLock()
	def, ok := r.defs[name]
	r.mu.RUnlock()

	if !ok {
		return nil
	}

	models := make([]Model, len(def.Models))
	for i, m := range def.Models {
		models[i] = Model{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Default:     m.Default,
		}
	}
	return models
}

// DefaultModel returns the default model for a harness
func (r *Registry) DefaultModel(name string) string {
	r.mu.RLock()
	def, ok := r.defs[name]
	r.mu.RUnlock()

	if !ok {
		return ""
	}
	return def.DefaultModel
}

// GetCapabilities returns the capabilities for a harness
func (r *Registry) GetCapabilities(name string) *Capabilities {
	r.mu.RLock()
	def, ok := r.defs[name]
	r.mu.RUnlock()

	if !ok {
		return nil
	}

	models := make([]Model, len(def.Models))
	for i, m := range def.Models {
		models[i] = Model{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Default:     m.Default,
		}
	}

	return &Capabilities{
		SupportedModels:   models,
		SupportsHooks:     def.Features.Hooks,
		SupportsMCP:       def.Features.MCP,
		SupportsStreaming: def.Features.Streaming,
		SupportsResume:    def.Features.Resume,
	}
}

// HarnessInfo provides information about a harness
type HarnessInfo struct {
	Name         string
	DisplayName  string
	Description  string
	Installed    bool
	Registered   bool
	DefaultModel string
	Models       []Model
	Features     HarnessFeatures
}

// Info returns detailed information about a harness
func (r *Registry) Info(name string) *HarnessInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	def, hasDef := r.defs[name]
	_, hasFactory := r.factories[name]

	if !hasDef {
		return nil
	}

	installed := false
	if _, err := exec.LookPath(def.Command); err == nil {
		installed = true
	}

	models := make([]Model, len(def.Models))
	for i, m := range def.Models {
		models[i] = Model{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Default:     m.Default,
		}
	}

	return &HarnessInfo{
		Name:         def.Name,
		DisplayName:  def.DisplayName,
		Description:  def.Description,
		Installed:    installed,
		Registered:   hasFactory,
		DefaultModel: def.DefaultModel,
		Models:       models,
		Features:     def.Features,
	}
}

// AllInfo returns information about all known harnesses
func (r *Registry) AllInfo() []HarnessInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]HarnessInfo, 0, len(r.defs))
	for name := range r.defs {
		if info := r.Info(name); info != nil {
			infos = append(infos, *info)
		}
	}
	return infos
}
