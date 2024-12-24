package resource

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2RouteTable struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2RouteTable) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{
		RouteTableId: &r.ID[0],
	})
	return err
}

func (e *ec2RouteTable) Dependencies() []string {
	// we cannot delete a route table associated with a subnet.
	return []string{ResourceTypeEC2Subnet}
}

// Type implements ResourceProvider.
func (e *ec2RouteTable) Type() string {
	return ResourceTypeEC2RouteTable
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &ec2RouteTable{}
	})
}
