# Complex Patterns

These patterns combine several features into one design instead of using each API in isolation.

## HTTP app with request scopes

```go
type DB struct{}

type RequestState struct {
	RequestID string
}

type Handler struct {
	DB    *DB
	State *RequestState
}

root := di.NewContainer()

di.MustProvideTo[*DB](root, func() *DB {
	return &DB{}
})

di.MustProvideTo[*RequestState](root, func(ctx context.Context) *RequestState {
	id, _ := ctx.Value("request_id").(string)
	return &RequestState{RequestID: id}
}, di.WithLifetime(di.LifetimeScoped))

di.MustProvideTo[*Handler](root, func(db *DB, state *RequestState) *Handler {
	return &Handler{DB: db, State: state}
}, di.WithLifetime(di.LifetimeScoped))

scope := root.MustNewScope()
defer scope.Close()

ctx := context.WithValue(context.Background(), "request_id", "req-42")
handler, err := di.ResolveInScopeContext[*Handler](ctx, scope)
_ = handler
_ = err
```

Why it works:

- `DB` is shared as a singleton
- `RequestState` is cached once per request scope
- `context.Context` can still feed scoped factories

Tradeoff:

- the request-scoped types must stay scoped or transient; do not promote this graph into a singleton root consumer

## Plugin host with groups and interface aliases

```go
type Plugin interface {
	Name() string
}

type Startup interface {
	Run() error
}

type SearchPlugin struct{}
type BillingPlugin struct{}

func (*SearchPlugin) Name() string  { return "search" }
func (*BillingPlugin) Name() string { return "billing" }
func (*SearchPlugin) Run() error    { return nil }
func (*BillingPlugin) Run() error   { return nil }

container := di.NewContainer()

di.MustProvideAsTo[*SearchPlugin, Plugin](container, func() *SearchPlugin {
	return &SearchPlugin{}
}, di.WithName("search"))

di.MustProvideAsTo[*BillingPlugin, Plugin](container, func() *BillingPlugin {
	return &BillingPlugin{}
}, di.WithName("billing"))

di.MustProvideGroupTo[*SearchPlugin](container, "startup", func() *SearchPlugin {
	return &SearchPlugin{}
}, di.WithLifetime(di.LifetimeTransient))

di.MustProvideGroupTo[*BillingPlugin](container, "startup", func() *BillingPlugin {
	return &BillingPlugin{}
}, di.WithLifetime(di.LifetimeTransient))

plugins := di.MustResolveImplementationsFrom[Plugin](container)
hooks := di.MustResolveGroupFrom[Startup](container, "startup")

_, _ = plugins, hooks
```

Why it works:

- interface aliases provide a typed plugin catalog
- groups provide batch startup execution
- names let you target one implementation when needed

Tradeoff:

- if you need deterministic startup order, encode it in your registration or plugin types rather than assuming group member order has business meaning

## Tenant overlays with diagnostics and overrides

```go
type Config struct {
	Tenant string
}

base := di.NewContainer()
di.MustProvideTo[*Config](base, func() *Config {
	return &Config{Tenant: "shared"}
})

tenant := di.MustNewOverlayContainer(base)
restore := di.MustOverrideInContainer[*Config](tenant, func() (*Config, error) {
	return &Config{Tenant: "acme"}, nil
})
defer restore()

if err := tenant.Validate(); err != nil {
	panic(err)
}

explanation := must(di.ExplainFrom[*Config](tenant))
fmt.Println(explanation.String())
```

Why it works:

- the overlay inherits the base graph
- the override swaps one dependency without rebinding everything
- validation and explanation still operate on the effective graph

Tradeoff:

- this is ideal for temporary or per-tenant customization, but a long-lived product split may be clearer as explicit child registrations instead of pervasive overrides

## Command Runner With `Invoke`

This pattern works well for CLIs or admin jobs where the command function is the entry point.

```go
type AuditService struct{}
type CleanupService struct{}

func runCleanup(audit *AuditService, cleanup *CleanupService) error {
	_ = audit
	_ = cleanup
	return nil
}

container := di.NewContainer()

di.MustProvideTo[*AuditService](container, func() *AuditService { return &AuditService{} })
di.MustProvideTo[*CleanupService](container, func() *CleanupService { return &CleanupService{} })

if err := di.InvokeOn(container, runCleanup); err != nil {
	panic(err)
}
```

Why it works:

- no extra registered command type is required
- command signatures stay explicit and testable
- the command can still return `error`

## Per-Tenant Child Graphs

This pattern gives each tenant a small custom graph while keeping shared infrastructure in one place.

```go
type SharedDB struct{}
type TenantConfig struct {
	Name string
}

base := di.NewContainer()
di.MustProvideTo[*SharedDB](base, func() *SharedDB { return &SharedDB{} })

acme := di.MustNewOverlayContainer(base)
di.MustProvideTo[*TenantConfig](acme, func() *TenantConfig {
	return &TenantConfig{Name: "acme"}
})

beta := di.MustNewOverlayContainer(base)
di.MustProvideTo[*TenantConfig](beta, func() *TenantConfig {
	return &TenantConfig{Name: "beta"}
})
```

Why it works:

- shared dependencies live once in the base container
- tenant-specific services can be added locally
- each child can still be validated independently

```go
func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}
```
