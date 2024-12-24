package resource

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

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

type ResourceProvider interface {
	Type() string
	DeleteResource(ctx context.Context, s *Settings, r Resource) error
}

type HasRootResources interface {
	// FindResources discovers "root" resources which must be deleted. returned resources
	// must have the 'Tags' property filled in correctly or the resources
	// will be ignored.
	FindResources(ctx context.Context, s *Settings) ([]Resource, error)
}

type HasDependentResources interface {
	// DependentResources discovers resources which must be deleted prior to deleting a
	// specific resource.
	DependentResources(ctx context.Context, s *Settings, r Resource) ([]Resource, error)
}

type HasDependencies interface {
	Dependencies() []string
}

type IsGlobal interface {
	IsGlobal() bool
}

var registry [](func(*Settings) ResourceProvider) = [](func(*Settings) ResourceProvider){}

func register(fn func(*Settings) ResourceProvider) {
	registry = append(registry, fn)
}

func GetAllResourceProviders(s *Settings) []ResourceProvider {
	var result []ResourceProvider
	for _, fn := range registry {
		result = append(result, fn(s))
	}
	return result
}

type Resource struct {
	Type string
	ID   []string
	Tags map[string]string
}

func (r Resource) String() string {
	return r.Type + "/" + strings.Join(r.ID, "/")
}

const defaultDeleteWaitTime = 5 * time.Minute
