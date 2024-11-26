package resource

import (
	"aws-project-scrub/config"
	"context"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type elbTargetGroup struct{}

func (e *elbTargetGroup) Dependencies() []string {
	return []string{ResourceTypeLoadBalancer}
}

// DeleteResource implements ResourceProvider.
func (e *elbTargetGroup) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := elb.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteTargetGroup(ctx, &elb.DeleteTargetGroupInput{
		TargetGroupArn: &r.ID[0],
	})
	return err
}

// Type implements ResourceProvider.
func (e *elbTargetGroup) Type() string {
	return ResourceTypeLoadBalancerTargetGroup
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &elbTargetGroup{}
	})
}
