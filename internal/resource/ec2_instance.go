package resource

import (
	"context"
	"fmt"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type ec2Instance struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2Instance) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{r.ID[0]},
	})
	if err != nil {
		return err
	}

	w := ec2.NewInstanceTerminatedWaiter(c)
	err = w.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{r.ID[0]},
	}, defaultDeleteWaitTime)
	if err != nil {
		return fmt.Errorf("waiting for instance termination: %s", err)
	}

	return nil
}

func (*ec2Instance) Dependencies() []string {
	// clean up EKS first, as it will fight against instance-deletion
	// and create more instances.
	return []string{ResourceTypeEKSCluster}
}

// Type implements ResourceProvider.
func (e *ec2Instance) Type() string {
	return ResourceTypeEC2Instance
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &ec2Instance{}
	})
}
