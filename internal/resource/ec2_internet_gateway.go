package resource

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type internetGateway struct{}

// DeleteResource implements ResourceProvider.
func (i *internetGateway) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)

	// TODO - move to root resource, as we could fail between the detach and delete?

	gws, err := c.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		InternetGatewayIds: []string{r.ID[0]},
	})
	if err != nil {
		return fmt.Errorf("describing internet gateway: %w", err)
	}
	if len(gws.InternetGateways) != 1 {
		return fmt.Errorf("unexpected count of internet gateways: %d", len(gws.InternetGateways))
	}

	if len(gws.InternetGateways) > 0 && len(gws.InternetGateways[0].Attachments) > 0 {
		_, err = c.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{
			InternetGatewayId: &r.ID[0],
			VpcId:             gws.InternetGateways[0].Attachments[0].VpcId,
		})
		if err != nil {
			return fmt.Errorf("detaching internet gateway: %s", err)
		}
	}

	_, err = c.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: &r.ID[0],
	})

	return err
}

func (i *internetGateway) FindResources(ctx context.Context, s *Settings) ([]Resource, error) {
	var result []Resource

	c := ec2.NewFromConfig(s.AwsConfig)
	igp := ec2.NewDescribeInternetGatewaysPaginator(c, &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + s.Filter.TagKey),
				Values: []string{s.Filter.TagValue},
			},
		},
	})

	for igp.HasMorePages() {
		igs, err := igp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing internet gateways: %s", err)
		}
		for _, ig := range igs.InternetGateways {
			var r Resource
			r.Type = i.Type()
			r.ID = []string{*ig.InternetGatewayId}
			r.Tags = map[string]string{}
			result = append(result, r)

			for _, t := range ig.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}
		}
	}

	return result, nil
}

// wait for stuff which could be using IPv4 routes
func (i *internetGateway) Dependencies() []string {
	return []string{
		ResourceTypeEC2Instance,
		ResourceTypeLoadBalancer,
		ResourceTypeEC2NATGateway,
		ResourceTypeEKSCluster,
	}
}

// Type implements ResourceProvider.
func (i *internetGateway) Type() string {
	return ResourceTypeEC2InternetGateway
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &internetGateway{}
	})
}
