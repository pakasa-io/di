# Introspection Metrics And Instrumentation

`di` exposes both metadata APIs and lightweight runtime instrumentation.

## Introspection

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type Service struct{}

func main() {
	container := di.NewContainer()
	di.MustProvideTo[*Service](container, func() *Service { return &Service{} })

	fmt.Println(di.HasInContainer[*Service](container))

	info, err := di.DescribeInContainer[*Service](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(info.Key, info.Lifetime, info.HasFactory)
}
```

Useful introspection APIs:

- `HasInContainer[T]` to check whether a binding or override is visible
- `DescribeInContainer[T]` to inspect the selected binding metadata
- `ListBindings()` to see local registrations in registration order
- `DescribeContainer()` to get container-level counts and state

One important detail: `ListBindings()` is local to that container. For an inherited view, use `Explain`, `Graph`, or `DescribeBinding` on the child.

## Instrumentation And Metrics

```go
resolveCount := 0

container.SetInstrumentation(di.Instrumentation{
	OnResolve: func(event di.ResolveEvent) {
		resolveCount++
		fmt.Println(event.Key, event.Duration, event.Err)
	},
	OnInstanceCreated: func(event di.InstanceEvent) {
		fmt.Println(event.Key, event.Lifetime, event.Err)
	},
})

_, _ = di.ResolveFrom[*Service](container)

metrics := container.Metrics()
fmt.Println(metrics.Resolutions, metrics.InstancesCreated, metrics.OverrideCalls)

container.ResetMetrics()
```

## What The Metrics Mean

- `Resolutions` counts all resolution attempts
- `ResolutionErrors` counts failed resolutions
- `InstancesCreated` counts successful instance constructions
- `InstanceCreationErrors` counts failed constructions
- `OverrideCalls` counts resolutions satisfied by an override

## Callback Guidance

Instrumentation callbacks run inline after the observed operation completes. Keep them lightweight.

Good uses:

- increment counters
- emit structured logs
- send small timing samples

Avoid:

- blocking network calls
- expensive formatting on every resolution
- logic that mutates the container graph

Use this when you need:

- startup reports of what is registered
- admin/debug endpoints describing the active graph
- lightweight latency and error counters around DI activity
- visibility into override usage during tests or experiments
