package resource

import (
	"context"
	"maps"
	"slices"
	"strings"

	"aws-project-scrub/config"
)

type ResourceProvider interface {
	Type() string
	IsGlobal() bool
	// FindResources discovers "root" resources which must be deleted. returned resources
	// must have the 'Tags' property filled in correctly or the resources
	// will be ignored.
	FindResources(ctx context.Context, s *config.Settings) ([]Resource, error)
	// DependentResources discovers resources which must be deleted prior to deleting a
	// specific resource.
	DependentResources(ctx context.Context, s *config.Settings, r Resource) ([]Resource, error)
	DeleteResource(ctx context.Context, s *config.Settings, r Resource) error
}

type HasDependencies interface {
	Dependencies() []string
}

var registry map[string]ResourceProvider = map[string]ResourceProvider{}

func register(r ResourceProvider) {
	registry[r.Type()] = r
}

func GetAllResourceProviders() []ResourceProvider {
	return slices.Collect(maps.Values(registry))
}

func GetResourceProvider(typ string) (ResourceProvider, bool) {
	rp, ok := registry[typ]
	return rp, ok
}

type Resource struct {
	Type string
	ID   []string
	Tags map[string]string
}

func (r Resource) String() string {
	return r.Type + "/" + strings.Join(r.ID, "/")
}
