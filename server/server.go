// server/server.go
package server

import (
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
	"github.com/lithammer/fuzzysearch/fuzzy" // CHANGED
)

//go:embed index.html
var content embed.FS

// handleSearch is for the initial fuzzy search.
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
		// We only want to search for top-level resources like projects initially.
		if res.Service == "project" || res.Service == "ec2" {
			searchTargets = append(searchTargets, res.Name+" "+res.ID)
		}
	}
	ranks := fuzzy.RankFind(query, searchTargets)
	sort.Sort(ranks)
	var results []fetcher.StandardizedResource


	// A simpler search for now: iterate all resources to find matches.
	for _, res := range resources {
		if fuzzy.Match(query, res.Name+" "+res.ID) {
			results = append(results, res)
		}
	}


	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// NEW: handleGetResources finds all child resources for a given parent project.
func handleGetResources(w http.ResponseWriter, r *http.Request) {
	parentProjectID := r.URL.Query().Get("parent")
	if parentProjectID == "" {
		http.Error(w, "query parameter 'parent' is required", http.StatusBadRequest)
		return
	}

	allResources, err := cache.LoadResources()
	if err != nil {
		http.Error(w, "Failed to load cache", http.StatusInternalServerError)
		return
	}

	// Group child resources by their service type (vpc, subnet, etc.)
	groupedChildren := make(map[string][]fetcher.StandardizedResource)

	for _, res := range allResources {
		if projID, ok := res.Attributes["project_id"]; ok && projID == parentProjectID {
			groupedChildren[res.Service] = append(groupedChildren[res.Service], res)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groupedChildren)
}

// StartServer starts the local web server.
func StartServer() {
	fs := http.FileServer(http.FS(content))
	http.Handle("/", fs)

	// API endpoint for the initial fuzzy search
	http.HandleFunc("/api/search", handleSearch)

	// NEW: API endpoint to get children of a specific resource
	http.HandleFunc("/api/resources", handleGetResources)

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}