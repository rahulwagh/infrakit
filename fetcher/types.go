// fetcher/types.go
package fetcher

// StandardizedResource is our common format for any cloud resource.
type StandardizedResource struct {
	Provider   string            `json:"provider"`
	Service    string            `json:"service"`
	Region     string            `json:"region"`
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}