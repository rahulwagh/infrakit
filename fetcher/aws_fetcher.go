// fetcher/aws_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
		"github.com/aws/aws-sdk-go-v2/service/iam" // <-- This is the corrected line
)

// FetchEC2Instances contains the logic to fetch all EC2 instances.
func FetchEC2Instances() ([]StandardizedResource, error) {
	var resources []StandardizedResource

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	client := ec2.NewFromConfig(cfg)
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	log.Println("Fetching EC2 instances...")

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to get a page of EC2 instances: %w", err)
		}

		for _, reservation := range page.Reservations {
			for _, instance := range reservation.Instances {
				instanceName := "N/A"
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						instanceName = *tag.Value
						break
					}
				}

				resource := StandardizedResource{
					Provider: "aws",
					Service:  "ec2",
					Region:   cfg.Region,
					ID:       *instance.InstanceId,
					Name:     instanceName,
					Attributes: map[string]string{
						"instance_type": string(instance.InstanceType),
						"state":         string(instance.State.Name),
					},
				}
				resources = append(resources, resource)
			}
		}
	}
	log.Printf("Successfully fetched %d EC2 instances.\n", len(resources))
	return resources, nil
}

// FetchIAMRoles contains the logic to fetch all IAM roles and their policies.
func FetchIAMRoles() ([]StandardizedResource, error) {
	var resources []StandardizedResource

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	client := iam.NewFromConfig(cfg)
	paginator := iam.NewListRolesPaginator(client, &iam.ListRolesInput{})

	log.Println("Fetching IAM roles...")

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to get a page of IAM roles: %w", err)
		}

		for _, role := range page.Roles {
			var policyNames []string

			attachedPaginator := iam.NewListAttachedRolePoliciesPaginator(client, &iam.ListAttachedRolePoliciesInput{
				RoleName: role.RoleName,
			})
			for attachedPaginator.HasMorePages() {
				attachedPage, err := attachedPaginator.NextPage(context.TODO())
				if err != nil {
					log.Printf("could not list attached policies for role %s: %v", *role.RoleName, err)
					continue
				}
				for _, policy := range attachedPage.AttachedPolicies {
					policyNames = append(policyNames, *policy.PolicyName)
				}
			}

			inlinePaginator := iam.NewListRolePoliciesPaginator(client, &iam.ListRolePoliciesInput{
				RoleName: role.RoleName,
			})
			for inlinePaginator.HasMorePages() {
				inlinePage, err := inlinePaginator.NextPage(context.TODO())
				if err != nil {
					log.Printf("could not list inline policies for role %s: %v", *role.RoleName, err)
					continue
				}
				for _, policyName := range inlinePage.PolicyNames {
					policyNames = append(policyNames, policyName)
				}
			}

			resource := StandardizedResource{
				Provider: "aws",
				Service:  "iam",
				Region:   "global",
				ID:       *role.Arn,
				Name:     *role.RoleName,
				Attributes: map[string]string{
					"policies": strings.Join(policyNames, ", "),
				},
			}
			resources = append(resources, resource)
		}
	}

	log.Printf("Successfully fetched %d IAM roles.\n", len(resources))
	return resources, nil
}