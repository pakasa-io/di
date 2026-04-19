package global_test

import (
	"context"
	"sync/atomic"
	"testing"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
)

type service struct {
	ID string
}

type boundValue struct {
	ID string
}

type mustBoundValue struct {
	ID string
}

type ctxValue struct {
	Value string
}

type groupValue struct {
	ID int
}

type overrideValue struct {
	ID string
}

type fillTarget struct {
	Service *service
}

type alias interface {
	AliasID() string
}

type aliasImpl struct{}

func (*aliasImpl) AliasID() string { return "alias-1" }

type aliasImpl2 struct{}

func (*aliasImpl2) AliasID() string { return "alias-2" }

type ctxKey struct{}

func prepareGlobal(t *testing.T) {
	t.Helper()
	if err := diglobal.Reset(); err != nil {
		t.Fatalf("reset: %v", err)
	}
	di.ClearRuntimeCaches()
	t.Cleanup(func() {
		_ = diglobal.Reset()
		di.ClearRuntimeCaches()
	})
}

func TestDefaultContainerWrappers(t *testing.T) {
	prepareGlobal(t)

	if diglobal.Default() != diglobal.Container() {
		t.Fatal("expected Container() to return the default container")
	}

	diglobal.SetStructAutoWiring(true)
	if !diglobal.StructAutoWiringEnabled() {
		t.Fatal("expected struct auto-wiring to be enabled")
	}
	diglobal.SetStructAutoWiring(false)

	builder, err := diglobal.Bind[*boundValue]()
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if _, err := builder.ToFactory(func() *boundValue { return &boundValue{ID: "bound"} }); err != nil {
		t.Fatalf("ToFactory failed: %v", err)
	}
	diglobal.MustBind[*mustBoundValue]().MustToFactory(func() *mustBoundValue {
		return &mustBoundValue{ID: "must-bound"}
	})

	if err := diglobal.Provide[*service](func() *service { return &service{ID: "svc"} }); err != nil {
		t.Fatalf("Provide failed: %v", err)
	}
	diglobal.MustProvide[*ctxValue](func(ctx context.Context) *ctxValue {
		value, _ := ctx.Value(ctxKey{}).(string)
		return &ctxValue{Value: value}
	}, di.WithLifetime(di.LifetimeTransient))

	if err := diglobal.ProvideNamed[*service]("named", func() *service { return &service{ID: "named"} }); err != nil {
		t.Fatalf("ProvideNamed failed: %v", err)
	}
	diglobal.MustProvideNamed[*service]("named-2", func() *service { return &service{ID: "named-2"} })

	if err := diglobal.ProvideGroup[*groupValue]("helpers", func() *groupValue {
		return &groupValue{ID: 1}
	}, di.WithName("one"), di.WithLifetime(di.LifetimeTransient)); err != nil {
		t.Fatalf("ProvideGroup failed: %v", err)
	}
	diglobal.MustProvideGroup[*groupValue]("helpers", func() *groupValue {
		return &groupValue{ID: 2}
	}, di.WithName("two"), di.WithLifetime(di.LifetimeTransient))

	if err := diglobal.ProvideAs[*aliasImpl, alias](func() *aliasImpl { return &aliasImpl{} }, di.WithName("one")); err != nil {
		t.Fatalf("ProvideAs failed: %v", err)
	}
	diglobal.MustProvideAs[*aliasImpl2, alias](func() *aliasImpl2 { return &aliasImpl2{} }, di.WithName("two"))

	if err := diglobal.Provide[*overrideValue](func() *overrideValue { return &overrideValue{ID: "base"} }); err != nil {
		t.Fatalf("Provide override target failed: %v", err)
	}

	if value, err := diglobal.Resolve[*service](); err != nil || value.ID != "svc" {
		t.Fatalf("Resolve failed: %#v %v", value, err)
	}
	if value := diglobal.MustResolve[*service](); value.ID != "svc" {
		t.Fatalf("MustResolve failed: %#v", value)
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "req-123")
	if value, err := diglobal.ResolveContext[*ctxValue](ctx); err != nil || value.Value != "req-123" {
		t.Fatalf("ResolveContext failed: %#v %v", value, err)
	}
	if value, err := diglobal.ResolveNamed[*service]("named"); err != nil || value.ID != "named" {
		t.Fatalf("ResolveNamed failed: %#v %v", value, err)
	}
	if value, err := diglobal.ResolveNamedContext[*service](ctx, "named-2"); err != nil || value.ID != "named-2" {
		t.Fatalf("ResolveNamedContext failed: %#v %v", value, err)
	}
	if value := diglobal.MustResolveNamed[*service]("named"); value.ID != "named" {
		t.Fatalf("MustResolveNamed failed: %#v", value)
	}

	if values, err := diglobal.ResolveGroup[*groupValue]("helpers"); err != nil || len(values) != 2 {
		t.Fatalf("ResolveGroup failed: %#v %v", values, err)
	}
	if values, err := diglobal.ResolveGroupContext[*groupValue](ctx, "helpers"); err != nil || len(values) != 2 {
		t.Fatalf("ResolveGroupContext failed: %#v %v", values, err)
	}
	if values := diglobal.MustResolveGroup[*groupValue]("helpers"); len(values) != 2 {
		t.Fatalf("MustResolveGroup failed: %#v", values)
	}

	if values, err := diglobal.ResolveImplementations[alias](); err != nil || len(values) != 2 {
		t.Fatalf("ResolveImplementations failed: %#v %v", values, err)
	}
	if values, err := diglobal.ResolveImplementationsContext[alias](ctx); err != nil || len(values) != 2 {
		t.Fatalf("ResolveImplementationsContext failed: %#v %v", values, err)
	}
	if values := diglobal.MustResolveImplementations[alias](); len(values) != 2 {
		t.Fatalf("MustResolveImplementations failed: %#v", values)
	}

	if err := diglobal.Invoke(func(s *service) error {
		if s.ID != "svc" {
			t.Fatalf("unexpected service: %#v", s)
		}
		return nil
	}); err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if err := diglobal.InvokeContext(ctx, func(injected context.Context, v *ctxValue) error {
		if injected.Value(ctxKey{}) != "req-123" || v.Value != "req-123" {
			t.Fatalf("unexpected invoke context values: %#v %#v", injected, v)
		}
		return nil
	}); err != nil {
		t.Fatalf("InvokeContext failed: %v", err)
	}

	injector := diglobal.Injector()
	if injector == nil {
		t.Fatal("expected Injector to return a value")
	}
	if _, err := injector.Call(func(s *service) {}); err != nil {
		t.Fatalf("Injector.Call failed: %v", err)
	}
	var target fillTarget
	if err := injector.FillStruct(&target); err != nil || target.Service == nil || target.Service.ID != "svc" {
		t.Fatalf("Injector.FillStruct failed: %#v %v", target, err)
	}

	var resolveEvents atomic.Int32
	diglobal.SetInstrumentation(di.Instrumentation{
		OnResolve: func(di.ResolveEvent) {
			resolveEvents.Add(1)
		},
	})
	diglobal.ResetMetrics()
	if _, err := diglobal.Resolve[*service](); err != nil {
		t.Fatalf("Resolve for metrics failed: %v", err)
	}
	if diglobal.Metrics().Resolutions == 0 || resolveEvents.Load() == 0 {
		t.Fatalf("expected metrics/instrumentation to record resolution, got %#v count=%d", diglobal.Metrics(), resolveEvents.Load())
	}
	diglobal.ResetMetrics()
	diglobal.SetInstrumentation(di.Instrumentation{})

	restore, err := diglobal.Override[*overrideValue](func() (*overrideValue, error) {
		return &overrideValue{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("Override failed: %v", err)
	}
	if value, err := diglobal.Resolve[*overrideValue](); err != nil || value.ID != "override" {
		t.Fatalf("override resolve failed: %#v %v", value, err)
	}
	restore()
	restore = diglobal.MustOverride[*overrideValue](func() (*overrideValue, error) {
		return &overrideValue{ID: "must-override"}, nil
	})
	if value, err := diglobal.Resolve[*overrideValue](); err != nil || value.ID != "must-override" {
		t.Fatalf("must override resolve failed: %#v %v", value, err)
	}
	restore()

	scope, err := diglobal.NewScope()
	if err != nil {
		t.Fatalf("NewScope failed: %v", err)
	}
	mustScope := diglobal.MustNewScope()
	_ = mustScope.Close()
	if _, err := di.ResolveInScope[*service](scope); err != nil {
		t.Fatalf("ResolveInScope via global scope failed: %v", err)
	}
	if err := scope.Close(); err != nil {
		t.Fatalf("scope close failed: %v", err)
	}

	if err := diglobal.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestNamedContainerWrappers(t *testing.T) {
	prepareGlobal(t)
	const name = "named-api"

	if diglobal.Container(name) != diglobal.Named(name) {
		t.Fatal("expected Container(name) to return the named container")
	}

	diglobal.SetStructAutoWiring(true, name)
	if !diglobal.StructAutoWiringEnabled(name) {
		t.Fatal("expected named struct auto-wiring to be enabled")
	}
	diglobal.SetStructAutoWiring(false, name)

	builder, err := diglobal.BindIn[*boundValue](name)
	if err != nil {
		t.Fatalf("BindIn failed: %v", err)
	}
	if _, err := builder.ToFactory(func() *boundValue { return &boundValue{ID: "bound"} }); err != nil {
		t.Fatalf("BindIn.ToFactory failed: %v", err)
	}
	diglobal.MustBindIn[*mustBoundValue](name).MustToFactory(func() *mustBoundValue {
		return &mustBoundValue{ID: "must-bound"}
	})

	if err := diglobal.ProvideIn[*service](name, func() *service { return &service{ID: "svc"} }); err != nil {
		t.Fatalf("ProvideIn failed: %v", err)
	}
	diglobal.MustProvideIn[*ctxValue](name, func(ctx context.Context) *ctxValue {
		value, _ := ctx.Value(ctxKey{}).(string)
		return &ctxValue{Value: value}
	}, di.WithLifetime(di.LifetimeTransient))

	if err := diglobal.ProvideNamedIn[*service](name, "named", func() *service { return &service{ID: "named"} }); err != nil {
		t.Fatalf("ProvideNamedIn failed: %v", err)
	}
	diglobal.MustProvideNamedIn[*service](name, "named-2", func() *service { return &service{ID: "named-2"} })

	if err := diglobal.ProvideGroupIn[*groupValue](name, "helpers", func() *groupValue {
		return &groupValue{ID: 1}
	}, di.WithName("one"), di.WithLifetime(di.LifetimeTransient)); err != nil {
		t.Fatalf("ProvideGroupIn failed: %v", err)
	}
	diglobal.MustProvideGroupIn[*groupValue](name, "helpers", func() *groupValue {
		return &groupValue{ID: 2}
	}, di.WithName("two"), di.WithLifetime(di.LifetimeTransient))

	if err := diglobal.ProvideAsIn[*aliasImpl, alias](name, func() *aliasImpl { return &aliasImpl{} }, di.WithName("one")); err != nil {
		t.Fatalf("ProvideAsIn failed: %v", err)
	}
	diglobal.MustProvideAsIn[*aliasImpl2, alias](name, func() *aliasImpl2 { return &aliasImpl2{} }, di.WithName("two"))

	if err := diglobal.ProvideIn[*overrideValue](name, func() *overrideValue { return &overrideValue{ID: "base"} }); err != nil {
		t.Fatalf("ProvideIn override target failed: %v", err)
	}

	ctx := context.WithValue(context.Background(), ctxKey{}, "req-456")
	if value, err := diglobal.ResolveIn[*service](name); err != nil || value.ID != "svc" {
		t.Fatalf("ResolveIn failed: %#v %v", value, err)
	}
	if value, err := diglobal.ResolveInContext[*ctxValue](ctx, name); err != nil || value.Value != "req-456" {
		t.Fatalf("ResolveInContext failed: %#v %v", value, err)
	}
	if value := diglobal.MustResolveIn[*service](name); value.ID != "svc" {
		t.Fatalf("MustResolveIn failed: %#v", value)
	}

	if value, err := diglobal.ResolveNamedIn[*service](name, "named"); err != nil || value.ID != "named" {
		t.Fatalf("ResolveNamedIn failed: %#v %v", value, err)
	}
	if value, err := diglobal.ResolveNamedInContext[*service](ctx, name, "named-2"); err != nil || value.ID != "named-2" {
		t.Fatalf("ResolveNamedInContext failed: %#v %v", value, err)
	}
	if value := diglobal.MustResolveNamedIn[*service](name, "named"); value.ID != "named" {
		t.Fatalf("MustResolveNamedIn failed: %#v", value)
	}

	if values, err := diglobal.ResolveGroupIn[*groupValue](name, "helpers"); err != nil || len(values) != 2 {
		t.Fatalf("ResolveGroupIn failed: %#v %v", values, err)
	}
	if values, err := diglobal.ResolveGroupInContext[*groupValue](ctx, name, "helpers"); err != nil || len(values) != 2 {
		t.Fatalf("ResolveGroupInContext failed: %#v %v", values, err)
	}
	if values := diglobal.MustResolveGroupIn[*groupValue](name, "helpers"); len(values) != 2 {
		t.Fatalf("MustResolveGroupIn failed: %#v", values)
	}

	if values, err := diglobal.ResolveImplementationsIn[alias](name); err != nil || len(values) != 2 {
		t.Fatalf("ResolveImplementationsIn failed: %#v %v", values, err)
	}
	if values, err := diglobal.ResolveImplementationsInContext[alias](ctx, name); err != nil || len(values) != 2 {
		t.Fatalf("ResolveImplementationsInContext failed: %#v %v", values, err)
	}
	if values := diglobal.MustResolveImplementationsIn[alias](name); len(values) != 2 {
		t.Fatalf("MustResolveImplementationsIn failed: %#v", values)
	}

	if err := diglobal.InvokeIn(name, func(s *service) error {
		if s.ID != "svc" {
			t.Fatalf("unexpected invoke service: %#v", s)
		}
		return nil
	}); err != nil {
		t.Fatalf("InvokeIn failed: %v", err)
	}
	if err := diglobal.InvokeInContext(ctx, name, func(injected context.Context, v *ctxValue) error {
		if injected.Value(ctxKey{}) != "req-456" || v.Value != "req-456" {
			t.Fatalf("unexpected invoke-in-context values: %#v %#v", injected, v)
		}
		return nil
	}); err != nil {
		t.Fatalf("InvokeInContext failed: %v", err)
	}

	injector := diglobal.Injector(name)
	if injector == nil {
		t.Fatal("expected named injector")
	}
	if _, err := injector.Call(func(s *service) {}); err != nil {
		t.Fatalf("named injector call failed: %v", err)
	}

	var resolveEvents atomic.Int32
	diglobal.SetInstrumentation(di.Instrumentation{
		OnResolve: func(di.ResolveEvent) { resolveEvents.Add(1) },
	}, name)
	diglobal.ResetMetrics(name)
	if _, err := diglobal.ResolveIn[*service](name); err != nil {
		t.Fatalf("ResolveIn for metrics failed: %v", err)
	}
	if diglobal.Metrics(name).Resolutions == 0 || resolveEvents.Load() == 0 {
		t.Fatalf("expected named metrics/instrumentation to record resolution, got %#v count=%d", diglobal.Metrics(name), resolveEvents.Load())
	}
	diglobal.ResetMetrics(name)
	diglobal.SetInstrumentation(di.Instrumentation{}, name)

	restore, err := diglobal.OverrideIn[*overrideValue](name, func() (*overrideValue, error) {
		return &overrideValue{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("OverrideIn failed: %v", err)
	}
	if value, err := diglobal.ResolveIn[*overrideValue](name); err != nil || value.ID != "override" {
		t.Fatalf("OverrideIn resolve failed: %#v %v", value, err)
	}
	restore()
	restore = diglobal.MustOverrideIn[*overrideValue](name, func() (*overrideValue, error) {
		return &overrideValue{ID: "must-override"}, nil
	})
	if value, err := diglobal.ResolveIn[*overrideValue](name); err != nil || value.ID != "must-override" {
		t.Fatalf("MustOverrideIn resolve failed: %#v %v", value, err)
	}
	restore()

	scope, err := diglobal.NewScope(name)
	if err != nil {
		t.Fatalf("NewScope(name) failed: %v", err)
	}
	mustScope := diglobal.MustNewScope(name)
	_ = mustScope.Close()
	if _, err := di.ResolveInScope[*service](scope); err != nil {
		t.Fatalf("ResolveInScope(named) failed: %v", err)
	}
	if err := scope.Close(); err != nil {
		t.Fatalf("named scope close failed: %v", err)
	}

	if err := diglobal.Close(name); err != nil {
		t.Fatalf("Close(name) failed: %v", err)
	}
}
