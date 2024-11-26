package config

import "github.com/aws/aws-sdk-go-v2/aws"

type Settings struct {
	AwsConfig aws.Config
	Region    string
	Partition string
	Account   string
	Filter    struct {
		TagKey   string
		TagValue string
	}
}
