package resource

import (
	"aws-project-scrub/config"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eks"
)

type eksPodIdentityAssoc struct{}

// DeleteResource implements ResourceProvider.
func (e *eksPodIdentityAssoc) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := eks.NewFromConfig(s.AwsConfig)
	_, err := c.DeletePodIdentityAssociation(ctx, &eks.DeletePodIdentityAssociationInput{
		ClusterName:   &r.ID[0],
		AssociationId: &r.ID[1],
	})
	return err
}

// Type implements ResourceProvider.
func (e *eksPodIdentityAssoc) Type() string {
	return ResourceTypeEKSPodIdentityAssociation
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &eksPodIdentityAssoc{}
	})
}