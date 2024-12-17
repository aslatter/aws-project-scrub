# cleaning up resources

New resources may be added by:

* Creating a new file in this package named after the resource
* Creating a new struct-type to represent the new resource-provider
  (aka thing which can take actions on a new resource-type)
* Add:

  ```go
  func init() {
	register(func(s *config.Settings) ResourceProvider {
		return &provider{}
	})
}
  ```

  To the new file. Implement the required methods.

# Resource Discovery

There are two kinds of resources:

* Root resources
* Dependent resources

Root-resources are supported by a resource-provider if the provider
has a method `FindResources(context.Context, *config.Settings) ([]Resource, error)`.
The returned resources must include any tags, and returned resources
will only be deleted if they match tag-filters passed in to the program.

A resource-provider does not need to enforce the tag-filer, but the filter
is available in the passed-in settings. This can be used if the API used to
search for resources supports natively filtering by tags.

Dependent resources are supported by a provider if the provider has a
method `DependentResources(context.Context, *config.Settings, Resource) ([]Resource, error)`.

Dependent resources are resources which must be deleted prior to the
passed-in resource itself being delete.

Resource-dependencies will be evaluated for every resource
of the same type as the provider. Returned dependencies are expected
to be of a different type. Returned dependencies are cleaned
up before the evaluated resource.

For new resources:

* Make the resource a dependency of another if the lifetimes
  are tied together. For example, an EC2 instance cannot exist
  without a VPC, and the VPC cannot be cleaned up until the instance
  is deleted, so the instances are a dependency of the VPC.
* Make the resource a root-resource if it has an independent
  lifetime. For example, an EC2 Volume can persist beyond the
  lifetime of the instance it was associated with, so it is its
  own "root" resource with a `FindResources` method.

Root resources must be tagged during creation to be discovered
(and cleaned up).

# Cleanup ordering

Dependent resources found via `DependentResources` will be cleaned up
before the resource they are a dependency of.

A resource-provider as a whole can declare a manual dependency on
another provider by implementing the method `Dependencies() []string`.
This method returns a slice of resource-type-identifiers. All resources
of the indicated types will be cleaned up prior to any resources of the
current provider.

This is generally used to schedule "root resources" relative to each
other. For example, we want to make sure we're done using any IP addresses
before we release them, so we make sure we've cleaned up out VPCs before
we start releasing addresses:

```go
// Dependencies returns resource-providers which must run before this one.
func (e *ec2EIP) Dependencies() []string {
	return []string{ResourceTypeEC2VPC}
}
```
