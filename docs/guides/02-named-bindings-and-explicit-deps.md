# Named Bindings And Explicit Deps

Use names when you need multiple bindings of the same type.

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type DB struct {
	Role string
}

type ReadRepo struct {
	DB *DB
}

func main() {
	container := di.NewContainer()

	di.MustProvideNamedTo[*DB](container, "primary", func() *DB {
		return &DB{Role: "primary"}
	})

	di.MustProvideNamedTo[*DB](container, "replica", func() *DB {
		return &DB{Role: "replica"}
	})

	di.MustProvideTo[*ReadRepo](container, func(db *DB) *ReadRepo {
		return &ReadRepo{DB: db}
	}, di.WithDeps(di.Named[*DB]("replica")))

	primary := di.MustResolveNamedFrom[*DB](container, "primary")
	reader := di.MustResolveFrom[*ReadRepo](container)

	fmt.Println(primary.Role)
	fmt.Println(reader.DB.Role)
}
```

Use `WithDeps` when parameter types alone are not enough.

```go
di.MustProvideTo[*Service](container, func(cfg *Config, db *DB) *Service {
	return &Service{Config: cfg, DB: db}
}, di.WithDeps(
	di.Dep[*Config](),
	di.Named[*DB]("primary"),
))
```

Use this pattern for:

- read/write database splits
- multiple clients of the same concrete type
- feature-specific bindings selected by name
