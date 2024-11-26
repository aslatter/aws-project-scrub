package resource

// https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html
const (
	ResourceTypeEC2EIP                  = "AWS::EC2::EIP"
	ResourceTypeEC2Instance             = "AWS::EC2::Instance"
	ResourceTypeEC2NATGateway           = "AWS::EC2::NatGateway"
	ResourceTypeEC2VPC                  = "AWS::EC2::VPC"
	ResourceTypeLoadBalancer            = "AWS::ElasticLoadBalancingV2::LoadBalancer"
	ResourceTypeLoadBalancerTargetGroup = "AWS::ElasticLoadBalancingV2::TargetGroup"
	ResourceTypeEKSCluster              = "AWS::EKS::Cluster"
	ResourceTypeEKSFargateProfile       = "AWS::EKS::FargateProfile"
	ResourceTypeEKSNodegroup            = "AWS::EKS::Nodegroup"
	ResourceTypeIAMRole                 = "AWS::IAM::Role"
	ResourceTypeIAMInstanceProfile      = "AWS::IAM::InstanceProfile"
)
