// cmd/search.go
package cmd

import (
	"fmt"
	"log"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"github.com/rahulwagh/infrakit/cache"
	//"github.com/rahulwagh/infrakit/fetcher"
)

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for a resource in the local cache with a fuzzy finder.",
	Run: func(cmd *cobra.Command, args []string) {
		// Load resources from the cache
		resources, err := cache.LoadResources()
		if err != nil {
			log.Fatalf("Error loading cache: %v", err)
		}

		// Run the fuzzy finder
		idx, err := fuzzyfinder.Find(
			resources,
			func(i int) string {
				// This is the string that the finder will search against
				return fmt.Sprintf("%s :: %s", resources[i].Name, resources[i].ID)
			},
			fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
				// This creates the nice preview window on the right
				if i == -1 {
					return ""
				}
				r := resources[i]
				return fmt.Sprintf("Name: %s\nID: %s\nService: %s\nRegion: %s\nProvider: %s",
					r.Name, r.ID, r.Service, r.Region, r.Provider)
			}),
		)

		if err != nil {
			// If the user presses Esc or Ctrl-C, it's an error, but we can just exit gracefully.
			if err == fuzzyfinder.ErrAbort {
				log.Println("Search aborted.")
				return
			}
			log.Fatalf("Error with fuzzy finder: %v", err)
		}

		// Print the selected item's ID
		log.Printf("Selected: %s (%s)\n", resources[idx].Name, resources[idx].ID)
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}