// fetcher/gcp_discovery.go
package fetcher

import (
	"context"
	"fmt"
	"log"

	asset "cloud.google.com/go/asset/apiv1"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iterator"
	assetpb "google.golang.org/genproto/googleapis/cloud/asset/v1"
	resourcemanagerpb "google.golang.org/genproto/googleapis/cloud/resourcemanager/v3"
)

// DiscoverGCPOrganization searches for an organization the user can access.
func DiscoverGCPOrganization() (string, error) {
	ctx := context.Background()
	orgClient, err := resourcemanager.NewOrganizationsClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create organizations client: %w", err)
	}
	defer orgClient.Close()

	log.Println("Checking for a GCP Organization...")
	req := &resourcemanagerpb.SearchOrganizationsRequest{Query: ""}
	it := orgClient.SearchOrganizations(ctx, req)
	firstOrg, err := it.Next()
	if err == iterator.Done {
		log.Println("No GCP Organization found.")
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed during organization search: %w", err)
	}
	log.Println("Found GCP Organization:", firstOrg.DisplayName)
	return firstOrg.Name, nil
}

// FetchGCPResourcesFromOrg uses the Cloud Asset API to fetch all folders, projects, and their sub-resources.
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
		Scope:      organizationID,
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
		var standardizedRes StandardizedResource // Declare standardizedRes here
		var projectID string
		switch resource.AssetType {
		case "cloudresourcemanager.googleapis.com/Project":
			projectNumber := "N/A"
			if attrs := resource.GetAdditionalAttributes(); attrs != nil {
				if idVal := attrs.GetFields()["projectId"]; idVal != nil {
					projectID = idVal.GetStringValue()
				}
				if numVal := attrs.GetFields()["projectNumber"]; numVal != nil {
					projectNumber = numVal.GetStringValue()
				}
			}
			standardizedRes = StandardizedResource{
				Provider: "gcp", Service: "project", Region: "global", ID: projectID, Name: resource.GetDisplayName(),
				Attributes: map[string]string{"state": resource.GetState(), "project_number": projectNumber},
			}
			allResources = append(allResources, standardizedRes)

			if projectID != "" && projectID != "N/A" {
				networkRes, _ := FetchGCPNetworkResourcesForProject(projectID)
				allResources = append(allResources, networkRes...)
				cloudRunRes, _ := FetchGCPCloudRunServices(projectID)
				allResources = append(allResources, cloudRunRes...)
				appInfraRes, _ := FetchGCPAppInfraForProject(projectID)
				allResources = append(allResources, appInfraRes...)

				// CORRECTED: Added the missing call
				iamRes, err := FetchGCPServiceAccounts(projectID)
				if err != nil {
					log.Printf("Warning: could not fetch service accounts for project %s: %v", projectID, err)
					// Continue even if fetching SAs fails for one project
				}
				allResources = append(allResources, iamRes...)
			}
		case "cloudresourcemanager.googleapis.com/Folder":
			standardizedRes = StandardizedResource{
				Provider: "gcp", Service: "folder", Region: "global", ID: resource.GetName(), Name: resource.GetDisplayName(),
				Attributes: map[string]string{"state": resource.GetState()},
			}
			allResources = append(allResources, standardizedRes)
		}
	}
	return allResources, nil
}

// FetchGCPProjectsNoOrg uses the Resource Manager API to list all projects and their sub-resources.
func FetchGCPProjectsNoOrg() ([]StandardizedResource, error) {
	ctx := context.Background()
	var allResources []StandardizedResource
	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudresourcemanager service: %w", err)
	}
	log.Println("No GCP Organization ID provided. Fetching all accessible projects using v1 API...")
	call := crmService.Projects.List()
	err = call.Pages(ctx, func(page *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range page.Projects {
			standardizedRes := StandardizedResource{
				Provider: "gcp", Service: "project", Region: "global", ID: project.ProjectId, Name: project.Name,
				Attributes: map[string]string{
					"state":          project.LifecycleState,
					"project_number": fmt.Sprintf("%d", project.ProjectNumber),
				},
			}
			allResources = append(allResources, standardizedRes)

			networkRes, _ := FetchGCPNetworkResourcesForProject(project.ProjectId)
			allResources = append(allResources, networkRes...)
			cloudRunRes, _ := FetchGCPCloudRunServices(project.ProjectId)
			allResources = append(allResources, cloudRunRes...)
			appInfraRes, _ := FetchGCPAppInfraForProject(project.ProjectId)
			allResources = append(allResources, appInfraRes...)

			// CORRECTED: Added the missing call
			iamRes, err := FetchGCPServiceAccounts(project.ProjectId)
			if err != nil {
				log.Printf("Warning: could not fetch service accounts for project %s: %v", project.ProjectId, err)
				// Continue to next project even if SA fetching fails for one
				continue
			}
			allResources = append(allResources, iamRes...)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return allResources, nil
}