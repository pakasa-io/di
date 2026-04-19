# Global Containers

Use `global` when you want a process-wide default container or a few named application containers.

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
)

type Cache struct{}
type Worker struct {
	Cache *Cache
}

func main() {
	diglobal.MustProvide[*Cache](func() *Cache {
		return &Cache{}
	})

	diglobal.MustProvideIn[*Cache]("jobs", func() *Cache {
		return &Cache{}
	})

	diglobal.MustProvideIn[*Worker]("jobs", func(cache *Cache) *Worker {
		return &Worker{Cache: cache}
	})

	defaultCache := diglobal.MustResolve[*Cache]()
	jobWorker := diglobal.MustResolveIn[*Worker]("jobs")

	fmt.Println(defaultCache != nil, jobWorker.Cache != nil)
}
```

Named global containers are useful when one process hosts multiple graphs:

- API container
- background jobs container
- migration container

Each named container is independent. Register shared dependencies in every named graph that needs them, or use explicit root containers plus overlays when you want inheritance.

You still get scopes, metrics, overrides, and auto-wiring:

```go
scope := diglobal.MustNewScope("jobs")
defer scope.Close()

diglobal.SetStructAutoWiring(true, "jobs")
_ = di.MustResolveInScope[*Worker](scope)
```
