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

### Building the Application

```bash
# Build for current platform
go build -o infrakit

# Build for Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o dist/infrakit-linux-amd64

# Build for macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o dist/infrakit-macos-arm64
```

### Running Commands

```bash
# Sync resources from all providers
go run main.go sync

# Sync only AWS resources
go run main.go sync aws

# Sync only GCP resources
go run main.go sync gcp

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

Currently no test files exist. When adding tests:
- Place test files alongside source files (e.g., `cache/cache_test.go`)
- Run with `go test ./...`
- Mock cloud API calls to avoid requiring actual credentials

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
