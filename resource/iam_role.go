package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamRole struct{}

// DependentResources implements ResourceProvider.
func (i *iamRole) DependentResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	c := iam.NewFromConfig(s.AwsConfig)

	// instance profiles
	p := iam.NewListInstanceProfilesForRolePaginator(c, &iam.ListInstanceProfilesForRoleInput{
		RoleName: &r.ID[0],
	})

	var result []Resource

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list instance profiles for role %q: %s", r.ID, err)
		}

		for _, profile := range page.InstanceProfiles {
			var r Resource
			r.Type = ResourceTypeIAMInstanceProfile
			r.ID = []string{*profile.InstanceProfileName}
			result = append(result, r)
		}
	}

	return result, nil
}

// IsGlobal implements ResourceProvider.
func (i *iamRole) IsGlobal() bool {
	return true
}

// Type implements Resource.
func (i *iamRole) Type() string {
	return "AWS::IAM::Role"
}

// DeleteResource implements Resource.
func (i *iamRole) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	/**
	Need to delete first:
		- Inline policies (DeleteRolePolicy )
		- Attached managed policies (DetachRolePolicy )
		- Instance profile (RemoveRoleFromInstanceProfile )
	**/
	c := iam.NewFromConfig(s.AwsConfig)

	// detach role policies
	rpp := iam.NewListRolePoliciesPaginator(c, &iam.ListRolePoliciesInput{
		RoleName: &r.ID[0],
	})
	for rpp.HasMorePages() {
		inlineRoles, err := rpp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing role policies: %s", err)
		}

		for _, p := range inlineRoles.PolicyNames {
			_, err := c.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				RoleName:   &r.ID[0],
				PolicyName: &p,
			})
			if err != nil {
				return fmt.Errorf("deleting role policy %q: %s", p, err)
			}
		}
	}

	// delete role policies
	arpp := iam.NewListAttachedRolePoliciesPaginator(c, &iam.ListAttachedRolePoliciesInput{
		RoleName: &r.ID[0],
	})
	for arpp.HasMorePages() {
		rolePolicies, err := arpp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing attached role policies: %s", err)
		}

		for _, p := range rolePolicies.AttachedPolicies {
			if p.PolicyArn == nil {
				continue
			}
			_, err := c.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  &r.ID[0],
				PolicyArn: p.PolicyArn,
			})
			if err != nil {
				return fmt.Errorf("detaching role policy %q: %s", *p.PolicyArn, err)
			}
		}
	}

	// delete role
	_, err := c.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: &r.ID[0],
	})

	return err
}

// FindResources implements Resource.
func (i *iamRole) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var foundRoles []Resource
	c := iam.NewFromConfig(s.AwsConfig)

	lrp := iam.NewListRolesPaginator(c, &iam.ListRolesInput{})
	for lrp.HasMorePages() {
		result, err := lrp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing roles: %s", err)
		}
		for _, role := range result.Roles {
			if role.RoleName == nil {
				// ??
				continue
			}

			var r Resource
			r.Type = i.Type()
			r.ID = []string{*role.RoleName}
			r.Tags = map[string]string{}
			foundRoles = append(foundRoles, r)

			rtp := iam.NewListRoleTagsPaginator(c, &iam.ListRoleTagsInput{
				RoleName: role.RoleName,
			})
			for rtp.HasMorePages() {
				result, err := rtp.NextPage(ctx)
				if err != nil {
					return nil, fmt.Errorf("listing role tags: %s", err)
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

	return foundRoles, nil
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &iamRole{}
	})
}
