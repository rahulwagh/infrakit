// fetcher/aws_fetcher.go
package fetcher

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// StandardizedResource is our common format for any cloud resource.
type StandardizedResource struct {
	Provider   string            `json:"provider"`
	Service    string            `json:"service"`
	Region     string            `json:"region"`
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Attributes map[string]string `json:"attributes"`
}

// This function will contain the logic to fetch all EC2 instances.
func FetchEC2Instances() ([]StandardizedResource, error) {
	var resources []StandardizedResource

	// Load the default AWS configuration from ~/.aws/credentials
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create an EC2 client
	client := ec2.NewFromConfig(cfg)

	// Use a Paginator to handle multiple pages of results
	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	log.Println("Fetching EC2 instances...")

	// Loop through each page of results
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("failed to get a page of EC2 instances: %w", err)
		}

		// Loop through each reservation in the page
		for _, reservation := range page.Reservations {
			// Loop through each instance in the reservation
			for _, instance := range reservation.Instances {
				// Find the 'Name' tag of the instance
				instanceName := "N/A"
				for _, tag := range instance.Tags {
					if *tag.Key == "Name" {
						instanceName = *tag.Value
						break
					}
				}

				// Create our standardized resource object
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