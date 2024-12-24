package resource

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
