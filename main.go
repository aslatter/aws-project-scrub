package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/aslatter/aws-project-scrub/internal/resource"
	"github.com/aslatter/aws-project-scrub/internal/schedule"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"golang.org/x/sys/unix"
)

func main() {
	if err := mainErr(); err != nil {
		fmt.Fprintln(os.Stdout, "error:", err)
		os.Exit(1)
	}
}

func mainErr() error {
	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, unix.SIGTERM)
	defer close()

	c, err := getFlags()
	if err != nil {
		return err
	}

	// validate the passed-in account
	ac, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(c.region))
	if err != nil {
		return fmt.Errorf("loading aws config: %s", err)
	}
	stsClient := sts.NewFromConfig(ac)
	ident, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("looking up AWS account: %s", err)
	}
	if ident.Account == nil {
		return errors.New("account id unexpectedly nil")
	}
	if ident.Arn == nil {
		return errors.New("caller ARN unexpectedly nil")
	}
	if c.account != *ident.Account {
		return fmt.Errorf("expected account %q, got %q", c.account, *ident.Account)
	}

	parsedARN, err := arn.Parse(*ident.Arn)
	if err != nil {
		return fmt.Errorf("parsing identity ARN: %s", err)
	}

	var s resource.Settings
	s.AwsConfig = ac
	s.Partition = parsedARN.Partition
	s.Region = c.region
	s.Account = *ident.Account
	s.Filter.TagKey = c.tagKey
	s.Filter.TagValue = c.tagValue

	var rs []resource.ResourceProvider
	for _, p := range resource.GetAllResourceProviders(&s) {
		if g, ok := p.(resource.IsGlobal); ok && g.IsGlobal() {
			if !isGlobalRegion(c.region) {
				continue
			}
		}
		rs = append(rs, p)
	}

	plan := schedule.Plan{
		Providers: rs,
		Settings:  &s,
		Filter: func(r resource.Resource) bool {
			return isResourceOkayToDelete(c, r)
		},
		Action: func(ctx context.Context, p resource.ResourceProvider, r resource.Resource) error {
			if c.dryRun {
				fmt.Println(r)
				return nil
			}
			log.Printf("deleting %s ...", r)
			err := p.DeleteResource(ctx, &s, r)
			if err != nil {
				// keep going for not-found errors
				if resource.IsErrNotFound(err) {
					log.Printf("warning: %q: %s", r, err)
					return nil
				}

				// otherwise stop
				log.Printf("error: %q: %s", r, err)
				return err
			}
			return nil
		},
	}

	return plan.Exec(ctx)
}

func isResourceOkayToDelete(c *cfg, r resource.Resource) bool {
	tv, ok := r.Tags[c.tagKey]
	if !ok {
		return false
	}
	return tv == c.tagValue
}

/**

Global regions:

curl -L "https://raw.githubusercontent.com/boto/botocore/1ad32855c799456250b44c2762cacd67f5647a6e/botocore/data/partitions.json" | \
	jq -r '.partitions[].outputs.implicitGlobalRegion' | \
	xargs -n 1 printf "\tcase \"%s\":\n\t\treturn true\n"

**/

func isGlobalRegion(region string) bool {
	switch region {
	case "us-east-1":
		return true
	case "cn-northwest-1":
		return true
	case "us-gov-west-1":
		return true
	case "us-iso-east-1":
		return true
	case "us-isob-east-1":
		return true
	case "eu-isoe-west-1":
		return true
	case "us-isof-south-1":
		return true
	}
	return false
}
