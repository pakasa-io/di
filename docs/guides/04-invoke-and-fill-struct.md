# Invoke And Fill Struct

You can inject dependencies into functions and struct fields without writing a dedicated factory.

## Invoke A Function

```go
package main

import (
	"context"
	"fmt"

	di "github.com/pakasa-io/di"
)

type UserService struct{}

func main() {
	container := di.NewContainer()
	di.MustProvideTo[*UserService](container, func() *UserService {
		return &UserService{}
	})

	err := di.InvokeOnContext(context.Background(), container, func(ctx context.Context, svc *UserService) {
		fmt.Println(ctx != nil, svc != nil)
	})
	if err != nil {
		panic(err)
	}
}
```

`InvokeOn` and `InvokeOnContext` are useful for edge entry points:

- CLI subcommands
- HTTP startup hooks
- one-shot jobs
- adapters that should not own container plumbing directly

If the invoked function returns an `error`, that error is propagated:

```go
err := di.InvokeOn(container, func(svc *UserService) error {
	_ = svc
	return nil
})
```

You can also invoke against a scope:

```go
scope := container.MustNewScope()
defer scope.Close()

err := scope.Invoke(func(svc *UserService) error {
	_ = svc
	return nil
})
_ = err
```

## Fill A Struct

```go
type Metrics struct{}

type Handler struct {
	Default   *UserService
	Primary   *UserService `di:"name=primary"`
	Metrics   di.Optional[*Metrics]
	Request   context.Context
	Container *di.Container
	SkipMe    string `di:"-"`
}

container := di.NewContainer()

di.MustProvideTo[*UserService](container, func() *UserService {
	return &UserService{}
})

di.MustProvideNamedTo[*UserService](container, "primary", func() *UserService {
	return &UserService{}
})

ctx := context.Background()

var handler Handler
if err := container.Injector().FillStructContext(ctx, &handler); err != nil {
	panic(err)
}
```

`FillStruct` is most useful for adapter structs that already exist for framework reasons, such as HTTP handlers, commands, or workers.

## Field Rules

- the target must be a non-nil pointer to a struct
- only exported fields are considered
- `di:"name=primary"` selects a named binding
- `di:"-"` skips a field
- optional fields can use `di.Optional[T]`
- `context.Context` is only injected when using `FillStructContext`

## Choosing Between Factories, Invoke, And FillStruct

Prefer factory registration when:

- the type is part of the core dependency graph
- construction logic deserves one canonical place
- the value will be resolved many times

Prefer `Invoke` when:

- the function is an entry point, not a reusable dependency
- you want DI without introducing a new registered type

Prefer `FillStruct` when:

- a framework constructs the value for you
- field injection is more ergonomic than building an adapter factory
- you only need a few injected collaborators

## Gotchas

- `FillStruct` does not set unexported fields
- `FillStruct` does not work on non-struct pointers
- field injection is convenient, but constructor/factory injection is usually better for core business objects

Use this when:

- a CLI command wants injected function parameters
- a handler struct needs a few injected fields
- you want `context.Context` or `*di.Container` filled automatically
