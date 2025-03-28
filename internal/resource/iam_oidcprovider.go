package resource

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"context"
)

type iamOIDCProvider struct{}

// IsGlobal implements ResourceProvider.
func (i *iamOIDCProvider) IsGlobal() bool {
	return true
}

// DeleteResource implements ResourceProvider.
func (i *iamOIDCProvider) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := iam.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteOpenIDConnectProvider(ctx, &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &r.ID[0],
	})
	return err
}

// Type implements Resource.
func (i *iamOIDCProvider) Type() string {
	return "AWS::IAM::OIDCProvider"
}

func (i *iamOIDCProvider) FindResources(ctx context.Context, s *Settings) ([]Resource, error) {
	c := iam.NewFromConfig(s.AwsConfig)
	var found []Resource

	listResult, err := c.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, fmt.Errorf("listing oidc providers: %s", err)
	}

	for _, provider := range listResult.OpenIDConnectProviderList {
		if provider.Arn == nil {
			// ?!?
			continue
		}

		var r Resource
		r.Type = i.Type()
		r.ID = []string{*provider.Arn}
		r.Tags = map[string]string{}
		found = append(found, r)

		p := iam.NewListOpenIDConnectProviderTagsPaginator(c, &iam.ListOpenIDConnectProviderTagsInput{
			OpenIDConnectProviderArn: provider.Arn,
		})
		for p.HasMorePages() {
			tagResult, err := p.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("listing oidc provider tags: %s", err)
			}

			for _, tag := range tagResult.Tags {
				if tag.Key == nil || tag.Value == nil {
					continue
				}
				r.Tags[*tag.Key] = *tag.Value
			}
		}
	}

	return found, nil
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &iamOIDCProvider{}
	})
}
