# Overlays And Runtime Overrides

An overlay container inherits registrations and overrides from a parent, but keeps its own local additions. It is not a scope.

Use these features when you need different effective graphs without rebuilding everything from scratch.

## Scope Vs Overlay Vs Override

- use a `Scope` when registrations stay the same and only scoped caching should change
- use an overlay container when a child graph should inherit parent registrations but add or replace some local bindings
- use a runtime override when you need a temporary replacement for an already-addressable dependency

## Overlay Containers

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type Config struct {
	Env string
}

type JobRunner struct {
	Config *Config
}

func main() {
	parent := di.NewContainer()
	di.MustProvideTo[*Config](parent, func() *Config {
		return &Config{Env: "base"}
	})

	child := di.MustNewOverlayContainer(parent)
	di.MustProvideTo[*JobRunner](child, func(cfg *Config) *JobRunner {
		return &JobRunner{Config: cfg}
	})

	runner := di.MustResolveFrom[*JobRunner](child)
	fmt.Println(runner.Config.Env)
}
```

Overlay containers are a good fit for:

- tenant-specific graphs on top of shared infrastructure
- integration tests that add a few extra bindings
- modular apps where child modules extend a base graph

## Override A Named Binding

Overrides support the same options used for addressing dependencies.

```go
type Config struct {
	Env string
}

container := di.NewContainer()

di.MustProvideNamedTo[*Config](container, "tenant", func() *Config {
	return &Config{Env: "prod"}
})

restore := di.MustOverrideInContainer[*Config](container, func() (*Config, error) {
	return &Config{Env: "staging"}, nil
}, di.WithName("tenant"))
defer restore()

cfg := di.MustResolveNamedFrom[*Config](container, "tenant")
fmt.Println(cfg.Env)
```

## Runtime Overrides

```go
type Clock struct {
	Now string
}

container := di.NewContainer()
di.MustProvideTo[*Clock](container, func() *Clock {
	return &Clock{Now: "real"}
})

restore := di.MustOverrideInContainer[*Clock](container, func() (*Clock, error) {
	return &Clock{Now: "fake"}, nil
})
defer restore()

clock := di.MustResolveFrom[*Clock](container)
fmt.Println(clock.Now)
```

## Important Override Behavior

- override factories run at resolution time
- overrides are not cached by a binding lifetime
- if you want one stable replacement instance, capture it in the closure yourself
- parent overrides are visible from overlay children

Stable override example:

```go
fake := &Clock{Now: "fixed"}

restore := di.MustOverrideInContainer[*Clock](container, func() (*Clock, error) {
	return fake, nil
})
defer restore()
```

Use this pattern for:

- tenant-specific overlays built on a shared base graph
- temporary replacements in integration tests
- feature-flagged runtime swaps without mutating registrations

## Gotchas

- closing a parent container also closes its overlay children
- named global containers do not inherit from one another; overlays are the inheritance tool
- overrides are powerful but easy to overuse; for long-lived graph differences, prefer explicit registrations or overlays

Choose carefully:

- use a `Scope` for per-request caching
- use an overlay container for inherited registrations with local additions
- use an override for temporary runtime replacement of an existing binding
