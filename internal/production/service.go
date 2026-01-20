// Package production provides visualization of deployed services and production
// environments. It represents services as "cities" with population (traffic),
// happiness (SLO compliance), and weather (system health).
//
// This follows the project philosophy: everything shown is real data from
// actual production systems, not artificial game stats.
package production

import "time"

// ServiceType indicates what kind of production service this is.
type ServiceType string

const (
	// ServiceTypeHTTP is a generic HTTP/REST service.
	ServiceTypeHTTP ServiceType = "http"
	// ServiceTypeGRPC is a gRPC service.
	ServiceTypeGRPC ServiceType = "grpc"
	// ServiceTypeDatabase is a database service.
	ServiceTypeDatabase ServiceType = "database"
	// ServiceTypeQueue is a message queue service.
	ServiceTypeQueue ServiceType = "queue"
	// ServiceTypeKubernetes is a Kubernetes deployment.
	ServiceTypeKubernetes ServiceType = "kubernetes"
)

// HealthStatus represents the current health of a service.
type HealthStatus string

const (
	// HealthHealthy indicates all systems nominal.
	HealthHealthy HealthStatus = "healthy"
	// HealthDegraded indicates reduced performance but still functioning.
	HealthDegraded HealthStatus = "degraded"
	// HealthUnhealthy indicates service is failing or down.
	HealthUnhealthy HealthStatus = "unhealthy"
	// HealthUnknown indicates health status cannot be determined.
	HealthUnknown HealthStatus = "unknown"
)

// Weather represents the overall system conditions as a weather metaphor.
// This makes production metrics intuitive at a glance.
type Weather string

const (
	// WeatherClear indicates all systems nominal - low error rates, normal traffic.
	WeatherClear Weather = "clear"
	// WeatherCloudy indicates minor issues - slightly elevated latency or errors.
	WeatherCloudy Weather = "cloudy"
	// WeatherStorm indicates elevated error rates or outages.
	WeatherStorm Weather = "storm"
	// WeatherDrought indicates unusually low traffic (may indicate issues).
	WeatherDrought Weather = "drought"
	// WeatherFlood indicates traffic spike (may cause stress).
	WeatherFlood Weather = "flood"
)

// TrafficMetrics holds real-time traffic statistics for a service.
type TrafficMetrics struct {
	// RequestsPerSecond is the current request rate.
	RequestsPerSecond float64
	// ErrorRate is the percentage of requests resulting in errors (0-100).
	ErrorRate float64
	// LatencyP50 is the median response latency.
	LatencyP50 time.Duration
	// LatencyP99 is the 99th percentile response latency.
	LatencyP99 time.Duration
	// ActiveConnections is the number of active client connections.
	ActiveConnections int
}

// Service represents a deployed production service (a "city" in the game metaphor).
type Service struct {
	// ID is a unique identifier for the service.
	ID string
	// Name is the human-readable service name.
	Name string
	// Type indicates what kind of service this is.
	Type ServiceType
	// Endpoint is the URL or address used to reach the service.
	Endpoint string

	// Health is the current health status.
	Health HealthStatus
	// Weather is the derived weather condition based on metrics.
	Weather Weather
	// Metrics contains real-time traffic statistics.
	Metrics TrafficMetrics

	// Dependencies lists other services this service calls.
	Dependencies []string
	// Dependents lists services that call this service.
	Dependents []string

	// Source indicates where this service was discovered (config file path).
	Source string
	// LastChecked is when the health was last verified.
	LastChecked time.Time
	// LastError is the most recent error message, if any.
	LastError string
}

// NewService creates a new service with the given parameters.
func NewService(id, name string, serviceType ServiceType, endpoint string) *Service {
	return &Service{
		ID:       id,
		Name:     name,
		Type:     serviceType,
		Endpoint: endpoint,
		Health:   HealthUnknown,
		Weather:  WeatherClear,
	}
}

// WithDependencies sets the service dependencies.
func (s *Service) WithDependencies(deps ...string) *Service {
	s.Dependencies = deps
	return s
}

// WithSource sets the discovery source.
func (s *Service) WithSource(source string) *Service {
	s.Source = source
	return s
}

// UpdateMetrics updates the traffic metrics and derives weather condition.
func (s *Service) UpdateMetrics(metrics TrafficMetrics) {
	s.Metrics = metrics
	s.Weather = deriveWeather(metrics)
}

// UpdateHealth updates the health status.
func (s *Service) UpdateHealth(status HealthStatus, errMsg string) {
	s.Health = status
	s.LastChecked = time.Now()
	if errMsg != "" {
		s.LastError = errMsg
	}
}

// deriveWeather calculates weather based on traffic metrics.
// Thresholds are based on industry-standard SLO targets.
func deriveWeather(m TrafficMetrics) Weather {
	// Storm: High error rate (>5%) or very high latency (>1s p99)
	if m.ErrorRate > 5 || m.LatencyP99 > time.Second {
		return WeatherStorm
	}

	// Flood: Very high request rate (this is relative, using >1000 RPS as default)
	if m.RequestsPerSecond > 1000 {
		return WeatherFlood
	}

	// Drought: Very low traffic (could indicate routing issues)
	if m.RequestsPerSecond < 0.1 && m.RequestsPerSecond >= 0 {
		return WeatherDrought
	}

	// Cloudy: Elevated but acceptable error rate (1-5%) or latency (500ms-1s)
	if m.ErrorRate > 1 || m.LatencyP99 > 500*time.Millisecond {
		return WeatherCloudy
	}

	return WeatherClear
}

// AllServiceTypes returns all valid service types.
func AllServiceTypes() []ServiceType {
	return []ServiceType{
		ServiceTypeHTTP,
		ServiceTypeGRPC,
		ServiceTypeDatabase,
		ServiceTypeQueue,
		ServiceTypeKubernetes,
	}
}

// AllHealthStatuses returns all valid health statuses.
func AllHealthStatuses() []HealthStatus {
	return []HealthStatus{
		HealthHealthy,
		HealthDegraded,
		HealthUnhealthy,
		HealthUnknown,
	}
}

// AllWeathers returns all valid weather conditions.
func AllWeathers() []Weather {
	return []Weather{
		WeatherClear,
		WeatherCloudy,
		WeatherStorm,
		WeatherDrought,
		WeatherFlood,
	}
}
