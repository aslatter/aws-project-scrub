package resource

import (
	"context"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type vpcEndpoint struct{}

// DeleteResource implements ResourceProvider.
func (v *vpcEndpoint) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteVpcEndpoints(ctx, &ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: []string{r.ID[0]},
	})
	return err
}

// Type implements ResourceProvider.
func (v *vpcEndpoint) Type() string {
	return ResourceTypeEC2VPCEndpoint
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &vpcEndpoint{}
	})
}