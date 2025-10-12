// cmd/sync.go
package cmd

import (
	"log"

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch resources from cloud providers and update the local cache.",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Starting resource sync...")

		var allResources []fetcher.StandardizedResource

		// --- AWS ---
		ec2Resources, err := fetcher.FetchEC2Instances()
		if err != nil {
			log.Fatalf("Error fetching EC2 instances: %v", err)
		}
		allResources = append(allResources, ec2Resources...)

		iamResources, err := fetcher.FetchIAMRoles()
		if err != nil {
			log.Fatalf("Error fetching IAM roles: %v", err)
		}
		allResources = append(allResources, iamResources...)

		// --- GCP ---
		// Automatically discover the organization ID. No more hardcoding!
		gcpOrganizationID, err := fetcher.DiscoverGCPOrganization()
		if err != nil {
			// Log a warning but continue, as we can fall back to listing projects.
			log.Printf("Warning: Could not discover GCP organization: %v", err)
		}

		var gcpResources []fetcher.StandardizedResource
		if gcpOrganizationID != "" {
			// If an Org was discovered, scan the full hierarchy.
			gcpResources, err = fetcher.FetchGCPResourcesFromOrg(gcpOrganizationID)
		} else {
			// If no Org was found, just list all accessible projects.
			gcpResources, err = fetcher.FetchGCPProjectsNoOrg()
		}

		if err != nil {
			log.Fatalf("Error fetching GCP resources: %v", err)
		}
		allResources = append(allResources, gcpResources...)

		// --- Save combined results ---
		if err := cache.SaveResources(allResources); err != nil {
			log.Fatalf("Error saving cache: %v", err)
		}

		log.Printf("Sync completed successfully! Found %d total resources.\n", len(allResources))
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}