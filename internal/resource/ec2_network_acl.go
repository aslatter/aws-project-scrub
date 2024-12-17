package resource

import (
	"context"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type networkACL struct{}

// DeleteResource implements ResourceProvider.
func (n *networkACL) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteNetworkAcl(ctx, &ec2.DeleteNetworkAclInput{
		NetworkAclId: &r.ID[0],
	})
	return err
}

func (n *networkACL) Dependencies() []string {
	// we cannot delete ACLs associated with a subnet.
	// the easy way to get around this is to wait for subnets
	// to be deleted.
	return []string{ResourceTypeEC2Subnet}
}

// Type implements ResourceProvider.
func (n *networkACL) Type() string {
	return ResourceTypeEC2NetworkACL
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &networkACL{}
	})
}
