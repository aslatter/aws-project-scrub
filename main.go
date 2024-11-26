package main

import (
	"aws-project-scrub/config"
	"aws-project-scrub/resource"
	"context"
	"errors"
	"fmt"
	"log"
	"maps"
	"os"
	"os/signal"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
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

	var s config.Settings
	s.AwsConfig = ac
	s.Partition = parsedARN.Partition
	s.Region = c.region
	s.Account = *ident.Account
	s.Filter.TagKey = c.tagKey
	s.Filter.TagValue = c.tagValue

	rs, err := getOrderedResources(ctx, c, &s)
	if err != nil {
		return err
	}

	for _, b := range rs {
		r := b.provider
		for _, res := range b.resources {
			if c.dryRun {
				fmt.Println(res)
			} else {
				log.Printf("deleting %s ...", res)
				err := r.DeleteResource(ctx, &s, res)
				if err != nil {
					log.Printf("error: %q: %s", res, err)

					// stop everything if the user canceled
					if ctx.Err() != nil {
						return ctx.Err()
					}
				}
			}

		}
	}
	return nil
}

type resourceBundle struct {
	provider  resource.ResourceProvider
	resources []resource.Resource
}

func getOrderedResources(ctx context.Context, c *cfg, s *config.Settings) ([]resourceBundle, error) {
	cr, err := collectResources(ctx, c, s)
	if err != nil {
		return nil, err
	}

	// build a dag of implied-dependencies (based on related-resources
	// returned)

	d := dag.NewDAG()

	rs := resource.GetAllResourceProviders(s)
	for _, r := range rs {
		err := d.AddVertexByID(r.Type(), r)
		if err != nil {
			// TODO
			return nil, err
		}
	}

	// implied dependencies
	for fromType, v := range cr.impliedDeps {
		for toType := range v {
			err := d.AddEdge(toType, fromType)
			if err != nil {
				return nil, fmt.Errorf("adding dependency from %s to %s: %s", fromType, toType, err)
			}
		}
	}

	// explicit dependencies
	for _, r := range rs {
		if rd, ok := r.(resource.HasDependencies); ok {
			for _, dep := range rd.Dependencies() {
				err := d.AddEdge(dep, r.Type())
				if err != nil {
					return nil, fmt.Errorf("adding explicit dependency from %s to %s: %s", r.Type(), dep, err)
				}
			}
		}
	}

	var results []resourceBundle
	for _, resourceID := range dagDependencyOrder(d) {
		v, err := d.GetVertex(resourceID)
		if err != nil {
			return nil, fmt.Errorf("looking up graph vertex: %s", err)
		}
		rp, ok := v.(resource.ResourceProvider)
		if !ok {
			return nil, errors.New("graph vertex was not a resource provider")
		}
		var b resourceBundle
		b.provider = rp
		b.resources = append(b.resources, cr.resources[resourceID]...)
		results = append(results, b)
	}

	return results, nil
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

func collectResources(ctx context.Context, c *cfg, s *config.Settings) (*collectedResources, error) {
	var b resourceBag
	var result collectedResources

	// ask each provider for all the resources, filter them, then ask for related resources

	// most resource-providers don't actually provide resources - we find them from tagged
	// resource-roots

	b.providers = map[string]resource.ResourceProvider{}
	rps := resource.GetAllResourceProviders(s)
	for _, rp := range rps {
		b.providers[rp.Type()] = rp
	}
	for _, rp := range rps {
		rootProvider, ok := rp.(resource.HasRootResources)
		if !ok {
			continue
		}
		if rg, ok := rp.(resource.IsGlobal); ok && rg.IsGlobal() && !isGlobalRegion(s.Region) {
			continue
		}
		rs, err := rootProvider.FindResources(ctx, s)
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			if !isResourceOkayToDelete(c, r) {
				continue
			}
			deps, err := b.addResource(ctx, s, r)
			if err != nil {
				return nil, err
			}
			result.impliedDeps.copy(deps)
		}
	}

	// get all resources from resource-bag
	result.resources = map[string][]resource.Resource{}
	for k, v := range b.foundResources {
		var rs []resource.Resource
		for _, r := range v {
			rs = append(rs, r)
		}
		result.resources[k] = rs
	}

	return &result, nil
}

type collectedResources struct {
	resources   map[string][]resource.Resource
	impliedDeps dependencies
}

type resourceBag struct {
	providers      map[string]resource.ResourceProvider
	foundResources map[string]map[string]resource.Resource
}

type dependencies map[string]map[string]struct{}

func (d *dependencies) add(from string, to string) {
	if *d == nil {
		(*d) = map[string]map[string]struct{}{}
	}
	v, ok := (*d)[from]
	if !ok {
		v = map[string]struct{}{}
		(*d)[from] = v
	}
	v[to] = struct{}{}
}

func (d *dependencies) copy(other dependencies) {
	if len(other) == 0 {
		return
	}
	if *d == nil {
		(*d) = map[string]map[string]struct{}{}
	}
	for k, ov := range other {
		v, ok := (*d)[k]
		if !ok {
			v = map[string]struct{}{}
			(*d)[k] = v
		}
		maps.Copy(v, ov)
	}
}

func (rb *resourceBag) addResource(ctx context.Context, s *config.Settings, r resource.Resource) (dependencies, error) {
	if rb.foundResources == nil {
		rb.foundResources = map[string]map[string]resource.Resource{}
	}

	resourceKey := strings.Join(r.ID, "/")

	foundByType, ok := rb.foundResources[r.Type]
	if !ok {
		foundByType = map[string]resource.Resource{}
		rb.foundResources[r.Type] = foundByType
	}
	_, exist := foundByType[resourceKey]
	if exist {
		return nil, nil
	}
	foundByType[resourceKey] = r

	rp, ok := rb.providers[r.Type]
	if !ok {
		return nil, fmt.Errorf("unknown resource type %q", r.Type)
	}

	dependencyProvider, ok := rp.(resource.HasDependentResources)
	if !ok {
		return nil, nil
	}

	related, err := dependencyProvider.DependentResources(ctx, s, r)
	if err != nil {
		return nil, fmt.Errorf("finding resources related to %s", r)
	}

	var foundDeps dependencies

	for _, rr := range related {
		foundDeps.add(r.Type, rr.Type)
		d, err := rb.addResource(ctx, s, rr)
		if err != nil {
			return nil, err
		}
		foundDeps.copy(d)
	}
	return foundDeps, nil
}

// dagDependencyOrder returns topologically sorted
// vertex-ids in the DAG.
func dagDependencyOrder(d *dag.DAG) []string {
	var results []string
	d.OrderedWalk(dagVisitor(func(v dag.Vertexer) {
		id, _ := v.Vertex()
		results = append(results, id)
	}))
	return results
}

type dagVisitor func(dag.Vertexer)

// Visit implements dag.Visitor.
func (d dagVisitor) Visit(v dag.Vertexer) {
	d(v)
}
