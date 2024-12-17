package resource

import (
	"context"
	"fmt"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type securityGroup struct{}

// DeleteResource implements ResourceProvider.
func (*securityGroup) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)

	// TODO - drop VPC associations
	// this is for security groups shared across VPCs.
	// https://github.com/aws/aws-sdk-go-v2/issues/2911

	_, err := c.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId: &r.ID[0],
	})
	return err
}

func (*securityGroup) DependentResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	groupID := r.ID[0]
	var results []Resource

	// because we need to delete any security-group-rules referencing this security
	// group, we drop all rules before any security groups.
	c := ec2.NewFromConfig(s.AwsConfig)
	rules, err := c.DescribeSecurityGroupRules(ctx, &ec2.DescribeSecurityGroupRulesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-id"),
				Values: []string{groupID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("describing security-group rules: %s", err)
	}
	for _, rule := range rules.SecurityGroupRules {
		var r Resource
		r.Type = ResourceTypeEC2SecurityGroupRule

		ruleType := "ingress"
		if *rule.IsEgress {
			ruleType = "egress"
		}
		r.ID = []string{
			groupID,
			ruleType,
			*rule.SecurityGroupRuleId,
		}
		results = append(results, r)
	}

	return results, nil
}

func (s *securityGroup) Dependencies() []string {
	// we cannot delete the security group until anything
	// referencing it is also gone.
	return []string{
		ResourceTypeEC2Instance,
		ResourceTypeLoadBalancer,
		ResourceTypeEKSCluster,
	}
}

// Type implements ResourceProvider.
func (s *securityGroup) Type() string {
	return ResourceTypeEC2SecurityGroup
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &securityGroup{}
	})
}
