// server/server.go
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rahulwagh/infrakit/cache"
	"github.com/rahulwagh/infrakit/fetcher"
)

//go:embed index.html
var content embed.FS

// --- handleSearch function ---
func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]fetcher.StandardizedResource{})
		return
	}
	resources, err := cache.LoadResources()
	if err != nil {
		http.Error(w, "Failed to load cache. Run 'sync' first.", http.StatusInternalServerError)
		return
	}
	var results []fetcher.StandardizedResource
	lowerQuery := strings.ToLower(query)
	for _, res := range resources {
		if res.Service == "project" || res.Service == "ec2" {
			searchText := strings.ToLower(res.Name + " " + res.ID)
			if strings.Contains(searchText, lowerQuery) {
				results = append(results, res)
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// --- handleGetResources function ---
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
	groupedChildren := make(map[string][]fetcher.StandardizedResource)
	for _, res := range allResources {
		if projID, ok := res.Attributes["project_id"]; ok && projID == parentProjectID {
			groupedChildren[res.Service] = append(groupedChildren[res.Service], res)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groupedChildren)
}

// --- handleGetLBFlows function ---
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

	backendServices := make(map[string]fetcher.StandardizedResource)
	urlMaps := make(map[string]fetcher.StandardizedResource)
	targetProxies := make(map[string]fetcher.StandardizedResource)
	cloudRunServices := make(map[string]fetcher.StandardizedResource)

	for _, res := range allResources {
		if proj, ok := res.Attributes["project_id"]; ok && proj == projectID {
			// Create a mock SelfLink for lookup, as it's not in the base object
			// Note: This simplified link generation might need adjustment if regional resources are involved.
			selfLink := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/%ss/%s", projectID, res.Service, res.Name)
			switch res.Service {
			case "backendservice":
				backendServices[selfLink] = res
			case "urlmap":
				urlMaps[selfLink] = res
			case "targethttpsproxy":
				targetProxies[selfLink] = res
			case "cloudrun":
				cloudRunServices[res.Name] = res // Cloud Run lookup is by name
			}
		}
	}

	var flows []fetcher.LoadBalancerFlow
	for _, res := range allResources {
		if res.Service == "forwardingrule" && res.Attributes["project_id"] == projectID {
			flow := fetcher.LoadBalancerFlow{
				Name:      res.Name,
				ProjectID: projectID,
				Frontend: fetcher.FrontendConfig{
					IPAddress:           res.Attributes["ip_address"],
					PortRange:           res.Attributes["port_range"],
					Protocol:            res.Attributes["protocol"],
					LoadBalancingScheme: res.Attributes["load_balancing_scheme"],
				},
			}
			if proxy, ok := targetProxies[res.Attributes["target"]]; ok {
				// Populate SSL certificates from target proxy
				if certStr, exists := proxy.Attributes["ssl_certificates"]; exists && certStr != "" {
					flow.Frontend.Certificates = strings.Split(certStr, ",")
				}
				// Populate SSL policy from target proxy
				if sslPolicy, exists := proxy.Attributes["ssl_policy"]; exists {
					flow.Frontend.SSLPolicy = sslPolicy
				}
				if urlMap, ok2 := urlMaps[proxy.Attributes["url_map"]]; ok2 {
					flow.RoutingRules = append(flow.RoutingRules, fetcher.RoutingRule{Hosts: []string{"all"}, PathMatcher: "default"}) // Simplified routing
					if bs, ok3 := backendServices[urlMap.Attributes["default_service"]]; ok3 {
						flow.Backend.Name = bs.Name
						for _, cr := range cloudRunServices {
							if strings.Contains(bs.Name, cr.Name) { // Simple name matching
								flow.Backend.Type = "Cloud Run"; flow.Backend.ServiceName = cr.Name; flow.Backend.Region = cr.Region; break
							}
						}
						if policyURL, ok := bs.Attributes["cloud_armor_policy"]; ok && policyURL != "" {
							flow.CloudArmor.Name = policyURL[strings.LastIndex(policyURL, "/")+1:]
						}
						flows = append(flows, flow) // Add flow only if backend service is found
					}
				}
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(flows)
}

// --- handleGetIAMTemplate function ---
func handleGetIAMTemplate(w http.ResponseWriter, r *http.Request) {
	templateBytes, err := content.ReadFile("iam_tab.html")
	if err != nil {
		log.Printf("Error reading embedded iam_tab.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(templateBytes)
}


// --- CORRECTED StartServer function ---
func StartServer() {
	// Create a file system from the embedded content.
	embeddedFS := http.FS(content)

	// Serve static files (index.html, iam_tab.html if needed directly) from the root.
	// http.FileServer automatically serves index.html for "/" requests from the FS root.
	http.Handle("/", http.FileServer(embeddedFS))

	// API endpoints (these take precedence over the file server for specific paths)
	http.HandleFunc("/api/search", handleSearch)
	http.HandleFunc("/api/resources", handleGetResources)
	http.HandleFunc("/api/lb-flows", handleGetLBFlows)
	http.HandleFunc("/templates/iam", handleGetIAMTemplate) // Still needed for the IAM tab JS

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}