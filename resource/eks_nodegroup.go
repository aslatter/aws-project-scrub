package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
)

type eksNodegroup struct{}

// DeleteResource implements ResourceProvider.
func (e *eksNodegroup) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	if len(r.ID) != 2 {
		return fmt.Errorf("invalid id: %q", strings.Join(r.ID, "/"))
	}
	cluster := r.ID[0]
	nodegroup := r.ID[1]

	c := eks.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
		ClusterName:   &cluster,
		NodegroupName: &nodegroup,
	})
	if err != nil {
		return err
	}

	w := eks.NewNodegroupDeletedWaiter(c)
	err = w.Wait(ctx, &eks.DescribeNodegroupInput{
		ClusterName:   &cluster,
		NodegroupName: &nodegroup,
	}, 5*time.Minute)

	if err != nil {
		return fmt.Errorf("waiting for deletion: %s", err)
	}

	return nil
}

// FindResources implements ResourceProvider.
func (e *eksNodegroup) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	return nil, nil
}

// IsGlobal implements ResourceProvider.
func (e *eksNodegroup) IsGlobal() bool {
	return false
}

// RelatedResources implements ResourceProvider.
func (e *eksNodegroup) RelatedResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	return nil, nil
}

// Type implements ResourceProvider.
func (e *eksNodegroup) Type() string {
	return ResourceTypeEKSNodegroup
}

func init() {
	register(&eksNodegroup{})
}
