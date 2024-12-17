# aws-project-scrub

This project cleans up AWS resources related to a project.

This is for demonstration use only, and even if you use it for real it should
only be used in a sandbox/experimental AWS account.

It is inspired by and similar to `aws-nuke` (https://github.com/ekristen/aws-nuke),
except `aws-project-scrub` finds resources to delete based on input tags (and is wildly
incomplete). `aws-nuke` can use tags as a filter, but it is opt-in and not supported by
all resources.

`aws-project-scrub` has two phases of resource-discovery:

* 'root' resources, based on input tags
* 'child' resources, based on relationships to root resources

For example, we discover VPCs to delete based on the tags passed in,
but we then proceed to delete all subnets and EC2 instances discovered by
searching for things related to the VPC.
