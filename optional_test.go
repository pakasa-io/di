package di

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestResolveOptionalReturnsMissingWithoutError(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	value, err := ResolveFrom[Optional[*testService]](container)
	if err != nil {
		t.Fatalf("Resolve optional failed: %v", err)
	}
	if value.OK || value.Value != nil {
		t.Fatalf("expected missing optional binding, got %#v", value)
	}
}

func TestResolveOptionalBuiltinsWrapCorrectly(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	ctx := context.WithValue(context.Background(), testContextKey{}, "req-123")

	ctxValue, err := ResolveFromContext[Optional[context.Context]](ctx, container)
	if err != nil {
		t.Fatalf("ResolveContext optional context failed: %v", err)
	}
	if !ctxValue.OK || ctxValue.Value == nil || ctxValue.Value.Value(testContextKey{}) != "req-123" {
		t.Fatalf("unexpected optional context value: %#v", ctxValue)
	}

	containerValue, err := ResolveFrom[Optional[*Container]](container)
	if err != nil {
		t.Fatalf("Resolve optional container failed: %v", err)
	}
	if !containerValue.OK || containerValue.Value != container {
		t.Fatalf("unexpected optional container value: %#v", containerValue)
	}
}

func TestOptionalDependencyDoesNotFailWhenBindingIsMissing(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testOptionalDependent](container, func(service Optional[*testService]) *testOptionalDependent {
		return &testOptionalDependent{Service: service}
	})

	value, err := ResolveFrom[*testOptionalDependent](container)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if value.Service.OK || value.Service.Value != nil {
		t.Fatalf("expected optional dependency to be absent without error, got %#v", value.Service)
	}
	if err := container.Validate(); err != nil {
		t.Fatalf("expected optional missing dependency to validate cleanly, got %v", err)
	}

	explanation, err := ExplainFrom[*testOptionalDependent](container)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if len(explanation.Dependencies) != 1 || !explanation.Dependencies[0].Optional || !explanation.Dependencies[0].Missing {
		t.Fatalf("expected optional missing explanation, got %#v", explanation.Dependencies)
	}

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	foundOptionalMissing := false
	for _, node := range graph.Nodes {
		if node.Missing && node.Optional && node.Key == cacheKey(getType[*testService](), "") {
			foundOptionalMissing = true
			break
		}
	}
	if !foundOptionalMissing {
		t.Fatalf("expected graph to include optional missing dependency, got %#v", graph.Nodes)
	}

	dump, err := container.DumpGraph()
	if err != nil {
		t.Fatalf("DumpGraph failed: %v", err)
	}
	if !strings.Contains(dump, "optional-missing:"+cacheKey(getType[*testService](), "")) {
		t.Fatalf("expected graph dump to include optional missing dependency, got %q", dump)
	}
}

func TestOptionalDependencyStillFailsForRealResolutionErrors(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func(dep *testDependent) *testService {
		return &testService{ID: dep.Service.ID}
	})
	MustProvideTo[*testOptionalBrokenConsumer](container, func(service Optional[*testService]) *testOptionalBrokenConsumer {
		return &testOptionalBrokenConsumer{Service: service}
	})

	if _, err := ResolveFrom[*testOptionalBrokenConsumer](container); err == nil {
		t.Fatal("expected optional dependency to fail when the binding exists but cannot be constructed")
	}

	if err := container.Validate(); err == nil {
		t.Fatal("expected validation to fail for an invalid optional dependency graph")
	}
}

func TestOptionalFillStructFields(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "default"}
	})
	MustProvideNamedTo[*testService](container, "named", func() *testService {
		return &testService{ID: "named"}
	})

	var target optionalFillTarget
	if err := container.Injector().FillStruct(&target); err != nil {
		t.Fatalf("FillStruct failed: %v", err)
	}

	if !target.Default.OK || target.Default.Value == nil || target.Default.Value.ID != "default" {
		t.Fatalf("unexpected default optional field: %#v", target.Default)
	}
	if !target.Named.OK || target.Named.Value == nil || target.Named.Value.ID != "named" {
		t.Fatalf("unexpected named optional field: %#v", target.Named)
	}
	if target.Missing.OK || target.Missing.Value != nil {
		t.Fatalf("expected missing optional field to remain absent, got %#v", target.Missing)
	}
}

func TestOptionalFillStructBuiltins(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	ctx := context.WithValue(context.Background(), testContextKey{}, "req-123")

	var target optionalBuiltinFillTarget
	if err := container.Injector().FillStructContext(ctx, &target); err != nil {
		t.Fatalf("FillStructContext failed: %v", err)
	}

	if !target.Ctx.OK || target.Ctx.Value == nil || target.Ctx.Value.Value(testContextKey{}) != "req-123" {
		t.Fatalf("unexpected optional context field: %#v", target.Ctx)
	}
	if !target.Container.OK || target.Container.Value != container {
		t.Fatalf("unexpected optional container field: %#v", target.Container)
	}
}

func TestOptionalWrapperCannotBeRegisteredOrOverridden(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	if err := ProvideTo[Optional[*testService]](container, func() Optional[*testService] {
		return Optional[*testService]{Value: &testService{ID: "wrapped"}, OK: true}
	}); err == nil {
		t.Fatal("expected Provide to reject Optional[T] registration")
	}

	if _, err := OverrideInContainer[Optional[*testService]](container, func() (Optional[*testService], error) {
		return Optional[*testService]{Value: &testService{ID: "wrapped"}, OK: true}, nil
	}); err == nil {
		t.Fatal("expected Override to reject Optional[T] overrides")
	}
}

func TestAggregateAPIsRejectOptionalElementTypes(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideGroupTo[*testService](container, "services", func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	if _, err := ResolveGroupFrom[Optional[*testService]](container, "services"); !errors.Is(err, ErrUnsupportedAPIShape) {
		t.Fatalf("expected ResolveGroup Optional[T] to fail with ErrUnsupportedAPIShape, got %v", err)
	}

	if _, err := ResolveImplementationsFrom[Optional[testAlias]](container); !errors.Is(err, ErrUnsupportedAPIShape) {
		t.Fatalf("expected ResolveImplementations Optional[T] to fail with ErrUnsupportedAPIShape, got %v", err)
	}
}
