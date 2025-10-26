// cmd/sync.go
package cmd

import (
	"log"

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [provider] [project-id]",
	Short: "Fetch resources from cloud providers and update the local cache.",
	Long: `Sync resources from cloud providers. Examples:
  infrakit sync              - Sync all providers (AWS, GCP)
  infrakit sync aws          - Sync only AWS resources
  infrakit sync gcp          - Sync all GCP projects
  infrakit sync gcp my-proj  - Sync only the specified GCP project`,

    Run: func(cmd *cobra.Command, args []string) {
    	log.Println("Starting resource sync...")

    	// Parse arguments
    	providerToSync := ""
    	projectID := ""
    	if len(args) > 0 {
    		providerToSync = args[0]
    	}
    	if len(args) > 1 {
    		projectID = args[1]
    	}

    	// --- Handle GCP project-specific sync ---
    	if providerToSync == "gcp" && projectID != "" {
    		log.Printf("--- Syncing specific GCP project: %s ---", projectID)

    		// Fetch resources for the specific project
    		gcpResources, err := fetcher.FetchGCPSingleProject(projectID)
    		if err != nil {
    			log.Fatalf("Error fetching resources for project %s: %v", projectID, err)
    		}

    		log.Printf("Found %d resources for project %s", len(gcpResources), projectID)

    		// Merge with existing cache (intelligent merge)
    		if err := cache.MergeResourcesForProject(gcpResources, projectID); err != nil {
    			log.Fatalf("Error merging cache for project %s: %v", projectID, err)
    		}

    		log.Printf("Successfully synced project %s and merged with cache!\n", projectID)
    		return
    	}

    	// --- Handle full provider sync (existing behavior) ---
    	var allResources []fetcher.StandardizedResource

    	// --- Sync AWS Resources ---
    	// This block runs if no provider is specified (sync all) OR if the provider is "aws".
    	if providerToSync == "" || providerToSync == "aws" {
    		log.Println("--- Syncing AWS Resources ---")
    		var awsResources []fetcher.StandardizedResource

    		ec2Resources, err := fetcher.FetchEC2Instances()
    		if err != nil {
    			log.Fatalf("Error fetching EC2 instances: %v", err)
    		}
    		awsResources = append(awsResources, ec2Resources...)

    		iamResources, err := fetcher.FetchIAMRoles()
    		if err != nil {
    			log.Fatalf("Error fetching IAM roles: %v", err)
    		}
    		awsResources = append(awsResources, iamResources...)

    		allResources = append(allResources, awsResources...)
    		log.Printf("Found %d AWS resources.", len(awsResources))
    	}

    	// --- Sync GCP Resources ---
    	// This block runs if no provider is specified (sync all) OR if the provider is "gcp".
    	if providerToSync == "" || providerToSync == "gcp" {
    		log.Println("--- Syncing GCP Resources ---")

    		gcpOrganizationID, err := fetcher.DiscoverGCPOrganization()
    		if err != nil {
    			log.Printf("Warning: Could not discover GCP organization: %v", err)
    		}

    		var gcpResources []fetcher.StandardizedResource
    		if gcpOrganizationID != "" {
    			gcpResources, err = fetcher.FetchGCPResourcesFromOrg(gcpOrganizationID)
    		} else {
    			gcpResources, err = fetcher.FetchGCPProjectsNoOrg()
    		}

    		if err != nil {
    			log.Fatalf("Error fetching GCP resources: %v", err)
    		}

    		allResources = append(allResources, gcpResources...)
    		log.Printf("Found %d GCP resources.", len(gcpResources))
    	}

    	// --- Input Validation ---
    	// If a provider was specified but it wasn't "aws" or "gcp", it's invalid.
    	if providerToSync != "" && providerToSync != "aws" && providerToSync != "gcp" {
    		log.Fatalf("Error: Invalid provider '%s'. Valid providers are 'aws' or 'gcp', or no provider to sync all.", providerToSync)
    	}

    	// --- Save combined results (full replacement for full provider sync) ---
    	if len(allResources) > 0 {
    		if err := cache.SaveResources(allResources); err != nil {
    			log.Fatalf("Error saving cache: %v", err)
    		}
    		log.Printf("Sync completed successfully! Found %d total resources.\n", len(allResources))
    	} else {
    		log.Println("Sync finished. No new resources found.")
    	}
    },
}

func init() {
	rootCmd.AddCommand(syncCmd)
}