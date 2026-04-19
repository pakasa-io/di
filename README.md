# di

`di` is a typed dependency injection library for Go with explicit containers, scopes, named bindings, interface aliases, groups, validation, graph dumps, runtime overrides, and lightweight instrumentation.

The public API is non-panicking by default. Functions that can fail return an `error`. If you prefer startup-time panics, use the matching `Must*` helpers.

## Install

```bash
go get github.com/pakasa-io/di@latest
```

Go version:

- the repository follows the Go version declared in [go.mod](go.mod)

## Packages

- `github.com/pakasa-io/di`
  Explicit container, scope, options, dependency wrappers, and injector APIs.
- `github.com/pakasa-io/di/global`
  Process-wide default and named container helpers.
- `github.com/pakasa-io/di/diagnostics`
  Validation, explanation, graph, and metadata helpers.
- `github.com/pakasa-io/di/testutil`
  Test fixtures, isolated containers, scoped helpers, and overrides.

## Highlights

- typed registration and resolution APIs using generics
- named bindings, groups, and interface aliases
- singleton, transient, and scoped lifetimes
- validation, explanation, graph dumps, and introspection
- runtime overrides and overlay containers
- optional dependencies and opt-in struct auto-wiring
- test helpers for isolated container state

## Quick Start

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type DB struct{}
type Repo struct{ DB *DB }

func main() {
	container := di.NewContainer()

	if err := di.ProvideTo[*DB](container, func() *DB { return &DB{} }); err != nil {
		panic(err)
	}
	if err := di.ProvideTo[*Repo](container, func(db *DB) *Repo { return &Repo{DB: db} }); err != nil {
		panic(err)
	}

	repo, err := di.ResolveFrom[*Repo](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(repo.DB != nil)
}
```

## Global Container Variant

If you prefer process-wide containers:

```go
package main

import (
	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
)

type DB struct{}

func main() {
	_ = diglobal.Provide[*DB](func() *DB { return &DB{} })

	db := diglobal.MustResolve[*DB]()
	_ = db
}
```

## Core Recipes

### Named bindings

```go
container := di.NewContainer()

_ = di.ProvideNamedTo[*DB](container, "primary", func() *DB {
	return &DB{}
})

db := di.MustResolveNamedFrom[*DB](container, "primary")
_ = db
```

### Interface aliases

```go
type Clock interface {
	Now() int64
}

type SystemClock struct{}

func (*SystemClock) Now() int64 { return 42 }

container := di.NewContainer()

_ = di.ProvideAsTo[*SystemClock, Clock](container, func() *SystemClock {
	return &SystemClock{}
})

clock := di.MustResolveFrom[Clock](container)
_ = clock
```

### Groups

```go
type Hook interface {
	Run() error
}

type HookA struct{}

func (*HookA) Run() error { return nil }

container := di.NewContainer()

_ = di.ProvideGroupTo[*HookA](container, "startup", func() *HookA {
	return &HookA{}
}, di.WithLifetime(di.LifetimeTransient))

hooks, _ := di.ResolveGroupFrom[Hook](container, "startup")
_ = hooks
```

### Scopes

```go
container := di.NewContainer()
_ = di.ProvideTo[*DB](container, func() *DB { return &DB{} }, di.WithLifetime(di.LifetimeScoped))

scope, _ := container.NewScope()
defer scope.Close()

db1, _ := di.ResolveInScope[*DB](scope)
db2, _ := di.ResolveInScope[*DB](scope)

fmt.Println(db1 == db2)
```

### Validation and explanation

```go
import didiag "github.com/pakasa-io/di/diagnostics"

if err := didiag.Validate(container); err != nil {
	fmt.Println(didiag.FormatValidation(err))
}

explanation, err := didiag.Explain[*Repo](container)
if err == nil {
	fmt.Println(explanation.String())
}
```

### Optional dependencies

```go
type Handler struct {
	Metrics di.Optional[*Metrics]
}

container := di.NewContainer()

_ = di.ProvideTo[*Handler](container, func(metrics di.Optional[*Metrics]) *Handler {
	return &Handler{Metrics: metrics}
})

handler, _ := di.ResolveFrom[*Handler](container)
if handler.Metrics.OK {
	fmt.Println("metrics enabled")
}
```

`Optional[T]` is a resolution-time wrapper. Register or override `T` directly, and use aggregate APIs with `T` rather than `Optional[T]`.

### Opt-in struct auto-wiring

```go
type RepoDeps struct {
	DB *DB
}

container := di.NewContainer()
container.SetStructAutoWiring(true)

_ = di.ProvideTo[*DB](container, func() *DB { return &DB{} })
_ = di.ProvideTo[*Repo](container, func(deps RepoDeps) *Repo {
	return &Repo{DB: deps.DB}
})
```

Implicit struct auto-wiring is disabled by default. Opt in per container with `container.SetStructAutoWiring(true)` or for newly created containers with the `DI_ENABLE_STRUCT_AUTOWIRING=true` environment variable.

### Testing

Use `github.com/pakasa-io/di/testutil` for isolated named containers, scoped test contexts, and temporary overrides.

```go
name, container := testutil.Container(t)
testutil.MustProvide[*DB](t, name, func() *DB { return &DB{} })

scope := testutil.Scope(t)
_ = container
_ = scope
```

## Documentation

- API docs: [pkg.go.dev/github.com/pakasa-io/di](https://pkg.go.dev/github.com/pakasa-io/di)
- Guides: [docs/guides/README.md](docs/guides/README.md)
- Contributing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Release process: [RELEASING.md](RELEASING.md)

## Versioning

Releases are tagged with semantic versions.

- before `v1`, minor releases may include breaking changes
- `v1` will indicate a stable public API with stronger compatibility expectations

## License

[MIT](LICENSE)
