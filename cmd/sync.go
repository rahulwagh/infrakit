// cmd/sync.go
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/rahulwagh/infrakit/cache"   // CHANGE THIS
	"github.com/rahulwagh/infrakit/fetcher" // CHANGE THIS
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Fetch resources from cloud providers and update the local cache.",
	// cmd/sync.go

    Run: func(cmd *cobra.Command, args []string) {
        log.Println("Starting resource sync...")

        var allResources []fetcher.StandardizedResource

        // Fetch EC2 Instances
        ec2Resources, err := fetcher.FetchEC2Instances()
        if err != nil {
            log.Fatalf("Error fetching EC2 instances: %v", err)
        }
        allResources = append(allResources, ec2Resources...)

        // Fetch IAM Roles
        iamResources, err := fetcher.FetchIAMRoles()
        if err != nil {
            log.Fatalf("Error fetching IAM roles: %v", err)
        }
        allResources = append(allResources, iamResources...)

        // Save the combined results to the cache
        if err := cache.SaveResources(allResources); err != nil {
            log.Fatalf("Error saving cache: %v", err)
        }

        log.Printf("Sync completed successfully! Found %d total resources.\n", len(allResources))
    },
}

func init() {
	rootCmd.AddCommand(syncCmd)
}