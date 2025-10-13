// fetcher/gcp_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings"

	asset "cloud.google.com/go/asset/apiv1"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/run/v1"
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

// FetchGCPResourcesFromOrg uses the Cloud Asset API to fetch all folders, projects, and their resources.
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
		var standardizedRes StandardizedResource
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
				networkRes, err := FetchGCPNetworkResourcesForProject(projectID)
				if err != nil { log.Printf("Warning: could not fetch network resources for project %s: %v", projectID, err) }
				allResources = append(allResources, networkRes...)

				cloudRunRes, err := FetchGCPCloudRunServices(projectID)
				if err != nil { log.Printf("Warning: could not fetch cloud run services for project %s: %v", projectID, err) }
				allResources = append(allResources, cloudRunRes...)
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

// FetchGCPProjectsNoOrg uses the Resource Manager API to list all projects and their resources.
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
			// This is the block that was missing from the abbreviated code
			standardizedRes := StandardizedResource{
				Provider: "gcp", Service: "project", Region: "global", ID: project.ProjectId, Name: project.Name,
				Attributes: map[string]string{
					"state":          project.LifecycleState,
					"project_number": fmt.Sprintf("%d", project.ProjectNumber),
				},
			}
			allResources = append(allResources, standardizedRes)

			networkRes, err := FetchGCPNetworkResourcesForProject(project.ProjectId)
			if err != nil {
				log.Printf("Warning: could not fetch network resources for project %s: %v", project.ProjectId, err)
				continue
			}
			allResources = append(allResources, networkRes...)

			cloudRunRes, err := FetchGCPCloudRunServices(project.ProjectId)
			if err != nil {
				log.Printf("Warning: could not fetch cloud run services for project %s: %v", project.ProjectId, err)
				continue
			}
			allResources = append(allResources, cloudRunRes...)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return allResources, nil
}

// FetchGCPNetworkResourcesForProject scans a single project for its networking components.
func FetchGCPNetworkResourcesForProject(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var networkResources []StandardizedResource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service for project %s: %w", projectID, err)
	}
	log.Printf("   -> Fetching network resources for project: %s", projectID)

	networks, err := computeService.Networks.List(projectID).Do()
	if err != nil {
		log.Printf("Warning: could not list networks for project %s: %v", projectID, err)
	} else {
		for _, network := range networks.Items {
			networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "vpc", Region: "global", ID: network.Name, Name: network.Name, Attributes: map[string]string{"project_id": projectID, "mode": fmt.Sprintf("%t", network.AutoCreateSubnetworks)}})
		}
	}
	subnets, err := computeService.Subnetworks.AggregatedList(projectID).Do()
	if err != nil {
		log.Printf("Warning: could not list subnets for project %s: %v", projectID, err)
	} else {
		for _, scope := range subnets.Items {
			for _, subnet := range scope.Subnetworks {
				networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "subnet", Region: subnet.Region, ID: subnet.Name, Name: subnet.Name, Attributes: map[string]string{"project_id": projectID, "vpc": subnet.Network, "cidr_range": subnet.IpCidrRange}})
			}
		}
	}
	firewallList, err := computeService.Firewalls.List(projectID).Do()
	if err != nil {
		log.Printf("Warning: could not list firewall rules for project %s: %v", projectID, err)
	} else {
		for _, listRule := range firewallList.Items {
			rule, err := computeService.Firewalls.Get(projectID, listRule.Name).Do()
			if err != nil { log.Printf("Warning: could not get full details for firewall rule %s: %v", listRule.Name, err); continue }
			formatAllowedRules := func(details []*compute.FirewallAllowed) string { var parts []string; for _, d := range details { part := d.IPProtocol; if len(d.Ports) > 0 { part += ":" + strings.Join(d.Ports, ",") }; parts = append(parts, part) }; return strings.Join(parts, "; ") }
			formatDeniedRules := func(details []*compute.FirewallDenied) string { var parts []string; for _, d := range details { part := d.IPProtocol; if len(d.Ports) > 0 { part += ":" + strings.Join(d.Ports, ",") }; parts = append(parts, part) }; return strings.Join(parts, "; ") }
			action := "DENY"; if len(rule.Allowed) > 0 { action = "ALLOW" }
			attributes := map[string]string{"project_id": projectID, "action": action, "direction": rule.Direction, "priority": fmt.Sprintf("%d", rule.Priority), "disabled": fmt.Sprintf("%t", rule.Disabled), "source_ranges": strings.Join(rule.SourceRanges, ", "), "destination_ranges": strings.Join(rule.DestinationRanges, ", "), "target_tags": strings.Join(rule.TargetTags, ", "), "allowed": formatAllowedRules(rule.Allowed), "denied": formatDeniedRules(rule.Denied)}
			networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "firewall", Region: "global", ID: rule.Name, Name: rule.Name, Attributes: attributes})
		}
	}
	return networkResources, nil
}
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
				"vpc":        "N/A", // Default values
				"subnet":     "N/A",
			}

			// CORRECTED: Check for VPC Access configuration in the v1 API's annotations.
			if service.Spec.Template.Metadata != nil && service.Spec.Template.Metadata.Annotations != nil {
				annotations := service.Spec.Template.Metadata.Annotations
				if connectorName, ok := annotations["run.googleapis.com/vpc-access-connector"]; ok {
					attributes["vpc"] = "via-connector"
					attributes["subnet"] = connectorName // The connector name is the "subnet" in this context
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