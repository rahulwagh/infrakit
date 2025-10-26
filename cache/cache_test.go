package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rahulwagh/infrakit/fetcher"
)

// setupTestCache creates a temporary cache directory for testing
func setupTestCache(t *testing.T) (string, func()) {
	tempDir, err := os.MkdirTemp("", "infrakit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Override the cache directory for tests
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)

	cleanup := func() {
		os.Setenv("HOME", originalHome)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// createTestResources creates sample resources for testing
func createTestResources() []fetcher.StandardizedResource {
	return []fetcher.StandardizedResource{
		{
			Provider: "gcp",
			Service:  "project",
			Region:   "global",
			ID:       "test-project-1",
			Name:     "Test Project 1",
			Attributes: map[string]string{
				"state":          "ACTIVE",
				"project_number": "123456789",
			},
		},
		{
			Provider: "gcp",
			Service:  "vpc",
			Region:   "global",
			ID:       "test-vpc-1",
			Name:     "Test VPC",
			Attributes: map[string]string{
				"project_id": "test-project-1",
				"mode":       "true",
			},
		},
		{
			Provider: "gcp",
			Service:  "cloud-run",
			Region:   "us-central1",
			ID:       "test-service-1",
			Name:     "Test Service",
			Attributes: map[string]string{
				"project_id": "test-project-1",
				"url":        "https://test-service.run.app",
			},
		},
		{
			Provider: "gcp",
			Service:  "project",
			Region:   "global",
			ID:       "test-project-2",
			Name:     "Test Project 2",
			Attributes: map[string]string{
				"state":          "ACTIVE",
				"project_number": "987654321",
			},
		},
		{
			Provider: "gcp",
			Service:  "vpc",
			Region:   "global",
			ID:       "test-vpc-2",
			Name:     "Test VPC 2",
			Attributes: map[string]string{
				"project_id": "test-project-2",
				"mode":       "false",
			},
		},
		{
			Provider: "aws",
			Service:  "ec2",
			Region:   "us-east-1",
			ID:       "i-1234567890",
			Name:     "Test EC2 Instance",
			Attributes: map[string]string{
				"state": "running",
			},
		},
	}
}

func TestSaveResources(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	resources := createTestResources()

	err := SaveResources(resources)
	if err != nil {
		t.Fatalf("SaveResources failed: %v", err)
	}

	// Verify the cache file was created
	cacheDir, _ := getCacheDir()
	cachePath := filepath.Join(cacheDir, cacheFile)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("Cache file was not created")
	}

	// Verify the content is valid JSON
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var savedResources []fetcher.StandardizedResource
	if err := json.Unmarshal(data, &savedResources); err != nil {
		t.Fatalf("Cache file contains invalid JSON: %v", err)
	}

	// Verify the number of resources
	if len(savedResources) != len(resources) {
		t.Errorf("Expected %d resources, got %d", len(resources), len(savedResources))
	}
}

func TestLoadResources(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	originalResources := createTestResources()

	// Save resources first
	if err := SaveResources(originalResources); err != nil {
		t.Fatalf("Failed to save resources: %v", err)
	}

	// Load resources
	loadedResources, err := LoadResources()
	if err != nil {
		t.Fatalf("LoadResources failed: %v", err)
	}

	// Verify the count
	if len(loadedResources) != len(originalResources) {
		t.Errorf("Expected %d resources, got %d", len(originalResources), len(loadedResources))
	}

	// Verify some specific resources
	if loadedResources[0].Provider != "gcp" {
		t.Errorf("Expected provider 'gcp', got '%s'", loadedResources[0].Provider)
	}

	if loadedResources[0].ID != "test-project-1" {
		t.Errorf("Expected ID 'test-project-1', got '%s'", loadedResources[0].ID)
	}
}

func TestLoadResourcesNoCacheFile(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Try to load without creating a cache file
	_, err := LoadResources()
	if err == nil {
		t.Fatal("Expected error when cache file doesn't exist, got nil")
	}
}

func TestBelongsToProject(t *testing.T) {
	tests := []struct {
		name      string
		resource  fetcher.StandardizedResource
		projectID string
		expected  bool
	}{
		{
			name: "GCP project resource - matches",
			resource: fetcher.StandardizedResource{
				Provider: "gcp",
				Service:  "project",
				ID:       "my-project",
				Name:     "My Project",
			},
			projectID: "my-project",
			expected:  true,
		},
		{
			name: "GCP project resource - doesn't match",
			resource: fetcher.StandardizedResource{
				Provider: "gcp",
				Service:  "project",
				ID:       "other-project",
				Name:     "Other Project",
			},
			projectID: "my-project",
			expected:  false,
		},
		{
			name: "GCP resource with project_id attribute - matches",
			resource: fetcher.StandardizedResource{
				Provider: "gcp",
				Service:  "vpc",
				ID:       "my-vpc",
				Name:     "My VPC",
				Attributes: map[string]string{
					"project_id": "my-project",
				},
			},
			projectID: "my-project",
			expected:  true,
		},
		{
			name: "GCP resource with project_id attribute - doesn't match",
			resource: fetcher.StandardizedResource{
				Provider: "gcp",
				Service:  "vpc",
				ID:       "other-vpc",
				Name:     "Other VPC",
				Attributes: map[string]string{
					"project_id": "other-project",
				},
			},
			projectID: "my-project",
			expected:  false,
		},
		{
			name: "AWS resource - never matches GCP project",
			resource: fetcher.StandardizedResource{
				Provider: "aws",
				Service:  "ec2",
				ID:       "i-123456",
				Name:     "My Instance",
			},
			projectID: "my-project",
			expected:  false,
		},
		{
			name: "GCP resource without project_id attribute",
			resource: fetcher.StandardizedResource{
				Provider:   "gcp",
				Service:    "folder",
				ID:         "folder-123",
				Name:       "My Folder",
				Attributes: map[string]string{},
			},
			projectID: "my-project",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := belongsToProject(tt.resource, tt.projectID)
			if result != tt.expected {
				t.Errorf("belongsToProject() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMergeResourcesForProject(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Setup: Save initial resources (2 projects with resources)
	initialResources := createTestResources()
	if err := SaveResources(initialResources); err != nil {
		t.Fatalf("Failed to save initial resources: %v", err)
	}

	// Create updated resources for test-project-1 (2 resources instead of 3)
	updatedProject1Resources := []fetcher.StandardizedResource{
		{
			Provider: "gcp",
			Service:  "project",
			Region:   "global",
			ID:       "test-project-1",
			Name:     "Test Project 1 - Updated",
			Attributes: map[string]string{
				"state":          "ACTIVE",
				"project_number": "123456789",
			},
		},
		{
			Provider: "gcp",
			Service:  "vpc",
			Region:   "global",
			ID:       "test-vpc-1-new",
			Name:     "Test VPC New",
			Attributes: map[string]string{
				"project_id": "test-project-1",
				"mode":       "false",
			},
		},
	}

	// Merge the updated resources
	err := MergeResourcesForProject(updatedProject1Resources, "test-project-1")
	if err != nil {
		t.Fatalf("MergeResourcesForProject failed: %v", err)
	}

	// Load and verify
	finalResources, err := LoadResources()
	if err != nil {
		t.Fatalf("Failed to load resources after merge: %v", err)
	}

	// Count resources by project
	project1Count := 0
	project2Count := 0
	awsCount := 0

	for _, res := range finalResources {
		if res.Provider == "gcp" {
			if res.Service == "project" && res.ID == "test-project-1" {
				project1Count++
			} else if res.Service == "project" && res.ID == "test-project-2" {
				project2Count++
			} else if res.Attributes != nil {
				if projID, exists := res.Attributes["project_id"]; exists {
					if projID == "test-project-1" {
						project1Count++
					} else if projID == "test-project-2" {
						project2Count++
					}
				}
			}
		} else if res.Provider == "aws" {
			awsCount++
		}
	}

	// Verify: test-project-1 should have 2 resources (updated)
	if project1Count != 2 {
		t.Errorf("Expected 2 resources for test-project-1, got %d", project1Count)
	}

	// Verify: test-project-2 should still have 2 resources (untouched)
	if project2Count != 2 {
		t.Errorf("Expected 2 resources for test-project-2, got %d", project2Count)
	}

	// Verify: AWS resources should still be present (untouched)
	if awsCount != 1 {
		t.Errorf("Expected 1 AWS resource, got %d", awsCount)
	}

	// Verify total count
	expectedTotal := project1Count + project2Count + awsCount
	if len(finalResources) != expectedTotal {
		t.Errorf("Expected %d total resources, got %d", expectedTotal, len(finalResources))
	}

	// Verify the new VPC name exists
	foundNewVPC := false
	for _, res := range finalResources {
		if res.ID == "test-vpc-1-new" {
			foundNewVPC = true
			if res.Name != "Test VPC New" {
				t.Errorf("Expected VPC name 'Test VPC New', got '%s'", res.Name)
			}
		}
		// Old Cloud Run service should be gone
		if res.ID == "test-service-1" {
			t.Error("Old Cloud Run service should have been removed")
		}
	}

	if !foundNewVPC {
		t.Error("New VPC resource was not found after merge")
	}
}

func TestMergeResourcesForProjectNoCacheFile(t *testing.T) {
	_, cleanup := setupTestCache(t)
	defer cleanup()

	// Merge without existing cache should create a new cache
	newResources := []fetcher.StandardizedResource{
		{
			Provider: "gcp",
			Service:  "project",
			Region:   "global",
			ID:       "new-project",
			Name:     "New Project",
		},
	}

	err := MergeResourcesForProject(newResources, "new-project")
	if err != nil {
		t.Fatalf("MergeResourcesForProject failed on empty cache: %v", err)
	}

	// Verify resources were saved
	loadedResources, err := LoadResources()
	if err != nil {
		t.Fatalf("Failed to load resources: %v", err)
	}

	if len(loadedResources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(loadedResources))
	}

	if loadedResources[0].ID != "new-project" {
		t.Errorf("Expected ID 'new-project', got '%s'", loadedResources[0].ID)
	}
}
