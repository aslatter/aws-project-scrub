package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

type iamRole struct{}

// IsGlobal implements ResourceProvider.
func (i *iamRole) IsGlobal() bool {
	return true
}

// Dependencies implements Resource.
func (i *iamRole) Dependencies() []string {
	return []string{"AWS::IAM::InstanceProfile"}
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
	iamClient := iam.NewFromConfig(s.AwsConfig)

	// detach role policies
	var marker *string
	for {
		inlineRoles, err := iamClient.ListRolePolicies(ctx, &iam.ListRolePoliciesInput{
			RoleName: &r.ID,
			Marker:   marker,
		})
		if err != nil {
			return fmt.Errorf("listing role policies: %s", err)
		}

		for _, p := range inlineRoles.PolicyNames {
			_, err := iamClient.DeleteRolePolicy(ctx, &iam.DeleteRolePolicyInput{
				RoleName:   &r.ID,
				PolicyName: &p,
			})
			if err != nil {
				return fmt.Errorf("deleting role policy %q: %s", p, err)
			}
		}

		if inlineRoles.IsTruncated || inlineRoles.Marker == nil {
			break
		}
		marker = inlineRoles.Marker
	}

	// delete role policies
	marker = nil
	for {
		rolePolicies, err := iamClient.ListAttachedRolePolicies(ctx, &iam.ListAttachedRolePoliciesInput{
			RoleName: &r.ID,
			Marker:   marker,
		})
		if err != nil {
			return fmt.Errorf("listing attached role policies: %s", err)
		}

		for _, p := range rolePolicies.AttachedPolicies {
			if p.PolicyArn == nil {
				continue
			}
			_, err := iamClient.DetachRolePolicy(ctx, &iam.DetachRolePolicyInput{
				RoleName:  &r.ID,
				PolicyArn: p.PolicyArn,
			})
			if err != nil {
				return fmt.Errorf("detaching role policy %q: %s", *p.PolicyArn, err)
			}
		}

		if rolePolicies.IsTruncated || rolePolicies.Marker == nil {
			break
		}
		marker = rolePolicies.Marker
	}

	// delete role
	_, err := iamClient.DeleteRole(ctx, &iam.DeleteRoleInput{
		RoleName: &r.ID,
	})

	return err
}

// FindResources implements Resource.
func (i *iamRole) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var foundRoles []Resource
	iamClient := iam.NewFromConfig(s.AwsConfig)

	var marker *string
	for {
		result, err := iamClient.ListRoles(ctx, &iam.ListRolesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, fmt.Errorf("listing roles: %s", err)
		}
		for _, role := range result.Roles {
			if role.RoleName == nil {
				// ??
				continue
			}

			var r Resource
			r.ID = *role.RoleName
			r.Tags = map[string]string{}
			foundRoles = append(foundRoles, r)

			for _, t := range role.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}
		}
		// get next batch of roles
		if !result.IsTruncated || result.Marker == nil {
			break
		}
		marker = result.Marker
	}

	return foundRoles, nil
}

func init() {
	register(&iamRole{})
}
