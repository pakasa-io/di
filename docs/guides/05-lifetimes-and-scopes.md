# Lifetimes And Scopes

`di` supports three lifetimes:

- `singleton` for one instance per container
- `transient` for a new instance on each resolve
- `scoped` for one instance per scope

```go
package main

import (
	"fmt"
	"sync/atomic"

	di "github.com/pakasa-io/di"
)

type RequestCache struct {
	ID int32
}

func main() {
	container := di.NewContainer()

	var seq atomic.Int32
	di.MustProvideTo[*RequestCache](container, func() *RequestCache {
		return &RequestCache{ID: seq.Add(1)}
	}, di.WithLifetime(di.LifetimeScoped))

	scopeA := container.MustNewScope()
	scopeB := container.MustNewScope()
	defer scopeA.Close()
	defer scopeB.Close()

	a1 := di.MustResolveInScope[*RequestCache](scopeA)
	a2 := di.MustResolveInScope[*RequestCache](scopeA)
	b1 := di.MustResolveInScope[*RequestCache](scopeB)

	fmt.Println(a1 == a2)
	fmt.Println(a1 != b1)
}
```

## Choosing The Right Lifetime

- use `singleton` for shared clients, config, pools, and immutable services
- use `transient` for cheap throwaway values or stateless wrappers
- use `scoped` for request/job-local state, units of work, and per-operation caches

## Transient Example

```go
di.MustProvideTo[*Token](container, func() *Token {
	return &Token{}
}, di.WithLifetime(di.LifetimeTransient))
```

Every resolve gets a new instance:

```go
first := di.MustResolveFrom[*Token](container)
second := di.MustResolveFrom[*Token](container)

fmt.Println(first != second)
```

## Singleton Example

Singleton is the default:

```go
type Logger struct{}

di.MustProvideTo[*Logger](container, func() *Logger {
	return &Logger{}
})

first := di.MustResolveFrom[*Logger](container)
second := di.MustResolveFrom[*Logger](container)

fmt.Println(first == second)
```

## Request Or Job Scope Pattern

Create one scope per request, job, or workflow.

```go
type RequestState struct {
	ID string
}

type Handler struct {
	State *RequestState
}

di.MustProvideTo[*RequestState](container, func(ctx context.Context) *RequestState {
	value, _ := ctx.Value("request_id").(string)
	return &RequestState{ID: value}
}, di.WithLifetime(di.LifetimeScoped))

di.MustProvideTo[*Handler](container, func(state *RequestState) *Handler {
	return &Handler{State: state}
}, di.WithLifetime(di.LifetimeScoped))

scope := container.MustNewScope()
defer scope.Close()

ctx := context.WithValue(context.Background(), "request_id", "req-42")
handler := di.MustResolveInScopeContext[*Handler](ctx, scope)
fmt.Println(handler.State.ID)
```

## Important Rules

- resolve scoped bindings through a `Scope` when you want request-local reuse
- a singleton graph should not depend on `context.Context` or scoped values; `Validate()` will report that as an invalid lifetime graph

## Scope Notes

- a container can create many sibling scopes
- a scope can create nested child scopes with `scope.NewScope()`
- scoped caching is tied to the scope you resolve through
- scopes are not registration overlays; they only affect scoped instance lifetimes

## Scope Vs Overlay

Use a scope when the registrations stay the same but cached instances should be isolated.

Use an overlay container when the registrations themselves differ between children while still inheriting from a parent.
