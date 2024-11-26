package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type ec2Volume struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2Volume) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: &r.ID[0],
	})
	return err
}

func (e *ec2Volume) Dependencies() []string {
	// wait until we're done using the volumes
	return []string{ResourceTypeEC2VPC}
}

// FindResources implements ResourceProvider.
func (e *ec2Volume) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var result []Resource

	c := ec2.NewFromConfig(s.AwsConfig)

	p := ec2.NewDescribeVolumesPaginator(c, &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + s.Filter.TagKey),
				Values: []string{s.Filter.TagValue},
			},
		},
	})
	for p.HasMorePages() {
		vs, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing volumes: %s", err)
		}
		for _, v := range vs.Volumes {
			var r Resource
			r.Type = e.Type()
			r.ID = []string{*v.VolumeId}
			r.Tags = map[string]string{}
			result = append(result, r)

			for _, t := range v.Tags {
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
func (e *ec2Volume) Type() string {
	return ResourceTypeEC2Volume
}

func init() {
	register(&ec2Volume{})
}
