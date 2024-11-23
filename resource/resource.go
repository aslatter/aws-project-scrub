package resource

import (
	"context"
	"maps"
	"slices"

	"aws-project-scrub/config"
)

type ResourceProvider interface {
	Type() string
	IsGlobal() bool
	Dependencies() []string
	FindResources(ctx context.Context, s *config.Settings) ([]Resource, error)
	DeleteResource(ctx context.Context, s *config.Settings, r Resource) error
}

var registry map[string]ResourceProvider = map[string]ResourceProvider{}

func register(r ResourceProvider) {
	registry[r.Type()] = r
}

func GetAllResourceProviders() []ResourceProvider {
	return slices.Collect(maps.Values(registry))
}

func GetResourceProvider(typ string) ResourceProvider {
	return registry[typ]
}

type Resource struct {
	ID   string
	Tags map[string]string
}
