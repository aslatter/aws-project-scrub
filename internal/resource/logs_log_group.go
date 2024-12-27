package resource

import (
	"context"
	"fmt"
	"maps"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type logsLogGroup struct{}

// DeleteResource implements ResourceProvider.
func (l *logsLogGroup) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := cloudwatchlogs.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteLogGroup(ctx, &cloudwatchlogs.DeleteLogGroupInput{
		LogGroupName: &r.ID[0],
	})
	return err
}

// Type implements ResourceProvider.
func (l *logsLogGroup) Type() string {
	return ResourceTypeLogsLogGroup
}

func (l *logsLogGroup) FindResources(ctx context.Context, s *Settings) ([]Resource, error) {
	var result []Resource

	c := cloudwatchlogs.NewFromConfig(s.AwsConfig)
	lgp := cloudwatchlogs.NewDescribeLogGroupsPaginator(c, &cloudwatchlogs.DescribeLogGroupsInput{})
	for lgp.HasMorePages() {
		lgs, err := lgp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing log groups: %s", err)
		}
		for _, lg := range lgs.LogGroups {
			var r Resource
			r.Type = l.Type()
			r.ID = []string{*lg.LogGroupName}
			r.Tags = map[string]string{}
			result = append(result, r)

			tags, err := c.ListTagsForResource(ctx, &cloudwatchlogs.ListTagsForResourceInput{
				ResourceArn: lg.LogGroupArn,
			})
			if err != nil {
				return nil, fmt.Errorf("listing tags for %q (%q): %s", r, *lg.Arn, err)
			}
			maps.Copy(r.Tags, tags.Tags)
		}
	}

	return result, nil
}

func (l *logsLogGroup) Dependencies() []string {
	// some resources try to create log-groups on-the-fly, so we want to
	// wait until compute-resources are torn-down before deleting log-groups.
	return []string{ResourceTypeEC2VPC}
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &logsLogGroup{}
	})
}
