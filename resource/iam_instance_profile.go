package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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

	// removes roles before deletion
	p, err := c.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: &r.ID[0],
	})
	// TODO - allow not-found?
	if err != nil {
		return fmt.Errorf("getting instance profile: %s", err)
	}

	for _, role := range p.InstanceProfile.Roles {
		a, err := arn.Parse(*role.Arn)
		if err != nil {
			return fmt.Errorf("parsing ARN %q: %s", *role.Arn, err)
		}

		// name is last "/" delimited piece
		resourceParts := strings.Split(a.Resource, "/")
		if len(resourceParts) == 0 {
			// ?!?
			return fmt.Errorf("unexpected role ARN: %s", *role.Arn)
		}
		roleName := resourceParts[len(resourceParts)-1]

		_, err = c.RemoveRoleFromInstanceProfile(ctx, &iam.RemoveRoleFromInstanceProfileInput{
			InstanceProfileName: &r.ID[0],
			RoleName:            &roleName,
		})
		if err != nil {
			return fmt.Errorf("removing role from instance profile: %s", err)
		}
	}

	// delete!
	_, err = c.DeleteInstanceProfile(ctx, &iam.DeleteInstanceProfileInput{
		InstanceProfileName: &r.ID[0],
	})
	return err
}

// Type implements Resource.
func (i *iamInstanceProfile) Type() string {
	return ResourceTypeIAMInstanceProfile
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &iamInstanceProfile{}
	})
}
