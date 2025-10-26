package fetcher

import "testing"

func TestExtractResourceName(t *testing.T) {
	tests := []struct {
		name         string
		resourcePath string
		expected     string
	}{
		{
			name:         "Full VPC path",
			resourcePath: "projects/my-project/global/networks/my-vpc",
			expected:     "my-vpc",
		},
		{
			name:         "Full subnet path",
			resourcePath: "projects/my-project/regions/us-central1/subnetworks/my-subnet",
			expected:     "my-subnet",
		},
		{
			name:         "VPC connector path",
			resourcePath: "projects/my-project/locations/us-central1/connectors/my-connector",
			expected:     "my-connector",
		},
		{
			name:         "Single name without path",
			resourcePath: "my-resource",
			expected:     "my-resource",
		},
		{
			name:         "Empty string",
			resourcePath: "",
			expected:     "N/A",
		},
		{
			name:         "Path with trailing slash",
			resourcePath: "projects/my-project/networks/my-vpc/",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResourceName(tt.resourcePath)
			if result != tt.expected {
				t.Errorf("extractResourceName(%q) = %q, expected %q", tt.resourcePath, result, tt.expected)
			}
		})
	}
}

func TestParseNetworkInterfaces(t *testing.T) {
	tests := []struct {
		name               string
		networkInterfacesJSON string
		projectID          string
		expectedVPC        string
		expectedSubnet     string
	}{
		{
			name:               "Valid network interfaces with full paths",
			networkInterfacesJSON: `[{"network":"projects/my-project/global/networks/my-vpc","subnetwork":"projects/my-project/regions/us-central1/subnetworks/my-subnet"}]`,
			projectID:          "my-project",
			expectedVPC:        "my-vpc",
			expectedSubnet:     "my-subnet",
		},
		{
			name:               "Valid network interfaces with simple names",
			networkInterfacesJSON: `[{"network":"my-vpc","subnetwork":"my-subnet"}]`,
			projectID:          "my-project",
			expectedVPC:        "my-vpc",
			expectedSubnet:     "my-subnet",
		},
		{
			name:               "Network only, no subnetwork",
			networkInterfacesJSON: `[{"network":"my-vpc"}]`,
			projectID:          "my-project",
			expectedVPC:        "my-vpc",
			expectedSubnet:     "",
		},
		{
			name:               "Empty array",
			networkInterfacesJSON: `[]`,
			projectID:          "my-project",
			expectedVPC:        "",
			expectedSubnet:     "",
		},
		{
			name:               "Invalid JSON",
			networkInterfacesJSON: `{invalid json}`,
			projectID:          "my-project",
			expectedVPC:        "",
			expectedSubnet:     "",
		},
		{
			name:               "Multiple interfaces - takes first",
			networkInterfacesJSON: `[{"network":"vpc-1","subnetwork":"subnet-1"},{"network":"vpc-2","subnetwork":"subnet-2"}]`,
			projectID:          "my-project",
			expectedVPC:        "vpc-1",
			expectedSubnet:     "subnet-1",
		},
		{
			name:               "Empty network and subnetwork values",
			networkInterfacesJSON: `[{"network":"","subnetwork":""}]`,
			projectID:          "my-project",
			expectedVPC:        "",
			expectedSubnet:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpcName, subnetName := parseNetworkInterfaces(tt.networkInterfacesJSON, tt.projectID)
			if vpcName != tt.expectedVPC {
				t.Errorf("parseNetworkInterfaces() vpc = %q, expected %q", vpcName, tt.expectedVPC)
			}
			if subnetName != tt.expectedSubnet {
				t.Errorf("parseNetworkInterfaces() subnet = %q, expected %q", subnetName, tt.expectedSubnet)
			}
		})
	}
}
