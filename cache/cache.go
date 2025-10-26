// cache/cache.go
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rahulwagh/infrakit/fetcher" // CHANGE THIS to your module path
)

const cacheFile = "cache.json"

// getCacheDir gets the path to the cache directory, creating it if it doesn't exist.
func getCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".infrakit")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

// SaveResources saves the list of resources to the cache file.
func SaveResources(resources []fetcher.StandardizedResource) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return fmt.Errorf("failed to get cache directory: %w", err)
	}
	filePath := filepath.Join(cacheDir, cacheFile)

	// Marshal the data into pretty-printed JSON
	data, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resources to JSON: %w", err)
	}

	// Write the data to the file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Add this function to cache/cache.go

// LoadResources loads the list of resources from the cache file.
func LoadResources() ([]fetcher.StandardizedResource, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}
	filePath := filepath.Join(cacheDir, cacheFile)

	// Check if the cache file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cache file not found. Please run 'sync' first")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var resources []fetcher.StandardizedResource
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return resources, nil
}

// MergeResourcesForProject intelligently merges new resources for a specific project
// into the existing cache. It removes old resources from that project and adds the new ones,
// while preserving resources from all other projects.
func MergeResourcesForProject(newResources []fetcher.StandardizedResource, projectID string) error {
	// Load existing cache
	existingResources, err := LoadResources()
	if err != nil {
		// If cache doesn't exist yet, just save the new resources
		if os.IsNotExist(err) {
			return SaveResources(newResources)
		}
		return fmt.Errorf("failed to load existing cache: %w", err)
	}

	// Filter out resources belonging to the specified project
	var filteredResources []fetcher.StandardizedResource
	for _, resource := range existingResources {
		// Keep the resource if it doesn't belong to the project being synced
		if !belongsToProject(resource, projectID) {
			filteredResources = append(filteredResources, resource)
		}
	}

	// Add the newly fetched resources for the project
	filteredResources = append(filteredResources, newResources...)

	// Save the merged cache
	return SaveResources(filteredResources)
}

// belongsToProject checks if a resource belongs to a specific GCP project
func belongsToProject(resource fetcher.StandardizedResource, projectID string) bool {
	// For GCP resources only
	if resource.Provider != "gcp" {
		return false
	}

	// Check if the resource is the project itself
	if resource.Service == "project" && resource.ID == projectID {
		return true
	}

	// Check if the resource has project_id in attributes
	if resource.Attributes != nil {
		if projID, exists := resource.Attributes["project_id"]; exists && projID == projectID {
			return true
		}
	}

	return false
}