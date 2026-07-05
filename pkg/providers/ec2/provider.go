package ec2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SrJoJ/aws-tui/pkg/provider"
)

// Provider implements provider.ResourceProvider for EC2 Instances.
type Provider struct {
	// We can put an AWS EC2 Client here in the future
}

// NewProvider creates a new EC2 provider.
func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) GetResourceType() string {
	return "EC2 Instances"
}

func (p *Provider) GetShortNames() []string {
	return []string{"ec2", "ins", "instance"}
}

func (p *Provider) GetCategory() string {
	return "Compute"
}

func (p *Provider) List(ctx context.Context, filters map[string]string) ([]provider.Resource, error) {
	// Return mock instances for demonstration/development
	all := []provider.Resource{
		{
			ID:     "i-0123456789abcdef0",
			Name:   "web-server-prod",
			Status: "running",
			Metadata: map[string]string{
				"Type":             "t3.medium",
				"AvailabilityZone": "us-east-1a",
				"PublicIP":         "54.210.12.89",
				"LaunchTime":       time.Now().Add(-120 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			ID:     "i-0987654321fedcba0",
			Name:   "db-replica-01",
			Status: "stopped",
			Metadata: map[string]string{
				"Type":             "r5.large",
				"AvailabilityZone": "us-east-1b",
				"PublicIP":         "N/A",
				"LaunchTime":       time.Now().Add(-240 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			ID:     "i-abcde12345ffeedd0",
			Name:   "bastion-host",
			Status: "running",
			Metadata: map[string]string{
				"Type":             "t3.nano",
				"AvailabilityZone": "us-east-1c",
				"PublicIP":         "3.90.114.5",
				"LaunchTime":       time.Now().Add(-12 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	search, hasSearch := filters["search"]
	if !hasSearch || search == "" {
		return all, nil
	}

	search = strings.ToLower(search)
	var filtered []provider.Resource
	for _, res := range all {
		if strings.Contains(strings.ToLower(res.Name), search) ||
			strings.Contains(strings.ToLower(res.ID), search) ||
			strings.Contains(strings.ToLower(res.Status), search) ||
			strings.Contains(strings.ToLower(res.Metadata["Type"]), search) ||
			strings.Contains(strings.ToLower(res.Metadata["AvailabilityZone"]), search) {
			filtered = append(filtered, res)
		}
	}
	return filtered, nil
}

func (p *Provider) Describe(ctx context.Context, id string) (string, error) {
	return fmt.Sprintf(`---
InstanceID: %s
Name: mock-instance-details
State:
  Code: 16
  Name: running
Placement:
  AvailabilityZone: us-east-1a
  Tenancy: default
SecurityGroups:
  - GroupName: default
  - GroupId: sg-01234567
Tags:
  - Key: Environment
    Value: Production
  - Key: Project
    Value: aws-tui
`, id), nil
}

func (p *Provider) Delete(ctx context.Context, id string) error {
	// Mock termination
	return nil
}

func (p *Provider) GetColumns() []provider.ColumnDefinition {
	return []provider.ColumnDefinition{
		{
			Header: "ID",
			Width:  20,
			ValueFunc: func(r provider.Resource) string {
				return r.ID
			},
		},
		{
			Header: "NAME",
			Width:  25,
			ValueFunc: func(r provider.Resource) string {
				return r.Name
			},
		},
		{
			Header: "STATUS",
			Width:  15,
			ValueFunc: func(r provider.Resource) string {
				return r.Status
			},
		},
		{
			Header: "TYPE",
			Width:  15,
			ValueFunc: func(r provider.Resource) string {
				return r.Metadata["Type"]
			},
		},
		{
			Header: "ZONE",
			Width:  15,
			ValueFunc: func(r provider.Resource) string {
				return r.Metadata["AvailabilityZone"]
			},
		},
		{
			Header: "PUBLIC IP",
			Width:  15,
			ValueFunc: func(r provider.Resource) string {
				return r.Metadata["PublicIP"]
			},
		},
	}
}

func (p *Provider) GetCustomActions() []provider.CustomAction {
	return []provider.CustomAction{
		{
			Name:        "SSH Connect",
			Description: "Connect to instance using SSM Session Manager",
			Hotkey:      "s",
			Type:        "command",
			ActionFunc: func(ctx context.Context, r provider.Resource) (string, error) {
				// To be implemented using SSM
				return "", nil
			},
		},
	}
}
