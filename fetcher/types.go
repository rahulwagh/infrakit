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

// LoadBalancerFlow represents the entire traceable path of a GCP Load Balancer.
type LoadBalancerFlow struct {
	Name         string           `json:"name"`
	ProjectID    string           `json:"projectId"`
	Frontend     FrontendConfig   `json:"frontend"`
	RoutingRules []RoutingRule    `json:"routingRules"`
	Backend      BackendConfig    `json:"backend"`
	CloudArmor   CloudArmorPolicy `json:"cloudArmor"`
}

// FrontendConfig holds details about the user-facing side of the LB.
type FrontendConfig struct {
	IPAddress           string   `json:"ipAddress"`
	PortRange           string   `json:"portRange"`
	Protocol            string   `json:"protocol"`
	Certificates        []string `json:"certificates"`
	SSLPolicy           string   `json:"sslPolicy"`
	LoadBalancingScheme string   `json:"loadBalancingScheme"`
}

// RoutingRule holds details from the URL Map.
type RoutingRule struct {
	Hosts       []string `json:"hosts"`
	PathMatcher string   `json:"pathMatcher"`
}

// BackendConfig holds details about the final destination of traffic.
type BackendConfig struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // e.g., "Cloud Run"
	ServiceName string `json:"serviceName"`
	Region      string `json:"region"`
}

// CloudArmorPolicy holds details about the attached security policy.
type CloudArmorPolicy struct {
	Name  string           `json:"name"`
	Rules []CloudArmorRule `json:"rules"`
}

// CloudArmorRule holds details for a single rule within a policy.
type CloudArmorRule struct {
	Priority    int64  `json:"priority"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Match       string `json:"match"`
}