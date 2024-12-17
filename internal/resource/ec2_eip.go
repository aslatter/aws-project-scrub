package resource

import (
	"context"
	"fmt"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ec2EIP struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2EIP) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
		AllocationId: &r.ID[0],
	})
	return err
}

// Dependencies returns resource-providers which must run before this one.
func (e *ec2EIP) Dependencies() []string {
	// wait until we're done removing anything that could be using our
	// IPs.
	return []string{ResourceTypeEC2VPC}
}

// FindResources implements ResourceProvider.
func (e *ec2EIP) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var result []Resource

	c := ec2.NewFromConfig(s.AwsConfig)

	addresses, err := c.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + s.Filter.TagKey),
				Values: []string{s.Filter.TagValue},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describe addresses: %s", err)
	}

	for _, a := range addresses.Addresses {
		var r Resource
		r.ID = []string{*a.AllocationId}
		r.Type = ResourceTypeEC2EIP
		r.Tags = map[string]string{}
		result = append(result, r)

		for _, t := range a.Tags {
			if t.Key == nil || t.Value == nil {
				continue
			}
			r.Tags[*t.Key] = *t.Value
		}
	}

	return result, nil
}

// Type implements ResourceProvider.
func (e *ec2EIP) Type() string {
	return ResourceTypeEC2EIP
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &ec2EIP{}
	})
}
