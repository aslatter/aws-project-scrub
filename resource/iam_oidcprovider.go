package resource

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/iam"

	"aws-project-scrub/config"
	"context"
)

type iamOIDCProvider struct{}

// IsGlobal implements ResourceProvider.
func (i *iamOIDCProvider) IsGlobal() bool {
	return true
}

// DeleteResource implements ResourceProvider.
func (i *iamOIDCProvider) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	iamClient := iam.NewFromConfig(s.AwsConfig)
	_, err := iamClient.DeleteOpenIDConnectProvider(ctx, &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &r.ID,
	})
	return err
}

// Dependencies implements Resource.
func (i *iamOIDCProvider) Dependencies() []string {
	// "AWS::IAM::Role"?
	return []string{}
}

// Type implements Resource.
func (i *iamOIDCProvider) Type() string {
	return "AWS::IAM::OIDCProvider"
}

func (i *iamOIDCProvider) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	iamClient := iam.NewFromConfig(s.AwsConfig)
	var found []Resource

	listResult, err := iamClient.ListOpenIDConnectProviders(ctx, &iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, fmt.Errorf("listing oidc providers: %s", err)
	}

	for _, provider := range listResult.OpenIDConnectProviderList {
		if provider.Arn == nil {
			// ?!?
			continue
		}

		var r Resource
		r.ID = *provider.Arn
		r.Tags = map[string]string{}
		found = append(found, r)

		var tagLoopMarker *string
		for {
			tagResult, err := iamClient.ListOpenIDConnectProviderTags(ctx, &iam.ListOpenIDConnectProviderTagsInput{
				OpenIDConnectProviderArn: provider.Arn,
				Marker:                   tagLoopMarker,
			})
			if err != nil {
				return nil, fmt.Errorf("listing oidc provider tags: %s", err)
			}

			tagLoopMarker = tagResult.Marker

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
	register(&iamOIDCProvider{})
}
