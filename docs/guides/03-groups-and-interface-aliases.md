# Groups And Interface Aliases

Groups let you resolve many bindings at once. Interface aliases let you resolve by interface instead of concrete type.

## Groups

```go
package main

import (
	di "github.com/pakasa-io/di"
)

type StartupHook interface {
	Run() error
}

type CacheWarmup struct{}
type HealthProbe struct{}

func (*CacheWarmup) Run() error { return nil }
func (*HealthProbe) Run() error { return nil }

func main() {
	container := di.NewContainer()

	di.MustProvideGroupTo[*CacheWarmup](container, "startup", func() *CacheWarmup {
		return &CacheWarmup{}
	}, di.WithLifetime(di.LifetimeTransient))

	di.MustProvideGroupTo[*HealthProbe](container, "startup", func() *HealthProbe {
		return &HealthProbe{}
	}, di.WithLifetime(di.LifetimeTransient))

	hooks := di.MustResolveGroupFrom[StartupHook](container, "startup")
	for _, hook := range hooks {
		_ = hook.Run()
	}
}
```

## Interface aliases

```go
type Logger interface {
	Log(string)
}

type JSONLogger struct{}
type TextLogger struct{}

func (*JSONLogger) Log(string) {}
func (*TextLogger) Log(string) {}

container := di.NewContainer()

di.MustProvideAsTo[*JSONLogger, Logger](container, func() *JSONLogger {
	return &JSONLogger{}
})

di.MustProvideAsTo[*TextLogger, Logger](container, func() *TextLogger {
	return &TextLogger{}
})

allLoggers := di.MustResolveImplementationsFrom[Logger](container)
_ = allLoggers
```

Use groups for collections like startup hooks or middleware lists. Use interface aliases when you want interface-based resolution without registering the interface type directly.
