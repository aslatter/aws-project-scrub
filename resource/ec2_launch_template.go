package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ec2LaunchTemplate struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2LaunchTemplate) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteLaunchTemplate(ctx, &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: &r.ID[0],
	})
	return err
}

func (*ec2LaunchTemplate) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var result []Resource
	c := ec2.NewFromConfig(s.AwsConfig)
	p := ec2.NewDescribeLaunchTemplatesPaginator(c, &ec2.DescribeLaunchTemplatesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + s.Filter.TagKey),
				Values: []string{s.Filter.TagValue},
			},
		},
	})
	for p.HasMorePages() {
		lts, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing launch-templates: %s", err)
		}
		for _, lt := range lts.LaunchTemplates {
			var r Resource
			r.Type = ResourceTypeEC2LaunchTemplate
			r.ID = []string{*lt.LaunchTemplateId}
			r.Tags = map[string]string{}
			result = append(result, r)

			for _, t := range lt.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}
		}
	}

	return result, nil
}

// Type implements ResourceProvider.
func (e *ec2LaunchTemplate) Type() string {
	return ResourceTypeEC2LaunchTemplate
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &ec2LaunchTemplate{}
	})
}
