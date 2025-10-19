// fetcher/gcp_iam_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings" // MOVED: Import statement is now correctly placed here

	"google.golang.org/api/iam/v1"
)

// FetchGCPServiceAccounts fetches all service accounts for a given project.
func FetchGCPServiceAccounts(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var iamResources []StandardizedResource
	iamService, err := iam.NewService(ctx)
	if err != nil {
		// Log detailed error for service creation failure
		log.Printf("Error creating IAM service for project %s: %v", projectID, err)
		return nil, fmt.Errorf("failed to create iam service for project %s: %w", projectID, err)
	}

	log.Printf("   -> Fetching Service Accounts for project: %s", projectID)
	parent := fmt.Sprintf("projects/%s", projectID)

	resp, err := iamService.Projects.ServiceAccounts.List(parent).Do()
	if err != nil {
		// Log a warning but don't fail the whole sync if permissions are missing
		log.Printf("Warning: could not list service accounts for project %s: %v", projectID, err)
		// Return empty slice and no error to allow sync to continue for other resources/projects
		return []StandardizedResource{}, nil
	}

	if resp == nil || len(resp.Accounts) == 0 {
		log.Printf("   -> No service accounts found for project %s", projectID)
		return []StandardizedResource{}, nil // No accounts found is not an error
	}

	for _, account := range resp.Accounts {
		// Ensure account is not nil before accessing fields
		if account == nil {
			log.Printf("Warning: encountered nil service account in project %s", projectID)
			continue
		}

		displayName := account.DisplayName
		if displayName == "" {
			// Use the start of the email if display name is empty
			emailParts := strings.Split(account.Email, "@")
			if len(emailParts) > 0 {
				displayName = emailParts[0]
			} else {
				displayName = "N/A"
			}
		}

		iamResources = append(iamResources, StandardizedResource{
			Provider: "gcp",
			Service:  "serviceaccount",
			Region:   "global",
			ID:       account.Email, // Use email as the unique ID
			Name:     displayName,   // Use display name or derived name
			Attributes: map[string]string{
				"project_id":    projectID,
				"email":         account.Email,
				"unique_id":     account.UniqueId,
				"disabled":      fmt.Sprintf("%t", account.Disabled),
				"description":   account.Description,
			},
		})
	}
	log.Printf("   -> Fetched %d service accounts for project %s", len(iamResources), projectID)
	return iamResources, nil
}

// REMOVED: The misplaced import "strings" line is gone from here.