package capability

import (
	"sort"
	"sync"
)

// RegistryListener receives notifications about registry changes.
type RegistryListener interface {
	// OnCapabilitiesChanged is called when capabilities are added, removed, or updated.
	OnCapabilitiesChanged(capabilities []*Capability)
}

// Registry manages discovered capabilities and notifies listeners of changes.
type Registry struct {
	mu sync.RWMutex

	capabilities map[string]*Capability
	discoverers  []Discoverer
	listeners    []RegistryListener
}

// NewRegistry creates a new capability registry.
func NewRegistry() *Registry {
	return &Registry{
		capabilities: make(map[string]*Capability),
	}
}

// RegisterDiscoverer adds a discoverer to the registry.
func (r *Registry) RegisterDiscoverer(d Discoverer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.discoverers = append(r.discoverers, d)
}

// AddListener registers a listener for capability changes.
func (r *Registry) AddListener(l RegistryListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners = append(r.listeners, l)
}

// RemoveListener unregisters a listener.
func (r *Registry) RemoveListener(l RegistryListener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, listener := range r.listeners {
		if listener == l {
			r.listeners = append(r.listeners[:i], r.listeners[i+1:]...)
			return
		}
	}
}

// Refresh runs all discoverers and updates the capability list.
// Returns the number of capabilities discovered.
func (r *Registry) Refresh() int {
	r.mu.Lock()
	discoverers := make([]Discoverer, len(r.discoverers))
	copy(discoverers, r.discoverers)
	r.mu.Unlock()

	// Collect capabilities from all discoverers
	newCaps := make(map[string]*Capability)
	for _, d := range discoverers {
		caps, err := d.Discover()
		if err != nil {
			// Log error but continue with other discoverers
			continue
		}
		for _, cap := range caps {
			newCaps[cap.ID] = cap
		}
	}

	// Update registry
	r.mu.Lock()
	r.capabilities = newCaps
	listeners := make([]RegistryListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.mu.Unlock()

	// Notify listeners
	all := r.GetAll()
	for _, l := range listeners {
		go l.OnCapabilitiesChanged(all)
	}

	return len(newCaps)
}

// GetAll returns all capabilities sorted by domain then name.
func (r *Registry) GetAll() []*Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	caps := make([]*Capability, 0, len(r.capabilities))
	for _, cap := range r.capabilities {
		caps = append(caps, cap)
	}

	// Sort by domain first, then by name within domain
	sort.Slice(caps, func(i, j int) bool {
		if caps[i].Domain != caps[j].Domain {
			return domainOrder(caps[i].Domain) < domainOrder(caps[j].Domain)
		}
		return caps[i].Name < caps[j].Name
	})

	return caps
}

// GetByDomain returns capabilities for a specific domain.
func (r *Registry) GetByDomain(domain Domain) []*Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var caps []*Capability
	for _, cap := range r.capabilities {
		if cap.Domain == domain {
			caps = append(caps, cap)
		}
	}

	sort.Slice(caps, func(i, j int) bool {
		return caps[i].Name < caps[j].Name
	})

	return caps
}

// GetByType returns capabilities for a specific type.
func (r *Registry) GetByType(capType CapabilityType) []*Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var caps []*Capability
	for _, cap := range r.capabilities {
		if cap.Type == capType {
			caps = append(caps, cap)
		}
	}

	sort.Slice(caps, func(i, j int) bool {
		return caps[i].Name < caps[j].Name
	})

	return caps
}

// Get returns a specific capability by ID, or nil if not found.
func (r *Registry) Get(id string) *Capability {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.capabilities[id]
}

// Count returns the total number of capabilities.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.capabilities)
}

// CountByDomain returns counts for each domain.
func (r *Registry) CountByDomain() map[Domain]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	counts := make(map[Domain]int)
	for _, cap := range r.capabilities {
		counts[cap.Domain]++
	}
	return counts
}

// WatchPaths returns all paths that should be monitored for changes.
func (r *Registry) WatchPaths() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var paths []string
	for _, d := range r.discoverers {
		paths = append(paths, d.WatchPaths()...)
	}
	return paths
}

// domainOrder returns the sort order for a domain.
func domainOrder(d Domain) int {
	switch d {
	case DomainCore:
		return 0
	case DomainBuild:
		return 1
	case DomainVersionCtrl:
		return 2
	case DomainDeployment:
		return 3
	case DomainAnalysis:
		return 4
	default:
		return 99
	}
}
