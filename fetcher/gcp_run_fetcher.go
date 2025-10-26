// fetcher/gcp_run_fetcher.go
package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/run/v1"
)

// FetchGCPCloudRunServices fetches all Cloud Run services for a given project using the v1 API.
func FetchGCPCloudRunServices(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var cloudRunResources []StandardizedResource
	runService, err := run.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create run service for project %s: %w", projectID, err)
	}

	log.Printf("   -> Fetching Cloud Run services for project: %s", projectID)
	parent := fmt.Sprintf("projects/%s/locations/-", projectID)
	resp, err := runService.Projects.Locations.Services.List(parent).Do()
	if err != nil {
		log.Printf("Warning: could not list Cloud Run services for project %s: %v", projectID, err)
		return nil, nil
	}

	for _, service := range resp.Items {
		if service.Spec != nil && service.Spec.Template != nil && service.Spec.Template.Spec != nil && len(service.Spec.Template.Spec.Containers) > 0 {
			attributes := map[string]string{
				"project_id": projectID,
				"url":        service.Status.Url,
				"image":      service.Spec.Template.Spec.Containers[0].Image,
				"vpc":        "N/A",
				"subnet":     "N/A",
			}

			// Extract network configuration from annotations
			if service.Spec.Template.Metadata != nil && service.Spec.Template.Metadata.Annotations != nil {
				annotations := service.Spec.Template.Metadata.Annotations

				// Check for VPC Access Connector (older/simpler method)
				if connectorName, ok := annotations["run.googleapis.com/vpc-access-connector"]; ok {
					attributes["vpc"] = "via-connector"
					attributes["subnet"] = extractResourceName(connectorName)
				}

				// Check for network interfaces (direct VPC egress - newer method)
				if networkInterfaces, ok := annotations["run.googleapis.com/network-interfaces"]; ok {
					vpcName, subnetName := parseNetworkInterfaces(networkInterfaces, projectID)
					if vpcName != "" {
						attributes["vpc"] = vpcName
					}
					if subnetName != "" {
						attributes["subnet"] = subnetName
					}
				}
			}

			cloudRunResources = append(cloudRunResources, StandardizedResource{
				Provider:   "gcp",
				Service:    "cloudrun",
				Region:     service.Metadata.Labels["cloud.googleapis.com/location"],
				ID:         service.Metadata.Name,
				Name:       service.Metadata.Name,
				Attributes: attributes,
			})
		}
	}
	return cloudRunResources, nil
}

// extractResourceName extracts the resource name from a full GCP resource path
// Example: "projects/my-project/locations/us-central1/connectors/my-connector" -> "my-connector"
func extractResourceName(resourcePath string) string {
	if resourcePath == "" {
		return "N/A"
	}
	parts := strings.Split(resourcePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return resourcePath
}

// parseNetworkInterfaces parses the network-interfaces annotation JSON
// Format: [{"network":"vpc-name","subnetwork":"subnet-name"}]
func parseNetworkInterfaces(networkInterfacesJSON string, projectID string) (vpcName, subnetName string) {
	var interfaces []map[string]interface{}
	if err := json.Unmarshal([]byte(networkInterfacesJSON), &interfaces); err != nil {
		log.Printf("Warning: could not parse network-interfaces annotation: %v", err)
		return "", ""
	}

	if len(interfaces) == 0 {
		return "", ""
	}

	// Take the first interface
	iface := interfaces[0]

	// Extract VPC name
	if network, ok := iface["network"].(string); ok && network != "" {
		vpcName = extractResourceName(network)
	}

	// Extract subnet name
	if subnetwork, ok := iface["subnetwork"].(string); ok && subnetwork != "" {
		subnetName = extractResourceName(subnetwork)
	}

	return vpcName, subnetName
}