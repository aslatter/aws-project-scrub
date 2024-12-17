package resource

import (
	"context"
	"fmt"
	"strings"

	"github.com/aslatter/aws-project-scrub/internal/config"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

type hostedZone struct{}

// DeleteResource implements ResourceProvider.
func (h *hostedZone) DeleteResource(ctx context.Context, s *config.Settings, r Resource) error {

	zid := r.ID[0]
	zname := r.ID[1]

	c := route53.NewFromConfig(s.AwsConfig)

	// delete all record-sets
	rp := route53.NewListResourceRecordSetsPaginator(c, &route53.ListResourceRecordSetsInput{
		HostedZoneId: &zid,
	})

	w := route53.NewResourceRecordSetsChangedWaiter(c)

	var changes []types.Change
	for rp.HasMorePages() {
		result, err := rp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing record-sets: %w", err)
		}

		for _, rr := range result.ResourceRecordSets {
			// cannot remove root NS record or SOA record
			if rr.Type == types.RRTypeSoa {
				continue
			}
			if rr.Type == types.RRTypeNs && *rr.Name == zname {
				continue
			}

			changes = append(changes, types.Change{
				Action: types.ChangeActionDelete,
				ResourceRecordSet: &types.ResourceRecordSet{
					Name:            rr.Name,
					Type:            rr.Type,
					TTL:             rr.TTL,
					ResourceRecords: rr.ResourceRecords,
					AliasTarget:     rr.AliasTarget,
				},
			})

			if len(changes) == 1000 {
				changeResult, err := c.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
					HostedZoneId: &zid,
					ChangeBatch: &types.ChangeBatch{
						Changes: changes,
					},
				})
				if err != nil {
					return fmt.Errorf("updating record-sets: %s", err)
				}
				changes = changes[:0]

				err = w.Wait(ctx, &route53.GetChangeInput{
					Id: changeResult.ChangeInfo.Id,
				}, defaultDeleteWaitTime)
				if err != nil {
					return fmt.Errorf("waiting for changeset: %s", err)
				}
			}
		}
	}
	if len(changes) > 0 {
		changeResult, err := c.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: &zid,
			ChangeBatch: &types.ChangeBatch{
				Changes: changes,
			},
		})
		if err != nil {
			return fmt.Errorf("updating record-sets: %s", err)
		}

		err = w.Wait(ctx, &route53.GetChangeInput{
			Id: changeResult.ChangeInfo.Id,
		}, defaultDeleteWaitTime)
		if err != nil {
			return fmt.Errorf("waiting for changeset: %s", err)
		}
	}

	_, err := c.DeleteHostedZone(ctx, &route53.DeleteHostedZoneInput{
		Id: &zid,
	})
	return err
}

// FindResources implements ResourceProvider.
func (h *hostedZone) FindResources(ctx context.Context, s *config.Settings) ([]Resource, error) {
	c := route53.NewFromConfig(s.AwsConfig)

	var foundZones []Resource

	zp := route53.NewListHostedZonesPaginator(c, &route53.ListHostedZonesInput{})
	for zp.HasMorePages() {
		result, err := zp.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing hosted zones: %s", err)
		}
		for _, z := range result.HostedZones {
			if z.Id == nil || z.Name == nil {
				continue
			}

			var r Resource
			r.Type = h.Type()
			id := *z.Id
			// why do I need to do this?!
			id = strings.ReplaceAll(id, "/hostedzone/", "")
			r.ID = []string{id, *z.Name}
			r.Tags = map[string]string{}

			ts, err := c.ListTagsForResource(ctx, &route53.ListTagsForResourceInput{
				ResourceId:   &id,
				ResourceType: types.TagResourceTypeHostedzone,
			})
			if err != nil {
				return nil, fmt.Errorf("listing tags for zone %s: %s", id, err)
			}
			for _, t := range ts.ResourceTagSet.Tags {
				if t.Key == nil || t.Value == nil {
					continue
				}
				r.Tags[*t.Key] = *t.Value
			}

			foundZones = append(foundZones, r)
		}
	}

	return foundZones, nil
}

// IsGlobal implements ResourceProvider.
func (h *hostedZone) IsGlobal() bool {
	return true
}

// Type implements ResourceProvider.
func (h *hostedZone) Type() string {
	return "AWS::Route53::HostedZone"
}

func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &hostedZone{}
	})
}
