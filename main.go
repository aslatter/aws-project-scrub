package main

import (
	"aws-project-scrub/config"
	"aws-project-scrub/resource"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/heimdalr/dag"
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
	if c.account != *ident.Account {
		return fmt.Errorf("expected account %q, got %q", c.account, *ident.Account)
	}

	rs, err := getOrderedResources(c)
	if err != nil {
		return err
	}

	for _, r := range rs {
		resources, err := r.FindResources(ctx, &config.Settings{
			AwsConfig: ac,
		})
		if err != nil {
			return fmt.Errorf("finding resources %s: %s", r.Type(), err)
		}
		for _, res := range resources {
			if isResourceOkayToDelete(c, res) {
				if c.dryRun {
					fmt.Println(r.Type() + " " + res.ID)
				} else {
					return errors.New("unimplemented")
				}
			}
		}
	}
	return nil
}

func getOrderedResources(c *cfg) ([]resource.ResourceProvider, error) {

	allowGlobal := isGlobalRegion(c.region)
	var rs []resource.ResourceProvider
	for _, r := range resource.GetAllResourceProviders() {
		if r.IsGlobal() && !allowGlobal {
			continue
		}
		rs = append(rs, r)
	}

	d := dag.NewDAG()

	for _, r := range rs {
		err := d.AddVertexByID(r.Type(), r)
		if err != nil {
			// TODO
			return nil, err
		}
	}
	for _, r := range rs {
		for _, dep := range r.Dependencies() {
			err := d.AddEdge(dep, r.Type())
			if err != nil {
				return nil, fmt.Errorf("adding dependency from %s to %s: %s", r.Type(), dep, err)
			}
		}
	}

	var results []resource.ResourceProvider

	d.BFSWalk(visitorFunc(func(v dag.Vertexer) {
		_, r := v.Vertex()
		if rr, ok := r.(resource.ResourceProvider); ok {
			results = append(results, rr)
		}
	}))

	return results, nil
}

type visitorFunc func(dag.Vertexer)

// Visit implements dag.Visitor.
func (v visitorFunc) Visit(vx dag.Vertexer) {
	v(vx)
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

curl -L https://github.com/aws/aws-sdk-ruby/raw/refs/heads/version-3/gems/aws-partitions/partitions.json | \
	jq -r '.partitions | map (.services.iam.endpoints | select(. != null)) | map(to_entries[0].value.credentialScope.region) | .[]'

**/

func isGlobalRegion(region string) bool {
	switch region {
	case "us-east-1", "cn-north-1", "us-gov-west-1", "us-iso-east-1", "us-isob-east-1":
		return true
	}
	return false
}
