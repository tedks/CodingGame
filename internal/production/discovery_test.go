package production

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDiscovererName(t *testing.T) {
	d := NewConfigDiscoverer("/tmp/test")
	if d.Name() != "production-config" {
		t.Errorf("expected name 'production-config', got %q", d.Name())
	}
}

func TestConfigDiscovererWithNoConfigs(t *testing.T) {
	// Use a temp directory with no configs
	tmpDir := t.TempDir()

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	// Should return empty (no error, just no services)
	// Note: might find global configs if they exist on the machine
	_ = services
}

func TestConfigDiscovererParsesConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test .production.json file
	config := `{
		"services": {
			"api-gateway": {
				"type": "http",
				"endpoint": "http://localhost:8080",
				"healthPath": "/health",
				"dependencies": ["user-service", "order-service"]
			},
			"user-service": {
				"type": "grpc",
				"endpoint": "localhost:9090"
			},
			"postgres": {
				"type": "database",
				"endpoint": "localhost:5432"
			}
		}
	}`

	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	// Should find 3 services
	if len(services) != 3 {
		t.Errorf("expected 3 services, got %d", len(services))
	}

	// Find services by name
	svcMap := make(map[string]*Service)
	for _, svc := range services {
		svcMap[svc.Name] = svc
	}

	// Check api-gateway
	if apiGw, exists := svcMap["api-gateway"]; !exists {
		t.Error("expected api-gateway service not found")
	} else {
		if apiGw.Type != ServiceTypeHTTP {
			t.Errorf("api-gateway should be HTTP type, got %v", apiGw.Type)
		}
		if apiGw.Endpoint != "http://localhost:8080" {
			t.Errorf("expected endpoint 'http://localhost:8080', got %q", apiGw.Endpoint)
		}
		if len(apiGw.Dependencies) != 2 {
			t.Errorf("expected 2 dependencies, got %d", len(apiGw.Dependencies))
		}
		if apiGw.Source != configPath {
			t.Errorf("expected source %q, got %q", configPath, apiGw.Source)
		}
	}

	// Check user-service
	if userSvc, exists := svcMap["user-service"]; !exists {
		t.Error("expected user-service not found")
	} else {
		if userSvc.Type != ServiceTypeGRPC {
			t.Errorf("user-service should be gRPC type, got %v", userSvc.Type)
		}
	}

	// Check postgres
	if pg, exists := svcMap["postgres"]; !exists {
		t.Error("expected postgres service not found")
	} else {
		if pg.Type != ServiceTypeDatabase {
			t.Errorf("postgres should be Database type, got %v", pg.Type)
		}
	}
}

func TestConfigDiscovererWatchPaths(t *testing.T) {
	d := NewConfigDiscoverer("/tmp/test-project")
	paths := d.WatchPaths()

	// Should include both global and project paths
	if len(paths) == 0 {
		t.Error("expected at least one watch path")
	}

	// Check for project-local path
	found := false
	for _, p := range paths {
		if p == "/tmp/test-project/.production.json" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected project .production.json in watch paths")
	}
}

func TestInferServiceType(t *testing.T) {
	tests := []struct {
		input    string
		expected ServiceType
	}{
		{"http", ServiceTypeHTTP},
		{"HTTP", ServiceTypeHTTP},
		{"rest", ServiceTypeHTTP},
		{"api", ServiceTypeHTTP},
		{"grpc", ServiceTypeGRPC},
		{"GRPC", ServiceTypeGRPC},
		{"database", ServiceTypeDatabase},
		{"db", ServiceTypeDatabase},
		{"postgres", ServiceTypeDatabase},
		{"mysql", ServiceTypeDatabase},
		{"redis", ServiceTypeDatabase},
		{"queue", ServiceTypeQueue},
		{"kafka", ServiceTypeQueue},
		{"rabbitmq", ServiceTypeQueue},
		{"sqs", ServiceTypeQueue},
		{"kubernetes", ServiceTypeKubernetes},
		{"k8s", ServiceTypeKubernetes},
		{"deployment", ServiceTypeKubernetes},
		{"unknown", ServiceTypeHTTP}, // Default
		{"", ServiceTypeHTTP},        // Default
	}

	for _, tc := range tests {
		got := inferServiceType(tc.input)
		if got != tc.expected {
			t.Errorf("inferServiceType(%q) = %v, expected %v", tc.input, got, tc.expected)
		}
	}
}

func TestSanitizeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"api-gateway", "api-gateway"},
		{"API Gateway", "api-gateway"},
		{"user_service", "user-service"},
		{"MyService", "myservice"},
	}

	for _, tc := range tests {
		got := sanitizeID(tc.input)
		if got != tc.expected {
			t.Errorf("sanitizeID(%q) = %q, expected %q", tc.input, got, tc.expected)
		}
	}
}

// Error path tests

func TestConfigDiscovererMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a malformed JSON file
	malformedConfig := `{ this is not valid json `
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(malformedConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()

	// Should not return error (graceful degradation), but also no services from malformed file
	if err != nil {
		t.Fatalf("Discover() should not return error for malformed JSON: %v", err)
	}

	// Check that no services were found from the malformed file
	for _, svc := range services {
		if svc.Source == configPath {
			t.Error("should not have found services from malformed JSON")
		}
	}
}

func TestConfigDiscovererEmptyJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty JSON file
	emptyConfig := `{}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()

	if err != nil {
		t.Fatalf("Discover() should not return error for empty JSON: %v", err)
	}

	// Check that no services were found
	for _, svc := range services {
		if svc.Source == configPath {
			t.Error("should not have found services from empty JSON")
		}
	}
}

func TestConfigDiscovererEmptyServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create JSON with empty services
	config := `{"services": {}}`
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()

	if err != nil {
		t.Fatalf("Discover() should not return error for empty services: %v", err)
	}

	// Check that no services were found
	for _, svc := range services {
		if svc.Source == configPath {
			t.Error("should not have found services from empty services object")
		}
	}
}

func TestConfigDiscovererUnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file that's not readable
	configPath := filepath.Join(tmpDir, ".production.json")
	if err := os.WriteFile(configPath, []byte(`{}`), 0000); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	d := NewConfigDiscoverer(tmpDir)
	services, err := d.Discover()

	// Should not error (graceful degradation)
	if err != nil {
		t.Fatalf("Discover() should not return error for unreadable file: %v", err)
	}

	// Restore permissions for cleanup
	os.Chmod(configPath, 0644)

	_ = services
}
