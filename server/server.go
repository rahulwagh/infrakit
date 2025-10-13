// server/server.go
package server

import (
	"embed"
	"encoding/json"
	//"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

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

// CORRECTED: Full implementation for building Load Balancer flows on-demand from cache.
func handleGetLBFlows(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project")
	if projectID == "" {
		http.Error(w, "query parameter 'project' is required", http.StatusBadRequest)
		return
	}

	allResources, err := cache.LoadResources()
	if err != nil {
		http.Error(w, "Failed to load cache", http.StatusInternalServerError)
		return
	}

	// Create maps for easy lookup of components by their NAME.
	backendServices := make(map[string]fetcher.StandardizedResource)
	urlMaps := make(map[string]fetcher.StandardizedResource)
	targetProxies := make(map[string]fetcher.StandardizedResource)
	cloudRunServices := make(map[string]fetcher.StandardizedResource)

	for _, res := range allResources {
		if proj, ok := res.Attributes["project_id"]; ok && proj == projectID {
			switch res.Service {
			case "backendservice":
				backendServices[res.Name] = res
			case "urlmap":
				urlMaps[res.Name] = res
			case "targethttpsproxy":
				targetProxies[res.Name] = res
			case "cloudrun":
				cloudRunServices[res.Name] = res
			}
		}
	}

	var flows []fetcher.LoadBalancerFlow

	// Start from the outside: Iterate through all cached Forwarding Rules for the project.
	for _, res := range allResources {
		if res.Service == "forwardingrule" && res.Attributes["project_id"] == projectID {
			flow := fetcher.LoadBalancerFlow{
				Name:      res.Name,
				ProjectID: projectID,
				Frontend: fetcher.FrontendConfig{
					IPAddress: res.Attributes["ip_address"],
					PortRange: res.Attributes["port_range"],
					Protocol:  res.Attributes["protocol"],
				},
			}

			// 1. Trace Forwarding Rule -> Target Proxy
			targetURL := res.Attributes["target"]
			proxyName := targetURL[strings.LastIndex(targetURL, "/")+1:]
			if proxy, ok := targetProxies[proxyName]; ok {

				// 2. Trace Target Proxy -> URL Map
				urlMapURL := proxy.Attributes["url_map"]
				urlMapName := urlMapURL[strings.LastIndex(urlMapURL, "/")+1:]
				if urlMap, ok2 := urlMaps[urlMapName]; ok2 {
					flow.RoutingRules = append(flow.RoutingRules, fetcher.RoutingRule{Hosts: []string{"all"}, PathMatcher: "default"})

					// 3. Trace URL Map -> Backend Service
					backendServiceURL := urlMap.Attributes["default_service"]
					backendServiceName := backendServiceURL[strings.LastIndex(backendServiceURL, "/")+1:]
					if bs, ok3 := backendServices[backendServiceName]; ok3 {
						flow.Backend.Name = bs.Name

						// 4. Find associated Cloud Run service (simple match)
						for _, cr := range cloudRunServices {
							if strings.Contains(bs.Name, cr.Name) {
								flow.Backend.Type = "Cloud Run"
								flow.Backend.ServiceName = cr.Name
								flow.Backend.Region = cr.Region
								break
							}
						}

						// 5. Add Cloud Armor details
						if policyURL, ok := bs.Attributes["cloud_armor_policy"]; ok && policyURL != "" {
							flow.CloudArmor.Name = policyURL[strings.LastIndex(policyURL, "/")+1:]
						}
						flows = append(flows, flow)
					}
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
}

func StartServer() {
	http.Handle("/", http.FileServer(http.FS(content)))
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/resources", handleGetResources)
	http.HandleFunc("/api/lb-flows", handleGetLBFlows)

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}