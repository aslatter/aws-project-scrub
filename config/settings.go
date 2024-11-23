package config

import "github.com/aws/aws-sdk-go-v2/aws"

type Settings struct {
	DryRun    bool
	TagKey    string
	TagValue  string
	AwsConfig aws.Config
}
