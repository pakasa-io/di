# Best Practices And Gotchas

This guide is the production-oriented companion to the API guides. It focuses on decisions that keep graphs understandable, safe, and easy to debug.

## Best Practices

### 1. Validate The Graph At Startup

Run validation after registration and before the app begins serving traffic.

```go
import didiag "github.com/pakasa-io/di/diagnostics"

if err := didiag.Validate(container); err != nil {
	panic(didiag.FormatValidation(err))
}
```

Why:

- catches missing dependencies early
- catches invalid singleton-to-scoped edges
- avoids request-time surprises in large graphs

### 2. Default To Explicit Containers

Prefer `di.NewContainer()` for most library, service, and test code. Reach for `global` only when a process-wide container model is genuinely simpler.

Why:

- ownership is obvious
- tests stay isolated
- graph construction is easier to reason about

### 3. Keep Factories Small And Deterministic

Factories should mostly assemble dependencies. Avoid hiding broad side effects inside registration.

Good:

- build one dependency from other dependencies
- perform local validation
- return an error when setup is invalid

Avoid:

- mutating global state
- performing unrelated startup workflows
- mixing graph assembly with application orchestration

### 4. Use Constructor Injection For Core Domain Types

Prefer factories and typed parameters for the core graph. Use `FillStruct` mainly for adapters or framework-owned values.

Why:

- constructor signatures remain explicit
- hidden field injection is reduced
- testing and refactoring stay easier

### 5. Choose Lifetimes Intentionally

- `singleton` for shared clients, immutable services, pools, caches, and config
- `transient` for cheap disposable wrappers
- `scoped` for request/job-local state and unit-of-work style data

If you are unsure, start with singleton or scoped. Overusing transient usually adds churn without real benefit.

### 6. Use One Scope Per Request Or Job

Create a scope at the boundary of one logical operation, then resolve all scoped values through it.

```go
scope := container.MustNewScope()
defer scope.Close()
```

This is a good fit for:

- HTTP requests
- queue jobs
- scheduled task executions
- import/export pipelines

### 7. Use Named Bindings Sparingly And Deliberately

Named bindings are powerful, but names can become a second type system if overused.

Good uses:

- `primary` and `replica`
- `public` and `internal`
- tenant or environment variants

Avoid:

- arbitrary or opaque names
- using names where an interface alias or separate type would be clearer

### 8. Prefer Groups For Pipelines And Collections

Groups work well for:

- startup hooks
- middleware lists
- plugin batches
- validators or enrichers

Use interface aliases when you need typed discovery of implementations across the graph. Use groups when the real need is “give me all items in this collection.”

### 9. Use Overrides For Temporary Replacement, Not Structural Design

Overrides are best for:

- tests
- short-lived experiments
- canary or admin-driven swaps

If the replacement should exist permanently, use an explicit binding or an overlay container instead.

### 10. Keep Instrumentation Lightweight

Instrumentation callbacks run inline. Keep them fast and side-effect-light.

Good:

- counters
- simple logs
- timing metrics

Avoid:

- blocking I/O
- large allocations on every callback
- container mutation inside callbacks

## Gotchas

### Singletons Cannot Depend On Scoped Values Or `context.Context`

This is a real graph error, not just a style issue. Validation reports it.

Bad pattern:

```go
di.MustProvideTo[*RequestState](container, func(ctx context.Context) *RequestState {
	return &RequestState{}
}, di.WithLifetime(di.LifetimeScoped))

di.MustProvideTo[*App](container, func(state *RequestState) *App {
	return &App{State: state}
})
```

Fix it by making the consumer scoped or by removing request-specific data from the singleton graph.

### Named Global Containers Are Independent

`global.Default()` and `global.Named("jobs")` do not inherit from each other.

If you need inherited registrations, use explicit containers plus overlays.

### Auto-Wiring Only Applies To Unnamed Struct Values

These work:

- `func(deps RepoDeps) *Repo`

These do not auto-wire:

- `func(deps *RepoDeps) *Repo`
- `func(deps RepoDeps) *Repo` together with a named dependency request for the bundle itself

### `Optional[T]` Is Not A Registration Type

These are invalid:

- registering `Optional[T]`
- overriding `Optional[T]`
- resolving aggregate APIs with `Optional[T]`

Register or override `T`, then consume it as `Optional[T]`.

### `ResolveImplementations` Requires An Interface Type

This is for interface aliases, not “all concrete bindings assignable to some type-shaped idea.”

Good:

```go
type Logger interface {
	Log(string)
}

values, err := di.ResolveImplementationsFrom[Logger](container)
_, _ = values, err
```

Bad:

```go
values, err := di.ResolveImplementationsFrom[*JSONLogger](container)
_, _ = values, err
```

### Overrides Run On Each Resolve

Override factories are runtime replacements, not lifetime-managed bindings. If you want a stable fake, close over one instance yourself.

```go
fake := &Clock{}

restore := di.MustOverrideInContainer[*Clock](container, func() (*Clock, error) {
	return fake, nil
})
defer restore()
```

### `ListBindings` Is Local To One Container

On overlay containers, `ListBindings()` shows locally registered bindings only. Use `Explain`, `Graph`, or `DescribeBinding` to inspect the effective inherited graph.

### `FillStruct` Only Sets Exported Fields

If a field is unexported, skipped with `di:"-"`, or not addressable on the provided value, it will not be injected.

### Closing A Parent Container Closes Overlay Children

This is usually correct behavior, but it matters if child overlays are long-lived.

Design implication:

- parent owns child lifetime
- do not close the root container if children still need to resolve

## Practical Checklists

### Startup Checklist

- register all modules
- validate once
- format validation errors clearly
- only then begin serving traffic or running jobs

### Request/Job Checklist

- create one scope per operation
- resolve scoped values through that scope
- close the scope
- keep request-specific data out of singleton graphs

### Testing Checklist

- prefer explicit containers or `testutil`
- use overrides for temporary fakes
- reset globals only when truly needed
- keep test container setup close to the test
