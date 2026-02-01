package production

import (
	"sort"
	"sync"
)

// RegistryListener receives notifications about service changes.
type RegistryListener interface {
	// OnServicesChanged is called when services are added, removed, or updated.
	// The provided slice is read-only and must not be modified by listeners.
	// Listeners are called asynchronously in goroutines.
	OnServicesChanged(services []*Service)
}

// Registry manages discovered production services and notifies listeners of changes.
type Registry struct {
	mu sync.RWMutex

	services    map[string]*Service
	discoverers []Discoverer
	listeners   []RegistryListener
	lastErrors  map[string]error // Errors from last Refresh(), keyed by discoverer name
}

// NewRegistry creates a new production service registry.
func NewRegistry() *Registry {
	return &Registry{
		services: make(map[string]*Service),
	}
}

// RegisterDiscoverer adds a discoverer to the registry.
func (r *Registry) RegisterDiscoverer(d Discoverer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.discoverers = append(r.discoverers, d)
}

// AddListener registers a listener for service changes.
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

// Refresh runs all discoverers and updates the service list.
// Returns the number of services discovered.
// Errors from individual discoverers are stored and can be retrieved with LastErrors().
//
// Note: This method takes a snapshot of registered discoverers at the start.
// Discoverers added during Refresh() will not be included until the next Refresh().
func (r *Registry) Refresh() int {
	// Take a snapshot of discoverers to avoid holding lock during discovery
	r.mu.Lock()
	discoverers := make([]Discoverer, len(r.discoverers))
	copy(discoverers, r.discoverers)
	r.mu.Unlock()

	// Collect services from all discoverers
	newServices := make(map[string]*Service)
	errors := make(map[string]error)
	for _, d := range discoverers {
		services, err := d.Discover()
		if err != nil {
			// Track error but continue with other discoverers
			errors[d.Name()] = err
			continue
		}
		for _, svc := range services {
			newServices[svc.ID] = svc
		}
	}

	// Update registry
	r.mu.Lock()
	r.services = newServices
	r.lastErrors = errors
	listeners := make([]RegistryListener, len(r.listeners))
	copy(listeners, r.listeners)
	r.mu.Unlock()

	// Notify listeners (with panic recovery to prevent listener crashes from taking down the registry)
	all := r.GetAll()
	for _, l := range listeners {
		services := cloneServices(all)
		go func(listener RegistryListener, services []*Service) {
			defer func() {
				if rec := recover(); rec != nil {
					// Silently ignore panics from listeners
					// In production, this could be logged
				}
			}()
			listener.OnServicesChanged(services)
		}(l, services)
	}

	return len(newServices)
}

// LastErrors returns any errors from the most recent Refresh() call.
// The map is keyed by discoverer name. An empty map means no errors occurred.
func (r *Registry) LastErrors() map[string]error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.lastErrors == nil {
		return make(map[string]error)
	}

	// Return a copy to avoid concurrent access issues
	errs := make(map[string]error, len(r.lastErrors))
	for k, v := range r.lastErrors {
		errs[k] = v
	}
	return errs
}

// HasErrors returns true if the most recent Refresh() had any discoverer errors.
func (r *Registry) HasErrors() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.lastErrors) > 0
}

// GetAll returns all services sorted by name.
func (r *Registry) GetAll() []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]*Service, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, cloneService(svc))
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

// GetByType returns services of a specific type.
func (r *Registry) GetByType(serviceType ServiceType) []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []*Service
	for _, svc := range r.services {
		if svc.Type == serviceType {
			services = append(services, cloneService(svc))
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

// GetByHealth returns services with a specific health status.
func (r *Registry) GetByHealth(health HealthStatus) []*Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var services []*Service
	for _, svc := range r.services {
		if svc.Health == health {
			services = append(services, cloneService(svc))
		}
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services
}

// Get returns a specific service by ID, or nil if not found.
func (r *Registry) Get(id string) *Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneService(r.services[id])
}

// Count returns the total number of services.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.services)
}

// CountByHealth returns counts for each health status.
func (r *Registry) CountByHealth() map[HealthStatus]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	counts := make(map[HealthStatus]int)
	for _, svc := range r.services {
		counts[svc.Health]++
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

func cloneServices(services []*Service) []*Service {
	if services == nil {
		return nil
	}
	cloned := make([]*Service, len(services))
	for i, svc := range services {
		cloned[i] = cloneService(svc)
	}
	return cloned
}

func cloneService(service *Service) *Service {
	if service == nil {
		return nil
	}
	cloned := *service
	if service.Dependencies != nil {
		cloned.Dependencies = append([]string(nil), service.Dependencies...)
	}
	if service.Dependents != nil {
		cloned.Dependents = append([]string(nil), service.Dependents...)
	}
	return &cloned
}

// GetWeatherSummary returns a summary of weather conditions across all services.
func (r *Registry) GetWeatherSummary() map[Weather]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	summary := make(map[Weather]int)
	for _, svc := range r.services {
		summary[svc.Weather]++
	}
	return summary
}
