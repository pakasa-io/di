# Struct Auto-Wiring

Struct auto-wiring is opt-in. When enabled, unnamed struct dependencies can be materialized from their exported fields without registering the struct itself.

This feature is best used for dependency bundles: small structs that collect related collaborators and keep factories readable.

```go
package main

import (
	di "github.com/pakasa-io/di"
)

type DB struct{}
type Logger struct{}

type RepoDeps struct {
	DB     *DB
	Logger *Logger
}

type Repo struct {
	DB     *DB
	Logger *Logger
}

func main() {
	container := di.NewContainer()
	container.SetStructAutoWiring(true)

	di.MustProvideTo[*DB](container, func() *DB { return &DB{} })
	di.MustProvideTo[*Logger](container, func() *Logger { return &Logger{} })

	di.MustProvideTo[*Repo](container, func(deps RepoDeps) *Repo {
		return &Repo{
			DB:     deps.DB,
			Logger: deps.Logger,
		}
	})

	_ = di.MustResolveFrom[*Repo](container)
}
```

## Named Fields Inside Auto-Wired Structs

Tags on the dependency bundle still apply.

```go
type DB struct {
	Role string
}

type RepoDeps struct {
	Primary *DB `di:"name=primary"`
	Replica *DB `di:"name=replica"`
}

type Repo struct {
	Primary *DB
	Replica *DB
}

container := di.NewContainer()
container.SetStructAutoWiring(true)

di.MustProvideNamedTo[*DB](container, "primary", func() *DB {
	return &DB{Role: "primary"}
})
di.MustProvideNamedTo[*DB](container, "replica", func() *DB {
	return &DB{Role: "replica"}
})

di.MustProvideTo[*Repo](container, func(deps RepoDeps) *Repo {
	return &Repo{
		Primary: deps.Primary,
		Replica: deps.Replica,
	}
})
```

You can also enable it for newly created containers with:

```bash
DI_ENABLE_STRUCT_AUTOWIRING=true
```

## When To Use It

Use struct auto-wiring when:

- a factory signature is getting noisy from many collaborators
- the bundle is a plain dependency carrier, not business logic
- you want named and unnamed injected fields in one value object

Prefer explicit registration when:

- the bundle needs custom construction logic
- the dependency should be addressable or overridable as its own registered type
- you want the graph to stay fully explicit

## Constraints

- it only applies to unnamed struct dependencies
- the dependency type must be a struct value, not `*Struct`
- field tags like `di:"name=primary"` and `di:"-"` still work inside the auto-wired struct
- only exported fields are injected

## Gotchas

- `RepoDeps` works, `*RepoDeps` does not auto-wire
- named auto-wired bundle parameters are not supported because auto-wiring only applies to unnamed struct dependencies
- auto-wiring reduces boilerplate, but it can also hide the graph a bit; pair it with `Validate` and `Explain` in larger systems
