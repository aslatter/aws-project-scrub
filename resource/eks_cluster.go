package resource

import (
	"aws-project-scrub/config"
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type eksCluster struct{}

// RelatedResources implements ResourceProvider.
func (e *eksCluster) RelatedResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error) {
	// TODO - move fargate profiles and node-groups to external resources
	return nil, nil
}

// DeleteResource implements ResourceProvider.
func (e *eksCluster) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {
	c := eks.NewFromConfig(s.AwsConfig)

	// delete fargate profiles
	fpp := eks.NewListFargateProfilesPaginator(c, &eks.ListFargateProfilesInput{
		ClusterName: &r.ID,
	})
	for fpp.HasMorePages() {
		result, err := fpp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing fargate profiles: %s", err)
		}
		for _, f := range result.FargateProfileNames {
			_, err := c.DeleteFargateProfile(ctx, &eks.DeleteFargateProfileInput{
				ClusterName:        &r.ID,
				FargateProfileName: &f,
			})
			if err != nil {
				return fmt.Errorf("deleting fargate profile %q: %s", f, err)
			}
			waitForFargateDeletion(ctx, c, r.ID, f)
		}
	}

	// delete managed node groups
	ngp := eks.NewListNodegroupsPaginator(c, &eks.ListNodegroupsInput{
		ClusterName: &r.ID,
	})
	for ngp.HasMorePages() {
		result, err := ngp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing managed node groups: %s", err)
		}

		for _, ng := range result.Nodegroups {
			_, err := c.DeleteNodegroup(ctx, &eks.DeleteNodegroupInput{
				ClusterName:   &r.ID,
				NodegroupName: &ng,
			})
			if err != nil {
				return fmt.Errorf("deleting node group %q: %s", ng, err)
			}
		}
	}

	// do delete
	_, err := c.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: &r.ID,
	})
	return err
}

// Dependencies implements ResourceProvider.
func (e *eksCluster) Dependencies() []string {
	return []string{}
}

// FindResources implements ResourceProvider.
func (e *eksCluster) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	var results []Resource

	c := eks.NewFromConfig(s.AwsConfig)
	p := eks.NewListClustersPaginator(c, &eks.ListClustersInput{})
	for p.HasMorePages() {
		result, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing clusters: %s", err)
		}

		for _, k := range result.Clusters {
			var r Resource
			r.Type = e.Type()
			r.ID = k
			r.Tags = map[string]string{}
			results = append(results, r)

			// we need an ARN to look up the tags :-(
			arn := fmt.Sprintf("arn:%s:eks:%s:%s:cluster/%s",
				s.Partition, s.Region, s.Account, k,
			)

			ts, err := c.ListTagsForResource(ctx, &eks.ListTagsForResourceInput{
				ResourceArn: &arn,
			})
			if err != nil {
				return nil, fmt.Errorf("listing tags for EKS cluster %q: %s", k, err)
			}
			maps.Copy(r.Tags, ts.Tags)
		}
	}

	return results, nil
}

// IsGlobal implements ResourceProvider.
func (e *eksCluster) IsGlobal() bool {
	return false
}

// Type implements ResourceProvider.
func (e *eksCluster) Type() string {
	return "AWS::EKS::Cluster"
}

func init() {
	register(&eksCluster{})
}

func waitForFargateDeletion(ctx context.Context, c *eks.Client, cluster string, fargateProfile string) error {
	t := time.NewTicker(time.Second)
	for {
		s, err := c.DescribeFargateProfile(ctx, &eks.DescribeFargateProfileInput{
			ClusterName:        &cluster,
			FargateProfileName: &fargateProfile,
		})

		if err != nil {
			// did we get a 404?
			var rerror *smithyhttp.ResponseError
			if errors.As(err, &rerror) {
				if rerror.Response.StatusCode == 404 {
					// done!
					return nil
				}
			}

			// some other error
			return err
		}

		if s.FargateProfile.Status != types.FargateProfileStatusDeleting {
			return fmt.Errorf("unexpected profile status: %s", s.FargateProfile.Status)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// continue
		}
	}
}
