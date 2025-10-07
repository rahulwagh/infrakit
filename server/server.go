// server/server.go
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
	"github.com/lithammer/fuzzysearch/fuzzy" // CHANGED
)

// handleSearch is the function that powers our /search API endpoint.
func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	resources, err := cache.LoadResources()
	if err != nil {
		http.Error(w, "Failed to load cache. Run 'sync' first.", http.StatusInternalServerError)
		return
	}

	var searchTargets []string
	for _, res := range resources {
		searchTargets = append(searchTargets, res.Name+" "+res.ID)
	}

	// Use the fuzzy subpackage directly.
	ranks := fuzzy.RankFind(query, searchTargets) // CHANGED
	sort.Sort(ranks)

	var results []fetcher.StandardizedResource
	for _, rank := range ranks {
		results = append(results, resources[rank.OriginalIndex])
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// StartServer starts the local web server.
func StartServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "server/index.html")
	})

	http.HandleFunc("/search", handleSearch)

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}