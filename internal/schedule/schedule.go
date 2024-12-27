package schedule

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/aslatter/aws-project-scrub/internal/resource"

	"github.com/heimdalr/dag"
	"golang.org/x/sync/semaphore"
)

// A Plan schedules the execution of resource-deletion actions. Resource-providers
// are processed in dependency-order while deleting resources in parallel.
type Plan struct {
	Providers []resource.ResourceProvider
	Settings  *resource.Settings
	Filter    func(r resource.Resource) bool
	Action    func(ctx context.Context, p resource.ResourceProvider, r resource.Resource) error

	// hook that any child goroutine can use to wind things down
	abort func(error)

	providers map[string]resource.ResourceProvider
	deps      *dag.DAG
	resources map[string]map[string]resource.Resource

	doneSignal       chan string
	availableWorkers *semaphore.Weighted
}

func (p *Plan) Exec(ctx context.Context) error {
	return (&Plan{
		Providers: p.Providers,
		Settings:  p.Settings,
		Filter:    p.Filter,
		Action:    p.Action,
	}).exec(ctx)
}

func (p *Plan) exec(ctx context.Context) error {

	// we don't use much from this DAG library, but it does tell
	// us up-front if we have dependency cycles.
	p.deps = dag.NewDAG()

	// build up providers and relationships between providers
	p.providers = map[string]resource.ResourceProvider{}
	for _, pr := range p.Providers {
		p.providers[pr.Type()] = pr
		err := p.deps.AddVertexByID(pr.Type(), pr)
		if err != nil {
			return fmt.Errorf("adding provider to dependency graph %q: %s", pr.Type(), err)
		}
	}

	for _, pr := range p.Providers {
		hasDeps, ok := pr.(resource.HasDependencies)
		if !ok {
			continue
		}
		for _, dep := range hasDeps.Dependencies() {
			err := p.deps.AddEdge(dep, pr.Type())
			var isEdgeErr dag.EdgeDuplicateError
			if err != nil && !errors.As(err, &isEdgeErr) {
				return fmt.Errorf("adding dependency on %q from %q: %s", dep, pr.Type(), err)
			}
		}
	}

	// find root resources and dependent resources.
	// (discovering dependent-resources adds edges to our DAG)
	p.resources = map[string]map[string]resource.Resource{}
	for _, pr := range p.Providers {
		finder, ok := pr.(resource.HasRootResources)
		if !ok {
			continue
		}
		rs, err := finder.FindResources(ctx, p.Settings)
		if err != nil {
			return fmt.Errorf("finding root resources for %q: %s", pr.Type(), err)
		}
		for _, r := range rs {
			if !p.Filter(r) {
				continue
			}
			err := p.addOneResource(ctx, r)
			if err != nil {
				return fmt.Errorf("adding resource %q: %s", r, err)
			}
		}
	}

	//
	// prep data-structures for working the plan
	//

	// providers which have completed
	doneProviders := map[string]bool{}

	// providers which have not completed
	pendingProviders := map[string]bool{}

	// signal for done providers
	p.doneSignal = make(chan string, len(p.providers))

	ctx, ctxDone := context.WithCancelCause(ctx)
	p.abort = ctxDone
	defer ctxDone(nil)

	// allow deleting up to 20 resources concurrently. We
	// may have less concurrency than this if dependencies
	// are not met.
	p.availableWorkers = semaphore.NewWeighted(20)

	//
	// start execution
	//

	// move all providers to pending
	for k := range p.providers {
		pendingProviders[k] = true
	}

	// start all providers without dependencies
	for k := range p.deps.GetRoots() {
		delete(pendingProviders, k)
		go p.processOneProvider(ctx, k)
	}

	// queue up providers as they become ready
	for len(doneProviders) < len(p.Providers) {
		select {
		case <-ctx.Done():
			// TODO - capture multiple errors?
			return context.Cause(ctx)

		// a provider finished! Do stuff.
		case doneTyp := <-p.doneSignal:
			doneProviders[doneTyp] = true

		evalLoop:
			for _, evalType := range slices.Collect(maps.Keys(pendingProviders)) {
				ancestors, err := p.deps.GetAncestors(evalType)
				if err != nil {
					// ?!
					return fmt.Errorf("getting ancestors of %q: %s", evalType, err)
				}
				for ancestorTyp := range ancestors {
					if !doneProviders[ancestorTyp] {
						continue evalLoop
					}
				}
				// all dependencies are met!
				delete(pendingProviders, evalType)
				go p.processOneProvider(ctx, evalType)
			}

		}
	}

	return nil
}

// processOneProvider deletes all resources associated with a single
// resource-provider. Once complete it will send it's provider-type
// down the 'doneSignal' channel. Resources will be processed in
// parallel.
func (p *Plan) processOneProvider(ctx context.Context, typ string) {
	defer func() {
		p.doneSignal <- typ
	}()

	pr, ok := p.providers[typ]
	if !ok {
		p.abort(fmt.Errorf("processProvider: unknown type %q", typ))
	}

	var wg sync.WaitGroup
	for r := range maps.Values(p.resources[typ]) {
		err := p.availableWorkers.Acquire(ctx, 1)
		if err != nil {
			// context canceled
			return
		}
		wg.Add(1)
		go func() {
			defer p.availableWorkers.Release(1)
			defer wg.Done()

			err := p.Action(ctx, pr, r)
			if err != nil {
				p.abort(err)
			}
		}()
	}
	wg.Wait()
}

// addOneResource adds a resource to the plan. If the resource has
// dynamically-discovered dependencies, those are recursively added
// as well.
func (p *Plan) addOneResource(ctx context.Context, r resource.Resource) error {
	pr, ok := p.providers[r.Type]
	if !ok {
		return fmt.Errorf("unknown provider-id for resource %q: %s", r, r.Type)
	}

	typMap, ok := p.resources[r.Type]
	if !ok {
		typMap = map[string]resource.Resource{}
		p.resources[r.Type] = typMap
	}
	idStr := strings.Join(r.ID, "/")
	if _, ok := typMap[idStr]; ok {
		// done!
		return nil
	}
	typMap[idStr] = r

	// find more resources
	depProvider, ok := pr.(resource.HasDependentResources)
	if !ok {
		return nil
	}
	moreResources, err := depProvider.DependentResources(ctx, p.Settings, r)
	if err != nil {
		return fmt.Errorf("looking up dependent resources for %q: %s", r, err)
	}
	for _, nextResource := range moreResources {
		err := p.addOneResource(ctx, nextResource)
		if err != nil {
			return fmt.Errorf("adding dependent resource %q: %s", nextResource, err)
		}

		var isEdgeErr dag.EdgeDuplicateError
		err = p.deps.AddEdge(nextResource.Type, r.Type)
		if err != nil && !errors.As(err, &isEdgeErr) {
			return fmt.Errorf("adding dependency on %q from %q: %s", nextResource.Type, r.Type, err)
		}
	}

	return nil
}
