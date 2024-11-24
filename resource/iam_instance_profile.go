package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamInstanceProfile struct{}

// IsGlobal implements ResourceProvider.
func (i *iamInstanceProfile) IsGlobal() bool {
	return true
}

// DeleteResource implements ResourceProvider.
func (i *iamInstanceProfile) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := iam.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: &r.ID,
	})
	return err
}

// FindResources implements ResourceProvider.
func (i *iamInstanceProfile) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	c := iam.NewFromConfig(s.AwsConfig)
	var found []Resource

	p := iam.NewListInstanceProfilesPaginator(c, &iam.ListInstanceProfilesInput{})
	for p.HasMorePages() {
		result, err := p.NextPage(ctx)

		if err != nil {
			return nil, fmt.Errorf("listing instance profiles: %s", err)
		}

		for _, p := range result.InstanceProfiles {
			if p.InstanceProfileName == nil {
				// ??
				continue
			}

			var r Resource
			r.ID = *p.InstanceProfileName
			r.Tags = map[string]string{}
			found = append(found, r)

			iptp := iam.NewListInstanceProfileTagsPaginator(c, &iam.ListInstanceProfileTagsInput{
				InstanceProfileName: &r.ID,
			})

			for iptp.HasMorePages() {
				result, err := iptp.NextPage(ctx)
				if err != nil {
					return nil, fmt.Errorf("listing instance profile tags: %s", err)
				}

				for _, t := range result.Tags {
					if t.Key == nil || t.Value == nil {
						continue
					}
					r.Tags[*t.Key] = *t.Value
				}
			}

		}
	}

	return found, nil
}

// Dependencies implements Resource.
func (i *iamInstanceProfile) Dependencies() []string {
	return []string{}
}

// Type implements Resource.
func (i *iamInstanceProfile) Type() string {
	return "AWS::IAM::InstanceProfile"
}

func init() {
	register(&iamInstanceProfile{})
}
