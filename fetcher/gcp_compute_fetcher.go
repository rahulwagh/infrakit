// fetcher/gcp_compute_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/compute/v1"
)

// FetchGCPNetworkResourcesForProject scans a single project for its networking components.
func FetchGCPNetworkResourcesForProject(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var networkResources []StandardizedResource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service for project %s: %w", projectID, err)
	}
	log.Printf("   -> Fetching network resources for project: %s", projectID)

	networks, _ := computeService.Networks.List(projectID).Do()
	if networks != nil {
		for _, network := range networks.Items {
			networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "vpc", Region: "global", ID: network.Name, Name: network.Name, Attributes: map[string]string{"project_id": projectID, "mode": fmt.Sprintf("%t", network.AutoCreateSubnetworks)}})
		}
	}
	subnets, _ := computeService.Subnetworks.AggregatedList(projectID).Do()
	if subnets != nil {
		for _, scope := range subnets.Items {
			for _, subnet := range scope.Subnetworks {
				networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "subnet", Region: subnet.Region, ID: subnet.Name, Name: subnet.Name, Attributes: map[string]string{"project_id": projectID, "vpc": subnet.Network, "cidr_range": subnet.IpCidrRange}})
			}
		}
	}
	firewallList, _ := computeService.Firewalls.List(projectID).Do()
	if firewallList != nil {
		for _, listRule := range firewallList.Items {
			rule, err := computeService.Firewalls.Get(projectID, listRule.Name).Do()
			if err != nil {
				log.Printf("Warning: could not get full details for firewall rule %s: %v", listRule.Name, err)
				continue
			}
			formatAllowedRules := func(details []*compute.FirewallAllowed) string {
				var p []string
				for _, d := range details {
					r := d.IPProtocol
					if len(d.Ports) > 0 {
						r += ":" + strings.Join(d.Ports, ",")
					}
					p = append(p, r)
				}
				return strings.Join(p, "; ")
			}
			formatDeniedRules := func(details []*compute.FirewallDenied) string {
				var p []string
				for _, d := range details {
					r := d.IPProtocol
					if len(d.Ports) > 0 {
						r += ":" + strings.Join(d.Ports, ",")
					}
					p = append(p, r)
				}
				return strings.Join(p, "; ")
			}
			action := "DENY"
			if len(rule.Allowed) > 0 {
				action = "ALLOW"
			}
			attributes := map[string]string{"project_id": projectID, "action": action, "direction": rule.Direction, "priority": fmt.Sprintf("%d", rule.Priority), "disabled": fmt.Sprintf("%t", rule.Disabled), "source_ranges": strings.Join(rule.SourceRanges, ", "), "destination_ranges": strings.Join(rule.DestinationRanges, ", "), "target_tags": strings.Join(rule.TargetTags, ", "), "allowed": formatAllowedRules(rule.Allowed), "denied": formatDeniedRules(rule.Denied)}
			networkResources = append(networkResources, StandardizedResource{Provider: "gcp", Service: "firewall", Region: "global", ID: rule.Name, Name: rule.Name, Attributes: attributes})
		}
	}
	return networkResources, nil
}

// FetchGCPAppInfraForProject scans a single project for application infrastructure like LBs.
func FetchGCPAppInfraForProject(projectID string) ([]StandardizedResource, error) {
	ctx := context.Background()
	var appResources []StandardizedResource
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create compute service for project %s: %w", projectID, err)
	}
	log.Printf("   -> Fetching App infrastructure for project: %s", projectID)

	backendServices, _ := computeService.BackendServices.AggregatedList(projectID).Do()
	if backendServices != nil {
		for _, scope := range backendServices.Items {
			for _, bs := range scope.BackendServices {
				appResources = append(appResources, StandardizedResource{Provider: "gcp", Service: "backendservice", ID: bs.Name, Name: bs.Name, Attributes: map[string]string{"project_id": projectID, "load_balancing_scheme": bs.LoadBalancingScheme, "cloud_armor_policy": bs.SecurityPolicy}})
			}
		}
	}
	urlMaps, _ := computeService.UrlMaps.AggregatedList(projectID).Do()
	if urlMaps != nil {
		for _, scope := range urlMaps.Items {
			for _, um := range scope.UrlMaps {
				appResources = append(appResources, StandardizedResource{Provider: "gcp", Service: "urlmap", ID: um.Name, Name: um.Name, Attributes: map[string]string{"project_id": projectID, "default_service": um.DefaultService}})
			}
		}
	}
	targetProxies, _ := computeService.TargetHttpsProxies.AggregatedList(projectID).Do()
	if targetProxies != nil {
		for _, scope := range targetProxies.Items {
			for _, proxy := range scope.TargetHttpsProxies {
				appResources = append(appResources, StandardizedResource{Provider: "gcp", Service: "targethttpsproxy", ID: proxy.Name, Name: proxy.Name, Attributes: map[string]string{"project_id": projectID, "url_map": proxy.UrlMap}})
			}
		}
	}
	forwardingRules, _ := computeService.GlobalForwardingRules.List(projectID).Do()
	if forwardingRules != nil {
		for _, fr := range forwardingRules.Items {
			appResources = append(appResources, StandardizedResource{Provider: "gcp", Service: "forwardingrule", ID: fr.Name, Name: fr.Name, Attributes: map[string]string{"project_id": projectID, "ip_address": fr.IPAddress, "port_range": fr.PortRange, "target": fr.Target}})
		}
	}
	return appResources, nil
}

// FetchGCPLoadBalancerFlows traces connections from Forwarding Rules to Backends.
func FetchGCPLoadBalancerFlows(projectID string) ([]LoadBalancerFlow, error) {
	ctx := context.Background()
	var flows []LoadBalancerFlow
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create compute service: %w", err)
	}
	log.Printf("   -> Tracing Load Balancer flows for project: %s", projectID)

	forwardingRules, err := computeService.GlobalForwardingRules.List(projectID).Do()
	if err != nil {
		return nil, fmt.Errorf("could not list forwarding rules: %w", err)
	}
	for _, fr := range forwardingRules.Items {
		flow := LoadBalancerFlow{
			Name:      fr.Name,
			ProjectID: projectID,
			Frontend: FrontendConfig{
				IPAddress:           fr.IPAddress,
				PortRange:           fr.PortRange,
				Protocol:            fr.IPProtocol,
				LoadBalancingScheme: fr.LoadBalancingScheme,
			},
		}
		proxyName := strings.Split(fr.Target, "/")[len(strings.Split(fr.Target, "/"))-1]
		httpsProxy, err := computeService.TargetHttpsProxies.Get(projectID, proxyName).Do()
		if err == nil {
			flow.Frontend.Certificates = httpsProxy.SslCertificates
			flow.Frontend.SSLPolicy = httpsProxy.SslPolicy
			urlMapName := strings.Split(httpsProxy.UrlMap, "/")[len(strings.Split(httpsProxy.UrlMap, "/"))-1]
			urlMap, err := computeService.UrlMaps.Get(projectID, urlMapName).Do()
			if err == nil {
				for _, hostRule := range urlMap.HostRules {
					flow.RoutingRules = append(flow.RoutingRules, RoutingRule{Hosts: hostRule.Hosts, PathMatcher: hostRule.PathMatcher})
				}
				backendServiceName := strings.Split(urlMap.DefaultService, "/")[len(strings.Split(urlMap.DefaultService, "/"))-1]
				backendService, err := computeService.BackendServices.Get(projectID, backendServiceName).Do()
				if err == nil {
					flow.Backend.Name = backendService.Name
					for _, backend := range backendService.Backends {
						if strings.Contains(backend.Group, "run.googleapis.com") {
							flow.Backend.Type = "Cloud Run"
							negName := strings.Split(backend.Group, "/")[len(strings.Split(backend.Group, "/"))-1]
							neg, err := computeService.RegionNetworkEndpointGroups.Get(projectID, backendService.Region, negName).Do()
							if err == nil && neg.CloudRun != nil {
								flow.Backend.ServiceName = neg.CloudRun.Service
								flow.Backend.Region = backendService.Region
							}
						}
					}
					if backendService.SecurityPolicy != "" {
						policyName := strings.Split(backendService.SecurityPolicy, "/")[len(strings.Split(backendService.SecurityPolicy, "/"))-1]
						policy, err := computeService.SecurityPolicies.Get(projectID, policyName).Do()
						if err == nil {
							flow.CloudArmor.Name = policy.Name
							for _, rule := range policy.Rules {
								flow.CloudArmor.Rules = append(flow.CloudArmor.Rules, CloudArmorRule{Priority: rule.Priority, Action: rule.Action, Description: rule.Description, Match: fmt.Sprintf("%+v", rule.Match.Config)})
							}
						}
					}
				}
			}
		}
		flows = append(flows, flow)
	}
	return flows, nil
}