package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"
	"time"

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
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("waiting for instance termination: %s", err)
	}

	return nil
}

// DependentResources implements ResourceProvider.
func (e *ec2Instance) DependentResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	return nil, nil
}

// FindResources implements ResourceProvider.
func (e *ec2Instance) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	return nil, nil
}

// IsGlobal implements ResourceProvider.
func (e *ec2Instance) IsGlobal() bool {
	return false
}

// Type implements ResourceProvider.
func (e *ec2Instance) Type() string {
	return ResourceTypeEC2Instance
}

func init() {
	register(&ec2Instance{})
}
