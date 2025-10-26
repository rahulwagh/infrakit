# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Infrakit is a multi-cloud resource search CLI written in Go. It provides blazing-fast fuzzy searching of AWS and GCP resources through both an interactive terminal interface and a web UI. The tool works by caching cloud resource metadata locally and enabling instant searches without hitting cloud APIs repeatedly.

## Core Architecture

### Three-Layer Design

1. **Fetchers** (`fetcher/` package)
   - Cloud-specific resource collectors that fetch metadata from AWS and GCP
   - Each fetcher converts cloud-native resource formats into `StandardizedResource` struct
   - AWS fetchers: `FetchEC2Instances()`, `FetchIAMRoles()` in `fetcher/aws_fetcher.go`
   - GCP fetchers: `FetchGCPResourcesFromOrg()`, `FetchGCPProjectsNoOrg()` in `fetcher/gcp_*.go`
   - The `StandardizedResource` type (in `fetcher/types.go`) is the universal format for all cloud resources

2. **Cache** (`cache/` package)
   - Stores resources in `~/.infrakit/cache.json`
   - Provides `SaveResources()` and `LoadResources()` functions
   - All searches operate against this local cache, not live cloud APIs

3. **Commands** (`cmd/` package)
   - Built with cobra framework
   - `sync` - Fetches resources from cloud providers and updates cache
   - `search` - Interactive fuzzy finder using `go-fuzzyfinder`
   - `serve` - Starts HTTP server on port 8080 for web-based search

### Data Flow

```
Cloud APIs → Fetchers → StandardizedResource[] → Cache (JSON) → Search/Serve commands
```

## Common Development Commands

### Quick Start with Makefile

The project includes a Makefile for common tasks:

```bash
# Run all checks (format, test, build)
make check

# Build for current platform
make build

# Build binaries for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# See all available targets
make help
```

### Building the Application

```bash
# Build for current platform
go build -o infrakit
# or
make build

# Build for all platforms
make build-all

# Build manually for specific platforms
GOOS=linux GOARCH=amd64 go build -o dist/infrakit-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o dist/infrakit-macos-arm64
```

### Running Commands

```bash
# Sync resources from all providers
go run main.go sync

# Sync only AWS resources
go run main.go sync aws

# Sync all GCP resources
go run main.go sync gcp

# Sync a specific GCP project (intelligent merge with existing cache)
go run main.go sync gcp my-project-id

# Launch interactive search
go run main.go search

# Start web server
go run main.go serve
```

### Installing Locally

```bash
go install
```

This installs to `$HOME/go/bin/infrakit`

## Key Implementation Details

### Authentication

- **AWS**: Uses AWS SDK v2's default credential chain (respects `~/.aws/credentials`, environment variables, IAM roles)
- **GCP**: Uses Application Default Credentials (respects `GOOGLE_APPLICATION_CREDENTIALS`, `gcloud auth application-default login`)

### Adding New Cloud Resources

To add support for a new AWS/GCP resource type:

1. Create a new fetcher function in the appropriate file (e.g., `FetchS3Buckets()` in `fetcher/aws_fetcher.go`)
2. Convert the cloud-specific resource to `StandardizedResource` format
3. Call the new fetcher in `cmd/sync.go` Run function
4. Append results to `allResources` slice before saving cache

Example pattern:
```go
func FetchNewResource() ([]StandardizedResource, error) {
    var resources []StandardizedResource
    // ... fetch from cloud API
    for _, item := range apiResponse {
        resource := StandardizedResource{
            Provider: "aws",
            Service: "s3",
            Region: item.Region,
            ID: item.ID,
            Name: item.Name,
            Attributes: map[string]string{
                "key": "value",
            },
        }
        resources = append(resources, resource)
    }
    return resources, nil
}
```

### Web Server Routes

The `server/server.go` file serves both static HTML (embedded via `//go:embed`) and API endpoints:

- `GET /` - Serves `index.html` (search interface)
- `GET /api/search?q=<query>` - Fuzzy search cached resources
- `GET /api/resources?parent=<project_id>` - Get resources for a GCP project
- `GET /api/lb-flows?project=<project_id>` - Get load balancer flow details
- `GET /templates/iam` - Serves IAM template HTML

## Module Path

The Go module path is `github.com/rahulwagh/infrakit`. When importing packages:

```go
import "github.com/rahulwagh/infrakit/cache"
import "github.com/rahulwagh/infrakit/fetcher"
```

## Testing Strategy

Infrakit has a comprehensive test suite to ensure code quality and prevent regressions when adding new features.

### Test Structure

Tests are organized alongside their source files:
- `cache/cache_test.go` - Tests for cache operations (save, load, merge)
- `fetcher/types_test.go` - Tests for resource type serialization and validation

### Running Tests

Use the Makefile for convenient test execution:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage

# Run tests for specific package
make test-cache
make test-fetcher

# Run tests with race detection
make test-race

# Or use go test directly
go test ./...
go test -v ./cache/...
go test -v -cover ./...
```

### Test Coverage

Key areas covered by tests:
- **Cache Operations**: Save, load, and merge operations with temporary test directories
- **Resource Serialization**: JSON marshaling/unmarshaling of StandardizedResource
- **Project Filtering**: belongsToProject() logic for intelligent cache merging
- **Edge Cases**: Empty caches, nil attributes, missing files

### Writing New Tests

When adding new features:
1. Create test file alongside source (e.g., `myfile_test.go` next to `myfile.go`)
2. Use table-driven tests for multiple scenarios
3. Mock cloud API calls to avoid requiring credentials
4. Use temporary directories for file operations
5. Clean up resources in defer statements

Example test pattern:
```go
func TestNewFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "output1"},
        {"case 2", "input2", "output2"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := NewFeature(tt.input)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}
```

### CI/CD Integration

Run the full CI suite before committing:
```bash
make ci
```

This runs formatting, tests with coverage, and builds all platform binaries.

## GCP Organization vs Project-based Discovery

The tool supports two GCP discovery modes:

1. **Organization-based** (`DiscoverGCPOrganization()` in `gcp_discovery.go`)
   - Uses Cloud Asset API to scan entire org hierarchy
   - Fetches folders, projects, and resources in one pass
   - Requires organization-level permissions

2. **Project-based** (`FetchGCPProjectsNoOrg()`)
   - Falls back when no org is found
   - Uses Resource Manager API to list accessible projects
   - Then fetches resources per-project

## Embedded Web Assets

The web UI files are embedded into the binary using `//go:embed` directives in `server/server.go`. To modify the UI:

1. Edit `server/index.html` or `server/iam_tab.html`
2. Rebuild the binary - changes are automatically included

## Cache Location

All cached data is stored in `~/.infrakit/cache.json`. The cache format is a JSON array of `StandardizedResource` objects.
