# Builder Hooks And Cleanup

Use the fluent builder when you want to separate registration from factory assignment or attach hooks directly.

```go
package main

import (
	"fmt"

	di "github.com/pakasa-io/di"
)

type Cache struct {
	Ready bool
}

func main() {
	container := di.NewContainer()

	di.MustBindTo[*Cache](container).
		ToSingleton().
		WithHooks(&di.LifecycleHooks{
			PostConstruct: func(instance any) error {
				instance.(*Cache).Ready = true
				return nil
			},
			CloseFunc: func() error {
				fmt.Println("cache closed")
				return nil
			},
		}).
		MustToFactory(func() *Cache {
			return &Cache{}
		})

	cache := di.MustResolveFrom[*Cache](container)
	fmt.Println(cache.Ready)

	if err := container.Close(); err != nil {
		panic(err)
	}
}
```

The same hook behavior can be attached with registration options:

```go
di.MustProvideTo[*Server](container, func() *Server {
	return &Server{}
},
	di.WithPreConstruct(func(any) error { return nil }),
	di.WithPostConstruct(func(any) error { return nil }),
	di.WithCloseFunc(func() error { return nil }),
)
```

Notes:

- `Close()` shuts down singleton instances and child overlay containers
- close hooks run in reverse registration order
- values that implement `Close() error` are closed automatically even without `WithCloseFunc`
