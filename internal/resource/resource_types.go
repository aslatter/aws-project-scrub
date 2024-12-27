package resource

// https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-template-resource-type-ref.html
const (
	ResourceTypeEC2EIP                       = "AWS::EC2::EIP"
	ResourceTypeEC2Instance                  = "AWS::EC2::Instance"
	ResourceTypeEC2NATGateway                = "AWS::EC2::NatGateway"
	ResourceTypeEC2Volume                    = "AWS::EC2::Volume"
	ResourceTypeEC2VPC                       = "AWS::EC2::VPC"
	ResourceTypeEC2Subnet                    = "AWS::EC2::Subnet"
	ResourceTypeEC2SecurityGroup             = "AWS::EC2::SecurityGroup"
	ResourceTypeEC2SecurityGroupRule         = "AWS::EC2::SecurityGroupRule" // not a real cfn type
	ResourceTypeEC2LaunchTemplate            = "AWS::EC2::LaunchTemplate"
	ResourceTypeEC2NetworkACL                = "AWS::EC2::NetworkAcl"
	ResourceTypeEC2RouteTable                = "AWS::EC2::RouteTable"
	ResourceTypeEC2InternetGateway           = "AWS::EC2::InternetGateway"
	ResourceTypeEC2EgressOnlyInternetGateway = "AWS::EC2::EgressOnlyInternetGateway"
	ResourceTypeEC2VPCEndpoint               = "AWS::EC2::VPCEndpoint"
	ResourceTypeLoadBalancer                 = "AWS::ElasticLoadBalancingV2::LoadBalancer"
	ResourceTypeLoadBalancerTargetGroup      = "AWS::ElasticLoadBalancingV2::TargetGroup"
	ResourceTypeEKSCluster                   = "AWS::EKS::Cluster"
	ResourceTypeEKSFargateProfile            = "AWS::EKS::FargateProfile"
	ResourceTypeEKSNodegroup                 = "AWS::EKS::Nodegroup"
	ResourceTypeEKSPodIdentityAssociation    = "AWS::EKS::PodIdentityAssociation"
	ResourceTypeEventsRule                   = "AWS::Events::Rule"
	ResourceTypeIAMPolicy                    = "AWS::IAM::Policy"
	ResourceTypeIAMRole                      = "AWS::IAM::Role"
	ResourceTypeIAMInstanceProfile           = "AWS::IAM::InstanceProfile"
	ResourceTypeSQSQueue                     = "AWS::SQS::Queue"
	ResourceTypeLogsLogGroup                 = "AWS::Logs::LogGroup"
)
