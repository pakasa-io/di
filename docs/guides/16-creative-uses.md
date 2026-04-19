# Creative Uses

These patterns are not the only way to use `di`, but they show how the library can support more than plain service wiring.

## 1. CLI Command Bus Without A Framework Container

Use `InvokeOn` to turn plain functions into dependency-aware commands.

```go
type Audit struct{}
type Pruner struct{}

func runPrune(audit *Audit, pruner *Pruner) error {
	_ = audit
	_ = pruner
	return nil
}

container := di.NewContainer()
di.MustProvideTo[*Audit](container, func() *Audit { return &Audit{} })
di.MustProvideTo[*Pruner](container, func() *Pruner { return &Pruner{} })

if err := di.InvokeOn(container, runPrune); err != nil {
	panic(err)
}
```

Why it is useful:

- no extra command struct is required
- command handlers remain plain Go functions
- startup wiring still benefits from validation

## 2. Startup Pipelines With Groups

Use groups as a modular startup pipeline.

```go
type StartupStep interface {
	Run() error
}

type CacheWarmup struct{}
type IndexSync struct{}

func (*CacheWarmup) Run() error { return nil }
func (*IndexSync) Run() error   { return nil }

container := di.NewContainer()

di.MustProvideGroupTo[*CacheWarmup](container, "startup", func() *CacheWarmup {
	return &CacheWarmup{}
}, di.WithLifetime(di.LifetimeTransient))

di.MustProvideGroupTo[*IndexSync](container, "startup", func() *IndexSync {
	return &IndexSync{}
}, di.WithLifetime(di.LifetimeTransient))

steps := di.MustResolveGroupFrom[StartupStep](container, "startup")
for _, step := range steps {
	if err := step.Run(); err != nil {
		panic(err)
	}
}
```

This is a good pattern when teams or packages contribute their own startup behavior independently.

## 3. Dependency Bundles For Wide Constructors

Use auto-wired structs as dependency bundles when factories start to sprawl.

```go
type DB struct{}
type Logger struct{}
type Cache struct{}

type ServiceDeps struct {
	DB     *DB
	Logger *Logger
	Cache  *Cache
}

type Service struct {
	Deps ServiceDeps
}

container := di.NewContainer()
container.SetStructAutoWiring(true)

di.MustProvideTo[*DB](container, func() *DB { return &DB{} })
di.MustProvideTo[*Logger](container, func() *Logger { return &Logger{} })
di.MustProvideTo[*Cache](container, func() *Cache { return &Cache{} })
di.MustProvideTo[*Service](container, func(deps ServiceDeps) *Service {
	return &Service{Deps: deps}
})
```

This is useful when you want factories to stay readable without turning every collaborator into a long parameter list.

## 4. Tenant-Specific Feature Packs With Overlays

Use one base container for shared infrastructure and one overlay per tenant or customer segment.

```go
type SharedDB struct{}
type Branding struct {
	Theme string
}

base := di.NewContainer()
di.MustProvideTo[*SharedDB](base, func() *SharedDB { return &SharedDB{} })

enterprise := di.MustNewOverlayContainer(base)
di.MustProvideTo[*Branding](enterprise, func() *Branding {
	return &Branding{Theme: "enterprise"}
})

selfServe := di.MustNewOverlayContainer(base)
di.MustProvideTo[*Branding](selfServe, func() *Branding {
	return &Branding{Theme: "self-serve"}
})
```

This pattern works especially well when 90% of the graph is shared and only a few surface behaviors vary.

## 5. Canary And Admin Swaps With Overrides

Overrides let you swap a dependency at runtime without changing registration code.

```go
type SearchClient struct {
	Backend string
}

container := di.NewContainer()
di.MustProvideTo[*SearchClient](container, func() *SearchClient {
	return &SearchClient{Backend: "stable"}
})

restore := di.MustOverrideInContainer[*SearchClient](container, func() (*SearchClient, error) {
	return &SearchClient{Backend: "canary"}, nil
})
defer restore()
```

This can support:

- admin toggles
- canary experiments
- fault injection during tests or rehearsals

Use carefully. For permanent product modes, explicit graph design is clearer.

## 6. Self-Diagnosing Services

Use diagnostics and metrics to expose an internal graph report.

```go
import didiag "github.com/pakasa-io/di/diagnostics"

func debugSummary(container *di.Container) map[string]any {
	graph, _ := didiag.DumpGraph(container)
	metrics := container.Metrics()

	return map[string]any{
		"graph":   graph,
		"metrics": metrics,
	}
}
```

This is useful for:

- `/debug/di` style endpoints
- startup reports
- operational dashboards

## 7. Job-Scoped Units Of Work

Not every scope is an HTTP request. Scopes also work for background jobs, ingest flows, and sync loops.

```go
type JobState struct {
	JobID string
}

type Worker struct {
	State *JobState
}

container := di.NewContainer()

di.MustProvideTo[*JobState](container, func(ctx context.Context) *JobState {
	id, _ := ctx.Value("job_id").(string)
	return &JobState{JobID: id}
}, di.WithLifetime(di.LifetimeScoped))

di.MustProvideTo[*Worker](container, func(state *JobState) *Worker {
	return &Worker{State: state}
}, di.WithLifetime(di.LifetimeScoped))

scope := container.MustNewScope()
defer scope.Close()

ctx := context.WithValue(context.Background(), "job_id", "job-123")
worker := di.MustResolveInScopeContext[*Worker](ctx, scope)
_ = worker
```

This pattern is often cleaner than manually threading job-local caches and state through multiple layers.
