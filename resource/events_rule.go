package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
)

type eventsRule struct{}

// DeleteResource implements ResourceProvider.
func (e *eventsRule) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := eventbridge.NewFromConfig(s.AwsConfig)

	// remove targets
	ts, err := c.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
		Rule: &r.ID[0],
	})
	if err != nil {
		return fmt.Errorf("listing targets: %s", err)
	}
	for _, target := range ts.Targets {
		_, err := c.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
			Rule: &r.ID[0],
			Ids:  []string{*target.Id},
		})
		if err != nil {
			return fmt.Errorf("removing rule target: %s", err)
		}
	}

	// delete rule
	_, err = c.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name: &r.ID[0],
	})
	return err
}

func (*eventsRule) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var result []Resource

	c := eventbridge.NewFromConfig(s.AwsConfig)

	// no paginator types?

	var nextToken *string
	for {
		rules, err := c.ListRules(ctx, &eventbridge.ListRulesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("listing rules: %s", err)
		}

		for _, rule := range rules.Rules {
			var r Resource
			r.Type = ResourceTypeEventsRule
			r.ID = []string{*rule.Name}
			r.Tags = map[string]string{}
			result = append(result, r)

			tags, err := c.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
				ResourceARN: rule.Arn,
			})
			if err != nil {
				return nil, fmt.Errorf("listing tags: %s", err)
			}
			for _, t := range tags.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}
		}

		if rules.NextToken == nil {
			break
		}
		nextToken = rules.NextToken
	}

	return result, nil
}

// Type implements ResourceProvider.
func (e *eventsRule) Type() string {
	return ResourceTypeEventsRule
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &eventsRule{}
	})
}
