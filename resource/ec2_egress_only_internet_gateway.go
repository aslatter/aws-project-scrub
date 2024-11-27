package resource

import (
	"aws-project-scrub/config"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type egressOnlyInternetGateway struct{}

// DeleteResource implements ResourceProvider.
func (e *egressOnlyInternetGateway) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteEgressOnlyInternetGateway(ctx, &ec2.DeleteEgressOnlyInternetGatewayInput{
		EgressOnlyInternetGatewayId: &r.ID[0],
	})
	return err
}

// Type implements ResourceProvider.
func (e *egressOnlyInternetGateway) Type() string {
	return ResourceTypeEC2EgressOnlyInternetGateway
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &egressOnlyInternetGateway{}
	})
}
