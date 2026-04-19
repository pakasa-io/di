# Getting Started

Start with an explicit container and typed factories. This guide shows the smallest useful graph, the factory shapes the container accepts, and the first production-safe checks to add.

## What You Learn Here

- how to create a container
- how factories are wired by type
- how to use error-returning factories
- when to use `Must*` helpers
- what to validate before booting the app

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type Config struct {
	DSN string
}

type DB struct {
	DSN string
}

type Repo struct {
	DB *DB
}

func main() {
	container := di.NewContainer()

	if err := di.ProvideTo[*Config](container, func() *Config {
		return &Config{DSN: "postgres://main"}
	}); err != nil {
		panic(err)
	}

	if err := di.ProvideTo[*DB](container, func(cfg *Config) *DB {
		return &DB{DSN: cfg.DSN}
	}); err != nil {
		panic(err)
	}

	if err := di.ProvideTo[*Repo](container, func(db *DB) *Repo {
		return &Repo{DB: db}
	}); err != nil {
		panic(err)
	}

	repo, err := di.ResolveFrom[*Repo](container)
	if err != nil {
		panic(err)
	}

	fmt.Println(repo.DB.DSN)
}
```

What is happening:

- `ProvideTo[T]` registers a factory for `T`
- `ResolveFrom[T]` returns the fully-built value for `T`
- the default lifetime is singleton
- `MustProvideTo` and `MustResolveFrom` are available if you prefer panic-on-failure startup code

## Factories Can Return Errors

Factories may return either `T` or `(T, error)`. Use the second form when construction can fail for real reasons like invalid config or missing credentials.

```go
type Config struct {
	DSN string
}

type DB struct {
	DSN string
}

container := di.NewContainer()

di.MustProvideTo[*Config](container, func() *Config {
	return &Config{DSN: "postgres://main"}
})

di.MustProvideTo[*DB](container, func(cfg *Config) (*DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("missing DSN")
	}
	return &DB{DSN: cfg.DSN}, nil
})
```

## Prefer Startup Validation

Before serving requests or running jobs, validate the graph once.

```go
import didiag "github.com/pakasa-io/di/diagnostics"

if err := didiag.Validate(container); err != nil {
	panic(didiag.FormatValidation(err))
}
```

Validation is especially useful when:

- the app has many modules registering into one container
- named bindings or aliases are selected indirectly
- a singleton might accidentally depend on scoped data

## Built-In Values

The resolver can inject a few built-ins without explicit registration:

- `context.Context` when you resolve or invoke with context-aware APIs
- `*di.Container` for the current container

Example:

```go
di.MustProvideTo[*Repo](container, func(ctx context.Context, c *di.Container, db *DB) *Repo {
	_ = ctx
	_ = c
	return &Repo{DB: db}
})
```

Use `ResolveFromContext`, `InvokeOnContext`, or `FillStructContext` when you want a specific request or job context injected.

## When To Use `Must*`

Use `MustProvideTo` and `MustResolveFrom` when:

- wiring happens during program startup
- a failure should crash boot immediately
- you want terse examples or test setup

Prefer the error-returning forms when:

- you are building reusable libraries
- container assembly happens conditionally
- you want to collect and format validation/setup failures instead of panicking

## Common Next Steps

- go to [02-named-bindings-and-explicit-deps.md](02-named-bindings-and-explicit-deps.md) when one type has multiple variants
- go to [05-lifetimes-and-scopes.md](05-lifetimes-and-scopes.md) when values should be request- or job-local
- go to [08-diagnostics-and-graphs.md](08-diagnostics-and-graphs.md) when you need validation and graph introspection
