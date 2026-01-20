package production

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Discoverer finds production services from various sources.
type Discoverer interface {
	// Name returns a human-readable name for this discoverer.
	Name() string
	// Discover returns all services found by this discoverer.
	Discover() ([]*Service, error)
	// WatchPaths returns file paths that should be monitored for changes.
	WatchPaths() []string
}

// ConfigDiscoverer discovers services from .production.json config files.
// Config files can be located in:
//   - Project root (.production.json)
//   - User home (~/.production.json)
type ConfigDiscoverer struct {
	projectPath string
}

// NewConfigDiscoverer creates a discoverer for the given project.
func NewConfigDiscoverer(projectPath string) *ConfigDiscoverer {
	return &ConfigDiscoverer{
		projectPath: projectPath,
	}
}

// Name implements Discoverer.
func (d *ConfigDiscoverer) Name() string {
	return "production-config"
}

// Discover implements Discoverer.
// It reads .production.json files and parses service definitions.
func (d *ConfigDiscoverer) Discover() ([]*Service, error) {
	var services []*Service

	// Check project-local config
	projectConfig := filepath.Join(d.projectPath, ".production.json")
	if projectServices, err := d.parseConfig(projectConfig); err == nil {
		services = append(services, projectServices...)
	}

	// Check home directory config
	if home, err := os.UserHomeDir(); err == nil {
		homeConfig := filepath.Join(home, ".production.json")
		if homeServices, err := d.parseConfig(homeConfig); err == nil {
			services = append(services, homeServices...)
		}
	}

	return services, nil
}

// WatchPaths implements Discoverer.
func (d *ConfigDiscoverer) WatchPaths() []string {
	paths := []string{
		filepath.Join(d.projectPath, ".production.json"),
	}

	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".production.json"))
	}

	return paths
}

// productionConfig represents the structure of a .production.json file.
type productionConfig struct {
	Services map[string]serviceConfig `json:"services"`
}

// serviceConfig represents a single service entry in the config.
type serviceConfig struct {
	Type         string   `json:"type"`
	Endpoint     string   `json:"endpoint"`
	HealthPath   string   `json:"healthPath,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// parseConfig reads and parses a single config file.
func (d *ConfigDiscoverer) parseConfig(path string) ([]*Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config productionConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	var services []*Service
	for name, sc := range config.Services {
		svc := NewService(
			sanitizeID(name),
			name,
			inferServiceType(sc.Type),
			sc.Endpoint,
		)
		svc.Source = path
		if len(sc.Dependencies) > 0 {
			svc.Dependencies = sc.Dependencies
		}
		services = append(services, svc)
	}

	return services, nil
}

// sanitizeID creates a safe ID from a service name.
func sanitizeID(name string) string {
	// Convert to lowercase and replace spaces/special chars with dashes
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	return id
}

// inferServiceType maps a string type to a ServiceType constant.
func inferServiceType(typeStr string) ServiceType {
	switch strings.ToLower(typeStr) {
	case "http", "rest", "api":
		return ServiceTypeHTTP
	case "grpc":
		return ServiceTypeGRPC
	case "database", "db", "postgres", "mysql", "redis":
		return ServiceTypeDatabase
	case "queue", "kafka", "rabbitmq", "sqs":
		return ServiceTypeQueue
	case "kubernetes", "k8s", "deployment":
		return ServiceTypeKubernetes
	default:
		return ServiceTypeHTTP
	}
}
