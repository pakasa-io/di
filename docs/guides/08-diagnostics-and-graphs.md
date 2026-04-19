# Diagnostics And Graphs

Use `diagnostics` when you want startup-time validation, readable dependency explanations, or graph dumps.

## Validate Before Serving Traffic

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
	didiag "github.com/pakasa-io/di/diagnostics"
)

type DB struct{}
type Repo struct {
	DB *DB
}

func main() {
	container := di.NewContainer()

	di.MustProvideTo[*DB](container, func() *DB { return &DB{} })
	di.MustProvideTo[*Repo](container, func(db *DB) *Repo {
		return &Repo{DB: db}
	})

	if err := didiag.Validate(container); err != nil {
		fmt.Println(didiag.FormatValidation(err))
		return
	}

	explanation := must(didiag.Explain[*Repo](container))
	fmt.Println(explanation.String())

	dump := must(didiag.DumpGraph(container))
	fmt.Println(dump)

	dot := must(didiag.DumpGraphDOT(container))
	fmt.Println(dot)
}

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
```

`Validate` checks the graph without constructing instances. This is the safest place to catch:

- missing dependencies
- circular dependencies
- invalid singleton-to-scoped or singleton-to-context edges
- alias collisions and other registration issues

## Formatting Validation Failures

For CLI tools or startup logs, format the error instead of dumping the raw `error` value.

```go
broken := di.NewContainer()

di.MustProvideTo[*Repo](broken, func(db *DB) *Repo {
	return &Repo{DB: db}
})

if err := didiag.Validate(broken); err != nil {
	fmt.Println(didiag.FormatValidation(err))
}
```

## Explain Why A Binding Was Chosen

`Explain` is useful when names, aliases, auto-wiring, overlays, or overrides make the chosen path non-obvious.

```go
explanation := must(didiag.Explain[*Repo](container))
fmt.Println(explanation.String())
```

Named and scoped variants are available too:

```go
named := must(didiag.ExplainNamed[*DB](container, "primary"))
_ = named

scope := container.MustNewScope()
defer scope.Close()

scoped := must(didiag.ExplainInScope[*Repo](scope))
_ = scoped
```

## Metadata Helpers For Tooling

```go
info := didiag.DescribeContainer(container)
bindings := didiag.ListBindings(container)
hasRepo := didiag.Has[*Repo](container)

_, _, _ = info, bindings, hasRepo
```

Use these when building:

- admin or debug endpoints
- startup summaries
- tests that assert graph shape
- custom docs or code-generation helpers

## Reading Graph Dumps

`DumpGraph` gives a readable text view. `DumpGraphDOT` gives Graphviz output.

Graph dumps can show:

- ordinary bindings
- inherited bindings
- auto-wired nodes
- overrides
- missing dependencies
- optional missing dependencies

Use this guide for:

- validating the graph at startup
- printing a human-readable failure report
- exporting DOT for Graphviz
- inspecting which binding an interface or named dependency resolves to
