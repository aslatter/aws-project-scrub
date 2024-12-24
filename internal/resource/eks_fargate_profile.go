package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/eks"
)

type eksFargateProfile struct{}

// DeleteResource implements ResourceProvider.
func (e *eksFargateProfile) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	if len(r.ID) != 2 {
		return fmt.Errorf("invalid id: %q", strings.Join(r.ID, "/"))
	}
	cluster := r.ID[0]
	profile := r.ID[1]

	c := eks.NewFromConfig(s.AwsConfig)

	_, err := c.DeleteFargateProfile(ctx, &eks.DeleteFargateProfileInput{
		ClusterName:        &cluster,
		FargateProfileName: &profile,
	})
	if err != nil {
		return fmt.Errorf("deleting fargate profile %q: %s", profile, err)
	}

	w := eks.NewFargateProfileDeletedWaiter(c)
	err = w.Wait(ctx, &eks.DescribeFargateProfileInput{
		ClusterName:        &cluster,
		FargateProfileName: &profile,
	}, defaultDeleteWaitTime)
	if err != nil {
		return fmt.Errorf("waiting for deletion: %s", err)
	}

	return nil
}

// Type implements ResourceProvider.
func (e *eksFargateProfile) Type() string {
	return ResourceTypeEKSFargateProfile
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &eksFargateProfile{}
	})
}
