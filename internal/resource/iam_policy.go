package resource

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

type iamPolicy struct{}

// DeleteResource implements ResourceProvider.
func (i *iamPolicy) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := iam.NewFromConfig(s.AwsConfig)

	pvp := iam.NewListPolicyVersionsPaginator(c, &iam.ListPolicyVersionsInput{
		PolicyArn: &r.ID[0],
	})
	for pvp.HasMorePages() {
		versions, err := pvp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing policy versions for %q: %s", r, err)
		}
		for _, version := range versions.Versions {
			if version.IsDefaultVersion {
				continue
			}
			_, err := c.DeletePolicyVersion(ctx, &iam.DeletePolicyVersionInput{
				PolicyArn: &r.ID[0],
				VersionId: version.VersionId,
			})
			if err != nil {
				return fmt.Errorf("deleting policy version for %q: %s", r, err)
			}
		}
	}

	_, err := c.DeletePolicy(ctx, &iam.DeletePolicyInput{
		PolicyArn: &r.ID[0],
	})
	return err
}

func (*iamPolicy) Dependencies() []string {
	return []string{ResourceTypeIAMRole}
}

func (*iamPolicy) IsGlobal() bool {
	return true
}

func (*iamPolicy) FindResources(ctx context.Context, s *Settings) ([]Resource, error) {
	var result []Resource

	c := iam.NewFromConfig(s.AwsConfig)
	p := iam.NewListPoliciesPaginator(c, &iam.ListPoliciesInput{
		Scope: types.PolicyScopeTypeLocal,
	})
	for p.HasMorePages() {
		ps, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing policies: %s", err)
		}
		for _, p := range ps.Policies {
			var r Resource
			r.Type = ResourceTypeIAMPolicy
			r.ID = []string{*p.Arn}
			r.Tags = map[string]string{}
			result = append(result, r)

			tp := iam.NewListPolicyTagsPaginator(c, &iam.ListPolicyTagsInput{
				PolicyArn: p.Arn,
			})
			for tp.HasMorePages() {
				ts, err := tp.NextPage(ctx)
				if err != nil {
					return nil, fmt.Errorf("listing policies: %s", err)
				}
				for _, t := range ts.Tags {
					if t.Key == nil || t.Value == nil {
						continue
					}
					r.Tags[*t.Key] = *t.Value
				}
			}
		}
	}

	return result, nil
}

// Type implements ResourceProvider.
func (i *iamPolicy) Type() string {
	return ResourceTypeIAMPolicy
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &iamPolicy{}
	})
}
