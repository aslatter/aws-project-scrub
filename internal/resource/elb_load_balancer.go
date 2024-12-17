package resource

import (
	"context"
	"fmt"

	"github.com/aslatter/aws-project-scrub/internal/config"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type elbLoadBalancer struct{}

// DeleteResource implements ResourceProvider.
func (e *elbLoadBalancer) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := elb.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteLoadBalancer(ctx, &elb.DeleteLoadBalancerInput{
		LoadBalancerArn: &r.ID[0],
	})
	if err != nil {
		return err
	}

	w := elb.NewLoadBalancersDeletedWaiter(c)
	err = w.Wait(ctx, &elb.DescribeLoadBalancersInput{
		LoadBalancerArns: []string{r.ID[0]},
	}, defaultDeleteWaitTime)
	if err != nil {
		return fmt.Errorf("waiting for load-balancer deletion: %s", err)
	}

	return nil
}

// Type implements ResourceProvider.
func (e *elbLoadBalancer) Type() string {
	return ResourceTypeLoadBalancer
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &elbLoadBalancer{}
	})
}
