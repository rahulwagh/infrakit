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
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Starting resource sync...")

		// Call our fetcher
		resources, err := fetcher.FetchEC2Instances()
		if err != nil {
			log.Fatalf("Error fetching resources: %v", err)
		}

		// Save the results to the cache
		if err := cache.SaveResources(resources); err != nil {
			log.Fatalf("Error saving cache: %v", err)
		}

		log.Println("Sync completed successfully!")
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}