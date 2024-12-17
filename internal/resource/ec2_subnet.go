package resource

import (
	"context"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2Subnet struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2Subnet) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{
		SubnetId: &r.ID[0],
	})
	return err
}

func (*ec2Subnet) Dependencies() []string {
	// wait for stuff which could have interfaces on
	// subnets
	return []string{
		ResourceTypeEC2Instance,
		ResourceTypeEKSCluster,
		ResourceTypeLoadBalancer,
		ResourceTypeEC2VPCEndpoint,
	}
}

// Type implements ResourceProvider.
func (e *ec2Subnet) Type() string {
	return ResourceTypeEC2Subnet
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &ec2Subnet{}
	})
}
