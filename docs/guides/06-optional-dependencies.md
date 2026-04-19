# Optional Dependencies

Use `di.Optional[T]` when a dependency is allowed to be absent.

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type Metrics struct{}

type Handler struct {
	Metrics di.Optional[*Metrics]
}

func main() {
	container := di.NewContainer()

	di.MustProvideTo[*Handler](container, func(metrics di.Optional[*Metrics]) *Handler {
		return &Handler{Metrics: metrics}
	})

	handler := di.MustResolveFrom[*Handler](container)
	fmt.Println(handler.Metrics.OK)
}
```

This is a good fit for:

- optional metrics or tracing integrations
- feature-specific collaborators
- admin-only components
- gradual rollouts where a dependency may or may not be present

## Optional Fields Work With `FillStruct` Too

```go
type Target struct {
	Metrics di.Optional[*Metrics]
	Primary di.Optional[*Metrics] `di:"name=primary"`
}

var target Target
_ = container.Injector().FillStruct(&target)
```

## Optional Built-Ins

Optional resolution also works with built-ins:

```go
type Inspect struct {
	Ctx       di.Optional[context.Context]
	Container di.Optional[*di.Container]
}

var inspect Inspect
_ = container.Injector().FillStructContext(context.Background(), &inspect)
```

## Fallback Pattern

Optional values are explicit. You choose the fallback behavior in your own code.

```go
type Metrics interface {
	Inc(string)
}

type NoopMetrics struct{}

func (NoopMetrics) Inc(string) {}

func effectiveMetrics(optional di.Optional[Metrics]) Metrics {
	if optional.OK {
		return optional.Value
	}
	return NoopMetrics{}
}
```

## Rules

- register `T`, not `Optional[T]`
- override `T`, not `Optional[T]`
- aggregate APIs use `T`, not `Optional[T]`
- a missing optional dependency is fine, but a broken existing binding still fails resolution and validation

## Gotchas

- optional does not hide real construction failures
- `Optional[T]` is for consumption only; it is not a registration type
- optional aggregates like `ResolveGroupFrom[Optional[*X]]` are unsupported
