package resource

import (
	"aws-project-scrub/config"
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamInstanceProfile struct{}

// RelatedResources implements ResourceProvider.
func (i *iamInstanceProfile) RelatedResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	return nil, nil
}

// IsGlobal implements ResourceProvider.
func (i *iamInstanceProfile) IsGlobal() bool {
	return true
}

// DeleteResource implements ResourceProvider.
func (i *iamInstanceProfile) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := iam.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: &r.ID[0],
	})
	return err
}

// FindResources implements ResourceProvider.
func (i *iamInstanceProfile) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	return nil, nil
}

// Dependencies implements Resource.
func (i *iamInstanceProfile) Dependencies() []string {
	return []string{}
}

// Type implements Resource.
func (i *iamInstanceProfile) Type() string {
	return ResourceTypeIAMInstanceProfile
}

func init() {
	register(&iamInstanceProfile{})
}
