// fetcher/gcp_iam_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings"

	crm "google.golang.org/api/cloudresourcemanager/v1" // Use v1 for project policy
	"google.golang.org/api/iam/v1"
)

// FetchGCPServiceAccounts fetches all service accounts and their PROJECT-LEVEL assigned roles.
func FetchGCPServiceAccounts(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var iamResources []StandardizedResource

	// IAM client (for listing SAs)
	iamService, err := iam.NewService(ctx)
	if err != nil {
		log.Printf("Error creating IAM service for project %s: %v", projectID, err)
		return nil, fmt.Errorf("failed to create iam service for project %s: %w", projectID, err)
	}

	// Cloud Resource Manager client (for getting project policy)
	crmService, err := crm.NewService(ctx)
	if err != nil {
		log.Printf("Error creating CRM service for project %s: %v", projectID, err)
		return nil, fmt.Errorf("failed to create cloudresourcemanager service for project %s: %w", projectID, err)
	}

	log.Printf("   -> Fetching Service Accounts and Project Roles for project: %s", projectID)

	// --- Step 1: Get the Project's IAM Policy ---
	projectPolicy, err := crmService.Projects.GetIamPolicy(projectID, &crm.GetIamPolicyRequest{}).Do()
	if err != nil {
		// If we can't get the project policy, we can't determine roles. Log and return empty.
		log.Printf("Warning: could not get project IAM policy for project %s (permissions issue?): %v", projectID, err)
		return []StandardizedResource{}, nil
	}

	// --- Step 2: List Service Accounts ---
	parent := fmt.Sprintf("projects/%s", projectID)
	resp, err := iamService.Projects.ServiceAccounts.List(parent).Do()
	if err != nil {
		log.Printf("Warning: could not list service accounts for project %s: %v", projectID, err)
		return []StandardizedResource{}, nil
	}

	if resp == nil || len(resp.Accounts) == 0 {
		log.Printf("   -> No service accounts found for project %s", projectID)
		return []StandardizedResource{}, nil
	}

	// --- Step 3: For each SA, find its roles in the Project Policy ---
	for _, account := range resp.Accounts {
		if account == nil { continue }

		var assignedRoles []string
		memberIdentifier := "serviceAccount:" + account.Email // Format needed for policy matching

		// Iterate through the project policy bindings
		if projectPolicy != nil {
			for _, binding := range projectPolicy.Bindings {
				// Check if this binding's member list contains our SA
				for _, member := range binding.Members {
					if member == memberIdentifier {
						assignedRoles = append(assignedRoles, binding.Role)
						break // Found SA in this binding, no need to check other members of the same binding
					}
				}
			}
		}

		// --- (Rest of the code to create the StandardizedResource is the same) ---
		displayName := account.DisplayName
		if displayName == "" {
			emailParts := strings.Split(account.Email, "@")
			if len(emailParts) > 0 { displayName = emailParts[0] } else { displayName = "N/A" }
		}
		attributes := map[string]string{
			"project_id":    projectID,
			"email":         account.Email,
			"unique_id":     account.UniqueId,
			"disabled":      fmt.Sprintf("%t", account.Disabled),
			"description":   account.Description,
			"roles":         strings.Join(assignedRoles, ", "), // Add roles to attributes
		}
		iamResources = append(iamResources, StandardizedResource{
			Provider:   "gcp", Service:  "serviceaccount", Region:   "global",
			ID:         account.Email, Name:       displayName, Attributes: attributes,
		})
	}
	log.Printf("   -> Fetched %d service accounts for project %s", len(iamResources), projectID)
	return iamResources, nil
}