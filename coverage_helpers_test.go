package di

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

type scopeAlias interface {
	AliasID() string
}

type scopeAliasImpl struct{}

func (*scopeAliasImpl) AliasID() string { return "scope-alias" }

func TestScopeMustWrappersAndMetadata(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService { return &testService{ID: "svc"} })
	MustProvideNamedTo[*testService](container, "named", func() *testService { return &testService{ID: "named"} })
	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithName("one"), WithLifetime(LifetimeTransient))
	MustProvideAsTo[*scopeAliasImpl, scopeAlias](container, func() *scopeAliasImpl { return &scopeAliasImpl{} })

	scope := container.MustNewScope()
	defer scope.Close()

	if value := MustResolveInScope[*testService](scope); value.ID != "svc" {
		t.Fatalf("unexpected MustResolveInScope value: %#v", value)
	}
	if value, err := ResolveNamedInScopeContext[*testService](context.Background(), scope, "named"); err != nil || value.ID != "named" {
		t.Fatalf("unexpected ResolveNamedInScopeContext result: %#v %v", value, err)
	}
	if value := MustResolveNamedInScope[*testService](scope, "named"); value.ID != "named" {
		t.Fatalf("unexpected MustResolveNamedInScope value: %#v", value)
	}
	if values := MustResolveGroupInScope[*testGroupValue](scope, "helpers"); len(values) != 1 {
		t.Fatalf("unexpected MustResolveGroupInScope values: %#v", values)
	}
	if values := MustResolveImplementationsInScope[scopeAlias](scope); len(values) != 1 || values[0].AliasID() != "scope-alias" {
		t.Fatalf("unexpected MustResolveImplementationsInScope values: %#v", values)
	}

	if bindings := scope.ListBindings(); len(bindings) < 4 {
		t.Fatalf("unexpected scope list bindings: %#v", bindings)
	}
	if info := scope.DescribeContainer(); info.BindingCount < 4 || info.Closed {
		t.Fatalf("unexpected scope container info: %#v", info)
	}

	scope.SetStructAutoWiring(true)
	if !scope.StructAutoWiringEnabled() {
		t.Fatal("expected scope struct auto wiring to be enabled")
	}
	scope.SetStructAutoWiring(false)
}

func TestResolutionStateAndCloseExtractionHelpers(t *testing.T) {
	state := newResolutionState(context.WithValue(context.Background(), testContextKey{}, "trace"))
	if err := state.enter("root"); err != nil {
		t.Fatalf("enter failed: %v", err)
	}
	if err := state.enter("root"); !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected duplicate enter to fail with circular dependency, got %v", err)
	}
	if ctx := state.context(); ctx.Value(testContextKey{}) != "trace" {
		t.Fatalf("unexpected state context: %#v", ctx)
	}
	if trace := state.trace(); len(trace) != 1 || trace[0] != "root" {
		t.Fatalf("unexpected state trace: %#v", trace)
	}
	state.leave("root")

	if fn := extractCloseFunc(reflect.Value{}); fn != nil {
		t.Fatal("expected invalid reflect.Value to produce no close func")
	}

	var canceled atomic.Int32
	cancel := func() { canceled.Add(1) }
	closeFn := extractCloseFunc(reflect.ValueOf(cancel))
	if closeFn == nil {
		t.Fatal("expected cancel func extraction")
	}
	if err := closeFn(); err != nil {
		t.Fatalf("cancel close func failed: %v", err)
	}
	if canceled.Load() != 1 {
		t.Fatalf("expected cancel func to be invoked once, got %d", canceled.Load())
	}
}

type fluentAlias interface {
	AliasID() string
}

type fluentAliasImpl struct{}

func (*fluentAliasImpl) AliasID() string { return "fluent" }

func TestAdditionalRootWrappersForCoverage(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustBindTo[*testService](container).
		ToTransient().
		ToScoped().
		ToSingleton().
		WithHooks(&LifecycleHooks{}).
		MustToFactory(func() *testService { return &testService{ID: "svc"} })
	MustProvideNamedTo[*testService](container, "named", func() *testService { return &testService{ID: "named"} })
	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithName("one"), WithLifetime(LifetimeTransient))
	MustProvideAsTo[*fluentAliasImpl, fluentAlias](container, func() *fluentAliasImpl { return &fluentAliasImpl{} })

	if value := MustResolveFrom[*testService](container); value.ID != "svc" {
		t.Fatalf("unexpected MustResolveFrom value: %#v", value)
	}
	if value := MustResolveNamedFrom[*testService](container, "named"); value.ID != "named" {
		t.Fatalf("unexpected MustResolveNamedFrom value: %#v", value)
	}

	ctx := context.Background()
	if values, err := ResolveGroupFromContext[*testGroupValue](ctx, container, "helpers"); err != nil || len(values) != 1 {
		t.Fatalf("unexpected ResolveGroupFromContext values: %#v %v", values, err)
	}
	if values := MustResolveGroupFrom[*testGroupValue](container, "helpers"); len(values) != 1 {
		t.Fatalf("unexpected MustResolveGroupFrom values: %#v", values)
	}
	if values, err := ResolveImplementationsFromContext[fluentAlias](ctx, container); err != nil || len(values) != 1 {
		t.Fatalf("unexpected ResolveImplementationsFromContext values: %#v %v", values, err)
	}
	if values := MustResolveImplementationsFrom[fluentAlias](container); len(values) != 1 {
		t.Fatalf("unexpected MustResolveImplementationsFrom values: %#v", values)
	}

	explanation, err := ExplainNamedFrom[*testService](container, "named")
	if err != nil {
		t.Fatalf("ExplainNamedFrom failed: %v", err)
	}
	if text := explanation.String(); text == "" {
		t.Fatal("expected explanation String output")
	}
	if _, err := container.ExplainBinding(getType[*testService](), "named"); err != nil {
		t.Fatalf("ExplainBinding failed: %v", err)
	}

	scope := container.MustNewScope()
	defer scope.Close()
	if _, err := scope.Injector().Call(func(s *testService) {}); err != nil {
		t.Fatalf("scope injector call failed: %v", err)
	}
}
