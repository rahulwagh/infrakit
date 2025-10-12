// server/server.go
package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"embed" // NEW: Import the embed package

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
	"github.com/lithammer/fuzzysearch/fuzzy" // CHANGED
)

//go:embed index.html
// NEW: This directive tells Go to embed the index.html file into the 'content' variable.
var content embed.FS

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

	ranks := fuzzy.RankFind(query, searchTargets)
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
	// CHANGED: We now create a file server that serves content directly from our embedded variable.
	fs := http.FileServer(http.FS(content))
	http.Handle("/", fs)

	// This is our API endpoint for searching.
	http.HandleFunc("/search", handleSearch)

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}