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