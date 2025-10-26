package fetcher

import (
	"encoding/json"
	"testing"
)

func TestStandardizedResourceJSONSerialization(t *testing.T) {
	resource := StandardizedResource{
		Provider: "gcp",
		Service:  "cloud-run",
		Region:   "us-central1",
		ID:       "test-service",
		Name:     "Test Service",
		Attributes: map[string]string{
			"project_id": "my-project",
			"url":        "https://test-service.run.app",
			"status":     "ready",
		},
	}

	// Test marshaling
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource: %v", err)
	}

	// Test unmarshaling
	var unmarshaledResource StandardizedResource
	err = json.Unmarshal(data, &unmarshaledResource)
	if err != nil {
		t.Fatalf("Failed to unmarshal resource: %v", err)
	}

	// Verify fields
	if unmarshaledResource.Provider != resource.Provider {
		t.Errorf("Provider mismatch: expected %s, got %s", resource.Provider, unmarshaledResource.Provider)
	}
	if unmarshaledResource.Service != resource.Service {
		t.Errorf("Service mismatch: expected %s, got %s", resource.Service, unmarshaledResource.Service)
	}
	if unmarshaledResource.Region != resource.Region {
		t.Errorf("Region mismatch: expected %s, got %s", resource.Region, unmarshaledResource.Region)
	}
	if unmarshaledResource.ID != resource.ID {
		t.Errorf("ID mismatch: expected %s, got %s", resource.ID, unmarshaledResource.ID)
	}
	if unmarshaledResource.Name != resource.Name {
		t.Errorf("Name mismatch: expected %s, got %s", resource.Name, unmarshaledResource.Name)
	}

	// Verify attributes
	if len(unmarshaledResource.Attributes) != len(resource.Attributes) {
		t.Errorf("Attributes count mismatch: expected %d, got %d", len(resource.Attributes), len(unmarshaledResource.Attributes))
	}

	for key, expectedValue := range resource.Attributes {
		if actualValue, exists := unmarshaledResource.Attributes[key]; !exists {
			t.Errorf("Missing attribute key: %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("Attribute value mismatch for %s: expected %s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestStandardizedResourceEmptyAttributes(t *testing.T) {
	resource := StandardizedResource{
		Provider:   "aws",
		Service:    "ec2",
		Region:     "us-east-1",
		ID:         "i-1234567890",
		Name:       "Test Instance",
		Attributes: map[string]string{},
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource with empty attributes: %v", err)
	}

	var unmarshaledResource StandardizedResource
	err = json.Unmarshal(data, &unmarshaledResource)
	if err != nil {
		t.Fatalf("Failed to unmarshal resource with empty attributes: %v", err)
	}

	if unmarshaledResource.Attributes == nil {
		t.Error("Attributes should not be nil after unmarshaling")
	}
}

func TestStandardizedResourceNilAttributes(t *testing.T) {
	resource := StandardizedResource{
		Provider:   "aws",
		Service:    "iam",
		Region:     "global",
		ID:         "role-123",
		Name:       "Test Role",
		Attributes: nil,
	}

	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Failed to marshal resource with nil attributes: %v", err)
	}

	var unmarshaledResource StandardizedResource
	err = json.Unmarshal(data, &unmarshaledResource)
	if err != nil {
		t.Fatalf("Failed to unmarshal resource with nil attributes: %v", err)
	}

	// Attributes should be nil or empty after unmarshaling
	if unmarshaledResource.Attributes == nil {
		// This is acceptable
	} else if len(unmarshaledResource.Attributes) != 0 {
		t.Error("Attributes should be empty when originally nil")
	}
}

func TestMultipleResourcesSerialization(t *testing.T) {
	resources := []StandardizedResource{
		{
			Provider: "gcp",
			Service:  "project",
			Region:   "global",
			ID:       "project-1",
			Name:     "Project 1",
			Attributes: map[string]string{
				"state": "ACTIVE",
			},
		},
		{
			Provider: "aws",
			Service:  "ec2",
			Region:   "us-west-2",
			ID:       "i-987654321",
			Name:     "Instance 2",
			Attributes: map[string]string{
				"state": "running",
				"type":  "t2.micro",
			},
		},
		{
			Provider: "gcp",
			Service:  "vpc",
			Region:   "global",
			ID:       "vpc-1",
			Name:     "VPC 1",
			Attributes: map[string]string{
				"project_id": "project-1",
			},
		},
	}

	data, err := json.Marshal(resources)
	if err != nil {
		t.Fatalf("Failed to marshal resources: %v", err)
	}

	var unmarshaledResources []StandardizedResource
	err = json.Unmarshal(data, &unmarshaledResources)
	if err != nil {
		t.Fatalf("Failed to unmarshal resources: %v", err)
	}

	if len(unmarshaledResources) != len(resources) {
		t.Fatalf("Expected %d resources, got %d", len(resources), len(unmarshaledResources))
	}

	// Verify each resource
	for i, expected := range resources {
		actual := unmarshaledResources[i]
		if actual.Provider != expected.Provider {
			t.Errorf("Resource %d: Provider mismatch", i)
		}
		if actual.ID != expected.ID {
			t.Errorf("Resource %d: ID mismatch", i)
		}
	}
}

func TestLoadBalancerFlowSerialization(t *testing.T) {
	lbFlow := LoadBalancerFlow{
		Name:      "test-lb",
		ProjectID: "my-project",
		Frontend: FrontendConfig{
			IPAddress:    "34.120.45.67",
			PortRange:    "443",
			Protocol:     "HTTPS",
			Certificates: []string{"cert-1", "cert-2"},
			SSLPolicy:    "modern-ssl",
		},
		RoutingRules: []RoutingRule{
			{
				Hosts:       []string{"example.com", "www.example.com"},
				PathMatcher: "default",
			},
		},
		Backend: BackendConfig{
			Name:        "backend-1",
			Type:        "Cloud Run",
			ServiceName: "my-service",
			Region:      "us-central1",
		},
		CloudArmor: CloudArmorPolicy{
			Name: "security-policy-1",
			Rules: []CloudArmorRule{
				{
					Priority:    1000,
					Action:      "allow",
					Description: "Allow all traffic",
					Match:       "*",
				},
			},
		},
	}

	// Test marshaling
	data, err := json.Marshal(lbFlow)
	if err != nil {
		t.Fatalf("Failed to marshal LoadBalancerFlow: %v", err)
	}

	// Test unmarshaling
	var unmarshaledLB LoadBalancerFlow
	err = json.Unmarshal(data, &unmarshaledLB)
	if err != nil {
		t.Fatalf("Failed to unmarshal LoadBalancerFlow: %v", err)
	}

	// Verify fields
	if unmarshaledLB.Name != lbFlow.Name {
		t.Errorf("Name mismatch: expected %s, got %s", lbFlow.Name, unmarshaledLB.Name)
	}
	if unmarshaledLB.ProjectID != lbFlow.ProjectID {
		t.Errorf("ProjectID mismatch: expected %s, got %s", lbFlow.ProjectID, unmarshaledLB.ProjectID)
	}
	if unmarshaledLB.Frontend.IPAddress != lbFlow.Frontend.IPAddress {
		t.Errorf("Frontend IP mismatch: expected %s, got %s", lbFlow.Frontend.IPAddress, unmarshaledLB.Frontend.IPAddress)
	}
	if len(unmarshaledLB.RoutingRules) != len(lbFlow.RoutingRules) {
		t.Errorf("RoutingRules count mismatch: expected %d, got %d", len(lbFlow.RoutingRules), len(unmarshaledLB.RoutingRules))
	}
	if unmarshaledLB.Backend.Type != lbFlow.Backend.Type {
		t.Errorf("Backend type mismatch: expected %s, got %s", lbFlow.Backend.Type, unmarshaledLB.Backend.Type)
	}
	if unmarshaledLB.CloudArmor.Name != lbFlow.CloudArmor.Name {
		t.Errorf("CloudArmor name mismatch: expected %s, got %s", lbFlow.CloudArmor.Name, unmarshaledLB.CloudArmor.Name)
	}
}

func TestStandardizedResourceProviderValidation(t *testing.T) {
	testCases := []struct {
		name     string
		provider string
		valid    bool
	}{
		{"GCP provider", "gcp", true},
		{"AWS provider", "aws", true},
		{"Azure provider", "azure", true},
		{"Empty provider", "", true}, // Empty is technically valid JSON
		{"Unknown provider", "unknown", true}, // We don't enforce validation, just test serialization
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource := StandardizedResource{
				Provider: tc.provider,
				Service:  "test-service",
				Region:   "us-central1",
				ID:       "test-id",
				Name:     "Test Name",
			}

			data, err := json.Marshal(resource)
			if err != nil && tc.valid {
				t.Errorf("Unexpected marshal error for valid provider: %v", err)
			}

			if err == nil {
				var unmarshaled StandardizedResource
				err = json.Unmarshal(data, &unmarshaled)
				if err != nil {
					t.Errorf("Unexpected unmarshal error: %v", err)
				}
				if unmarshaled.Provider != tc.provider {
					t.Errorf("Provider mismatch: expected %s, got %s", tc.provider, unmarshaled.Provider)
				}
			}
		})
	}
}
