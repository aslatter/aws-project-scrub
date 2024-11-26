package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type natGateway struct{}

// DeleteResource implements ResourceProvider.
func (n *natGateway) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)

	_, err := c.DeleteNatGateway(ctx, &ec2.DeleteNatGatewayInput{
		NatGatewayId: &r.ID[0],
	})
	if err != nil {
		return err
	}

	w := ec2.NewNatGatewayDeletedWaiter(c)
	err = w.Wait(ctx, &ec2.DescribeNatGatewaysInput{
		NatGatewayIds: []string{r.ID[0]},
	}, defaultDeleteWaitTime)
	if err != nil {
		return fmt.Errorf("waiting for deletion: %s", err)
	}

	return nil
}

// Type implements ResourceProvider.
func (n *natGateway) Type() string {
	return ResourceTypeEC2NATGateway
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &natGateway{}
	})
}
