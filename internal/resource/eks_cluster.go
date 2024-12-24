package resource

import (
	"context"
	"fmt"
	"maps"

	"github.com/aws/aws-sdk-go-v2/service/eks"
)

type eksCluster struct{}

// DependentResources implements ResourceProvider.
func (e *eksCluster) DependentResources(ctx context.Context, s *Settings, r Resource) ([]Resource, error) {
	c := eks.NewFromConfig(s.AwsConfig)
	cluster := r.ID[0]

	var result []Resource

	pp := eks.NewListFargateProfilesPaginator(c, &eks.ListFargateProfilesInput{
		ClusterName: &cluster,
	})
	for pp.HasMorePages() {
		page, err := pp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing EKS fargate profiles: %s", err)
		}
		for _, p := range page.FargateProfileNames {
			var r Resource
			r.ID = []string{cluster, p}
			r.Type = ResourceTypeEKSFargateProfile
			result = append(result, r)
		}
	}

	ngp := eks.NewListNodegroupsPaginator(c, &eks.ListNodegroupsInput{
		ClusterName: &cluster,
	})
	for ngp.HasMorePages() {
		page, err := ngp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing EKS node groups: %s", err)
		}
		for _, ng := range page.Nodegroups {
			var r Resource
			r.ID = []string{cluster, ng}
			r.Type = ResourceTypeEKSNodegroup
			result = append(result, r)
		}
	}

	pip := eks.NewListPodIdentityAssociationsPaginator(c, &eks.ListPodIdentityAssociationsInput{
		ClusterName: &cluster,
	})
	for pip.HasMorePages() {
		pias, err := pip.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing pod-identity associations: %s", err)
		}
		for _, pia := range pias.Associations {
			var r Resource
			r.Type = ResourceTypeEKSPodIdentityAssociation
			r.ID = []string{cluster, *pia.AssociationId}
			result = append(result, r)
		}
	}

	return result, nil
}

// DeleteResource implements ResourceProvider.
func (e *eksCluster) DeleteResource(ctx context.Context, s *Settings, r Resource) error {
	c := eks.NewFromConfig(s.AwsConfig)
	_, err := c.DeleteCluster(ctx, &eks.DeleteClusterInput{
		Name: &r.ID[0],
	})
	if err != nil {
		return err
	}

	w := eks.NewClusterDeletedWaiter(c)
	err = w.Wait(ctx, &eks.DescribeClusterInput{
		Name: &r.ID[0],
	}, defaultDeleteWaitTime)
	if err != nil {
		return fmt.Errorf("waiting for deletion: %s", err)
	}

	return nil
}

// FindResources implements ResourceProvider.
func (e *eksCluster) FindResources(ctx context.Context, s *Settings) ([]Resource, error) {
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
			r.ID = []string{k}
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

// Type implements ResourceProvider.
func (e *eksCluster) Type() string {
	return ResourceTypeEKSCluster
}

func init() {
	register(func(s *Settings) ResourceProvider {
		return &eksCluster{}
	})
}
