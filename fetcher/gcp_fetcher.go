// fetcher/gcp_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"

	asset "cloud.google.com/go/asset/apiv1"
	//resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/iterator"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
	//resourcemanagerpb "google.golang.org/genproto/googleapis/cloud/resourcemanager/v3"
	"google.golang.org/api/cloudresourcemanager/v1"
)

// --- Fetcher for Organization Hierarchy ---

// FetchGCPResourcesFromOrg uses the Cloud Asset API to fetch all folders and projects in an organization.
func FetchGCPResourcesFromOrg(organizationID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var allResources []StandardizedResource

	client, err := asset.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset client: %w", err)
	}
	defer client.Close()

	log.Println("Fetching all GCP resources for organization", organizationID)
	req := &assetpb.SearchAllResourcesRequest{
		Scope:      "organizations/" + organizationID,
		AssetTypes: []string{"cloudresourcemanager.googleapis.com/Project", "cloudresourcemanager.googleapis.com/Folder"},
	}
	it := client.SearchAllResources(ctx, req)
	for {
		resource, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed during asset iteration: %w", err)
		}
		var standardizedRes StandardizedResource
		switch resource.AssetType {
		case "cloudresourcemanager.googleapis.com/Project":
			projectId, projectNumber := "N/A", "N/A"
			if attrs := resource.GetAdditionalAttributes(); attrs != nil {
				if idVal := attrs.GetFields()["projectId"]; idVal != nil {
					projectId = idVal.GetStringValue()
				}
				if numVal := attrs.GetFields()["projectNumber"]; numVal != nil {
					projectNumber = numVal.GetStringValue()
				}
			}
			standardizedRes = StandardizedResource{
				Provider: "gcp", Service: "project", Region: "global", ID: projectId, Name: resource.GetDisplayName(),
				Attributes: map[string]string{"state": resource.GetState(), "project_number": projectNumber},
			}
		case "cloudresourcemanager.googleapis.com/Folder":
			standardizedRes = StandardizedResource{
				Provider: "gcp", Service: "folder", Region: "global", ID: resource.GetName(), Name: resource.GetDisplayName(),
				Attributes: map[string]string{"state": resource.GetState()},
			}
		}
		allResources = append(allResources, standardizedRes)
	}
	log.Printf("Successfully fetched %d GCP resources from organization.\n", len(allResources))
	return allResources, nil
}

// --- Fetcher for Projects without an Organization ---

// FetchGCPProjectsNoOrg uses the Resource Manager API to list all projects accessible by the user.
func FetchGCPProjectsNoOrg() ([]StandardizedResource, error) {
	ctx := context.Background()
	var allResources []StandardizedResource

	// Create a new v1 cloudresourcemanager service client. ADC is handled automatically.
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudresourcemanager service: %w", err)
	}

	log.Println("No GCP Organization ID provided. Fetching all accessible projects using v1 API...")

	// Call the Projects.List method.
	call := crmService.Projects.List()

	// The Do() method executes the request. We wrap it in a function to handle pagination.
	err = call.Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range page.Projects {
			// The v1 'project' struct has direct fields for the data we need.
			standardizedRes := StandardizedResource{
				Provider: "gcp",
				Service:  "project",
				Region:   "global",
				ID:       project.ProjectId,
				Name:     project.Name, // In v1, 'Name' is the display name.
				Attributes: map[string]string{
					"state":          project.LifecycleState,
					"project_number": fmt.Sprintf("%d", project.ProjectNumber),
				},
			}
			allResources = append(allResources, standardizedRes)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	log.Printf("Successfully fetched %d GCP projects.\n", len(allResources))
	return allResources, nil
}