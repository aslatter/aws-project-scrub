package resource

import (
	"context"
	"fmt"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type securityGroupRule struct{}

// DeleteResource implements ResourceProvider.
func (*securityGroupRule) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)

	groupID := r.ID[0]
	ruleType := r.ID[1]
	ruleID := r.ID[2]

	switch ruleType {
	case "egress":
		_, err := c.RevokeSecurityGroupEgress(ctx, &ec2.RevokeSecurityGroupEgressInput{
			GroupId:              &groupID,
			SecurityGroupRuleIds: []string{ruleID},
		})
		return err
	case "ingress":
		_, err := c.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
			GroupId:              &groupID,
			SecurityGroupRuleIds: []string{ruleID},
		})
		return err
	}

	return fmt.Errorf("unknown rule type %q", ruleType)
}

// Type implements ResourceProvider.
func (s *securityGroupRule) Type() string {
	return ResourceTypeEC2SecurityGroupRule
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &securityGroupRule{}
	})
}
