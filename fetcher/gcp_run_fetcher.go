// fetcher/gcp_run_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"

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

			if service.Spec.Template.Metadata != nil && service.Spec.Template.Metadata.Annotations != nil {
				annotations := service.Spec.Template.Metadata.Annotations
				if connectorName, ok := annotations["run.googleapis.com/vpc-access-connector"]; ok {
					attributes["vpc"] = "via-connector"
					attributes["subnet"] = connectorName
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