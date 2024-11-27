package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"
	"maps"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type sqsQueue struct{}

// DeleteResource implements ResourceProvider.
func (*sqsQueue) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := sqs.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteQueue(ctx, &sqs.DeleteQueueInput{
		QueueUrl: &r.ID[0],
	})
	return err
}

func (*sqsQueue) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var result []Resource

	c := sqs.NewFromConfig(s.AwsConfig)
	p := sqs.NewListQueuesPaginator(c, &sqs.ListQueuesInput{})
	for p.HasMorePages() {
		qs, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing queues: %s", err)
		}
		for _, qUrl := range qs.QueueUrls {
			var r Resource
			r.Type = ResourceTypeSQSQueue
			r.ID = []string{qUrl}
			r.Tags = map[string]string{}
			result = append(result, r)

			ts, err := c.ListQueueTags(ctx, &sqs.ListQueueTagsInput{
				QueueUrl: &qUrl,
			})
			if err != nil {
				return nil, fmt.Errorf("listing queue tags: %s", err)
			}
			maps.Copy(r.Tags, ts.Tags)
		}
	}

	return result, nil
}

// Type implements ResourceProvider.
func (s *sqsQueue) Type() string {
	return ResourceTypeSQSQueue
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &sqsQueue{}
	})
}
