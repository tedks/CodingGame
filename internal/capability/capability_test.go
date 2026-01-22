package capability

import (
	"testing"
)

func TestNewCapability(t *testing.T) {
	cap := NewCapability("test-id", "Test Name", TypeTool, DomainCore)

	if cap.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got %q", cap.ID)
	}
	if cap.Name != "Test Name" {
		t.Errorf("expected Name 'Test Name', got %q", cap.Name)
	}
	if cap.Type != TypeTool {
		t.Errorf("expected Type TypeTool, got %v", cap.Type)
	}
	if cap.Domain != DomainCore {
		t.Errorf("expected Domain DomainCore, got %v", cap.Domain)
	}
	if !cap.Enabled {
		t.Error("expected Enabled to be true by default")
	}
}

func TestCapabilityBuilders(t *testing.T) {
	cap := NewCapability("id", "Name", TypeMCP, DomainBuild).
		WithDescription("Test description").
		WithSource("test-source").
		WithEnabled(false)

	if cap.Description != "Test description" {
		t.Errorf("expected Description 'Test description', got %q", cap.Description)
	}
	if cap.Source != "test-source" {
		t.Errorf("expected Source 'test-source', got %q", cap.Source)
	}
	if cap.Enabled {
		t.Error("expected Enabled to be false")
	}
}

func TestDomainString(t *testing.T) {
	tests := []struct {
		domain   Domain
		expected string
	}{
		{DomainCore, "Core"},
		{DomainBuild, "Build"},
		{DomainVersionCtrl, "Version Control"},
		{DomainDeployment, "Deployment"},
		{DomainAnalysis, "Analysis"},
		{Domain("unknown"), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.domain.String(); got != tc.expected {
			t.Errorf("Domain(%q).String() = %q, expected %q", tc.domain, got, tc.expected)
		}
	}
}

func TestCapabilityTypeString(t *testing.T) {
	tests := []struct {
		capType  CapabilityType
		expected string
	}{
		{TypeTool, "Tool"},
		{TypeMCP, "MCP Server"},
		{TypeCommand, "Command"},
		{TypeIntegration, "Integration"},
		{CapabilityType("unknown"), "unknown"},
	}

	for _, tc := range tests {
		if got := tc.capType.String(); got != tc.expected {
			t.Errorf("CapabilityType(%q).String() = %q, expected %q", tc.capType, got, tc.expected)
		}
	}
}

func TestAllDomains(t *testing.T) {
	domains := AllDomains()
	if len(domains) != 5 {
		t.Errorf("expected 5 domains, got %d", len(domains))
	}

	expectedOrder := []Domain{
		DomainCore,
		DomainBuild,
		DomainVersionCtrl,
		DomainDeployment,
		DomainAnalysis,
	}

	for i, expected := range expectedOrder {
		if domains[i] != expected {
			t.Errorf("domain at index %d: expected %v, got %v", i, expected, domains[i])
		}
	}
}
