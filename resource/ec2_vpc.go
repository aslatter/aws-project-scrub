package resource

import (
	"aws-project-scrub/config"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
)

type ec2Vpc struct{}

// DeleteResource implements ResourceProvider.
func (e *ec2Vpc) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := ec2.NewFromConfig(s.AwsConfig)

	_, err := c.DeleteVpc(ctx, &ec2.DeleteVpcInput{
		VpcId: &r.ID[0],
	})

	return err
}

// DependentResources implements ResourceProvider.
func (e *ec2Vpc) DependentResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	// https://docs.aws.amazon.com/vpc/latest/userguide/delete-vpc.html

	vpcID := r.ID[0]
	c := ec2.NewFromConfig(s.AwsConfig)

	var results []Resource

	// instances
	ip := ec2.NewDescribeInstancesPaginator(c, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	for ip.HasMorePages() {
		is, err := ip.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing instances: %s", err)
		}
		for _, r := range is.Reservations {
			for _, i := range r.Instances {
				var r Resource
				r.Type = ResourceTypeEC2Instance
				r.ID = []string{*i.InstanceId}
				results = append(results, r)
			}
		}
	}

	// NAT gateways
	ngp := ec2.NewDescribeNatGatewaysPaginator(c, &ec2.DescribeNatGatewaysInput{
		Filter: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	for ngp.HasMorePages() {
		ngs, err := ngp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing NAT Gateways: %s", err)
		}
		for _, ngw := range ngs.NatGateways {
			var r Resource
			r.ID = []string{*ngw.NatGatewayId}
			r.Type = ResourceTypeEC2NATGateway
			results = append(results, r)
		}
	}

	// load balancers (NLB or ALB - Classic LBs are a different API)
	elbClient := elb.NewFromConfig(s.AwsConfig)
	lbp := elb.NewDescribeLoadBalancersPaginator(elbClient, &elb.DescribeLoadBalancersInput{})
	for lbp.HasMorePages() {
		lbs, err := lbp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing load balancers: %s", err)
		}
		for _, lb := range lbs.LoadBalancers {
			if *lb.VpcId != vpcID {
				continue
			}
			var r Resource
			r.ID = []string{*lb.LoadBalancerArn}
			r.Type = ResourceTypeLoadBalancer
			results = append(results, r)
		}
	}

	// load balancer target groups
	lbtp := elb.NewDescribeTargetGroupsPaginator(elbClient, &elb.DescribeTargetGroupsInput{})
	for lbtp.HasMorePages() {
		tgs, err := lbtp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing ELB target groups: %s", err)
		}
		for _, tg := range tgs.TargetGroups {
			if tg.VpcId == nil || *tg.VpcId != vpcID {
				continue
			}
			var r Resource
			r.ID = []string{*tg.TargetGroupArn}
			r.Type = ResourceTypeLoadBalancerTargetGroup
			results = append(results, r)
		}
	}

	return results, nil
}

func (e *ec2Vpc) Dependencies() []string {
	// I'm not sure this is strictly needed ... but it feels like we
	// should clean up things running on the VPC before the VPC.
	//
	// In theory we could discover all EKS clusters which have been configured to
	// have their control-plane on the VPC we are deleting, and not have separate
	// discovery of EKS stuff? (i.e. add a VPC loop to the above method).
	//
	// Any other "compute platforms" hosted on VPC would also go here.
	return []string{ResourceTypeEKSCluster}
}

// FindResources implements ResourceProvider.
func (e *ec2Vpc) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var results []Resource

	c := ec2.NewFromConfig(s.AwsConfig)

	p := ec2.NewDescribeVpcsPaginator(c, &ec2.DescribeVpcsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:" + s.Filter.TagKey),
				Values: []string{s.Filter.TagValue},
			},
		},
	})
	for p.HasMorePages() {
		vpcs, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("describing vpcs: %s", err)
		}
		for _, vpc := range vpcs.Vpcs {
			var r Resource
			r.ID = []string{*vpc.VpcId}
			r.Type = ResourceTypeEC2VPC
			r.Tags = map[string]string{}

			results = append(results, r)

			for _, t := range vpc.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}
		}
	}

	return results, nil
}

// Type implements ResourceProvider.
func (e *ec2Vpc) Type() string {
	return ResourceTypeEC2VPC
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &ec2Vpc{}
	})
}
