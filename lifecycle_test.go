package di

import (
	"errors"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestCloseStopsResolutionAndClosesSingletons(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	var closed atomic.Int32
	MustProvideTo[*testClosable](container, func() *testClosable {
		return &testClosable{closed: &closed}
	})

	if _, err := ResolveFrom[*testClosable](container); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if err := container.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if closed.Load() != 1 {
		t.Fatalf("expected singleton close to run once, got %d", closed.Load())
	}

	if _, err := ResolveFrom[*testClosable](container); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected Resolve after Close to fail with ErrScopeClosed, got %v", err)
	}
	if err := container.Validate(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected Validate after Close to fail with ErrScopeClosed, got %v", err)
	}
}

func TestClosePropagatesBindingErrors(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	want := errors.New("close failed")
	MustProvideTo[*testService](container, func() *testService {
		return &testService{}
	}, WithCloseFunc(func() error {
		return want
	}))

	err := container.Close()
	if !errors.Is(err, want) {
		t.Fatalf("expected Close to return binding error, got %v", err)
	}
}

func TestParentCloseClosesChildContainer(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	child := MustNewOverlayContainer(parent)

	var closed atomic.Int32
	bindInContainer[*testClosable](child).MustToFactory(func() *testClosable {
		return &testClosable{closed: &closed}
	})

	if _, err := resolveFromContainer[*testClosable](child); err != nil {
		t.Fatalf("child resolve failed: %v", err)
	}
	if err := parent.Close(); err != nil {
		t.Fatalf("parent close failed: %v", err)
	}
	if closed.Load() != 1 {
		t.Fatalf("expected child singleton close to run once, got %d", closed.Load())
	}
	if _, err := resolveFromContainer[*testClosable](child); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected child resolve after parent close to fail with ErrScopeClosed, got %v", err)
	}
}

func TestRebindingRemovesStaleGroupIndexes(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideGroupTo[*testGroupValue](container, "g1", func() *testGroupValue {
		return &testGroupValue{ID: 1}
	})
	MustProvideGroupTo[*testGroupValue](container, "g2", func() *testGroupValue {
		return &testGroupValue{ID: 2}
	}, WithLifetime(LifetimeTransient))

	group1, err := ResolveGroupFrom[*testGroupValue](container, "g1")
	if err == nil {
		t.Fatal("expected ResolveAll(g1) to fail after the stale group entry was removed")
	}
	if len(group1) != 0 {
		t.Fatalf("expected stale g1 binding to be removed, got %#v", group1)
	}

	group2, err := ResolveGroupFrom[*testGroupValue](container, "g2")
	if err != nil {
		t.Fatalf("ResolveAll(g2) failed: %v", err)
	}
	if len(group2) != 1 || group2[0].ID != 2 {
		t.Fatalf("unexpected g2 binding: %#v", group2)
	}
}

func TestParentOverrideAppliesFromChildScope(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testService](parent, func() *testService {
		return &testService{ID: "real"}
	})

	restore := MustOverrideInContainer[*testService](parent, func() (*testService, error) {
		return &testService{ID: "override"}, nil
	})
	defer restore()

	child := MustNewOverlayContainer(parent)
	value, err := resolveFromContainer[*testService](child)
	if err != nil {
		t.Fatalf("child resolve failed: %v", err)
	}
	if value.ID != "override" {
		t.Fatalf("expected child scope to see parent override, got %q", value.ID)
	}
}

func TestParentSingletonUsesParentScopeForScopedDependencies(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	var seq atomic.Int32
	MustProvideTo[*testScopedValue](parent, func() *testScopedValue {
		return &testScopedValue{Seq: seq.Add(1)}
	}, WithLifetime(LifetimeScoped))
	MustProvideTo[*testAppSingleton](parent, func(value *testScopedValue) *testAppSingleton {
		return &testAppSingleton{Scoped: value}
	})

	scope1 := parent.MustNewScope()
	scope2 := parent.MustNewScope()

	app1, err := ResolveInScope[*testAppSingleton](scope1)
	if err != nil {
		t.Fatalf("child1 app resolve failed: %v", err)
	}
	req1, err := ResolveInScope[*testScopedValue](scope1)
	if err != nil {
		t.Fatalf("child1 scoped resolve failed: %v", err)
	}
	app2, err := ResolveInScope[*testAppSingleton](scope2)
	if err != nil {
		t.Fatalf("child2 app resolve failed: %v", err)
	}
	req2, err := ResolveInScope[*testScopedValue](scope2)
	if err != nil {
		t.Fatalf("child2 scoped resolve failed: %v", err)
	}

	if app1 != app2 {
		t.Fatal("expected parent singleton to be shared across child scopes")
	}
	if app1.Scoped != app2.Scoped {
		t.Fatal("expected singleton dependency graph to be cached once")
	}
	if app1.Scoped == req1 || app2.Scoped == req2 {
		t.Fatal("expected parent singleton to avoid capturing a child-scoped dependency instance")
	}
	if req1 == req2 {
		t.Fatal("expected distinct scoped instances per child scope")
	}
	if seq.Load() != 3 {
		t.Fatalf("expected one parent-scoped instance plus two child-scoped instances, got %d", seq.Load())
	}
}

func TestCloseUsesReverseRegistrationOrder(t *testing.T) {
	prepareTest(t)

	container := newTestContainer()
	order := make([]string, 0, 2)

	MustProvideTo[*testCloseZ](container, func() *testCloseZ {
		return &testCloseZ{}
	}, WithCloseFunc(func() error {
		order = append(order, "Z")
		return nil
	}))
	MustProvideTo[*testCloseA](container, func() *testCloseA {
		return &testCloseA{}
	}, WithCloseFunc(func() error {
		order = append(order, "A")
		return nil
	}))

	if err := container.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if len(order) != 2 || order[0] != "A" || order[1] != "Z" {
		t.Fatalf("expected reverse registration close order [A Z], got %v", order)
	}
}

func TestClosedContainerRejectsMutations(t *testing.T) {
	prepareTest(t)

	container := newTestContainer()
	if err := container.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if _, err := BindTo[*testService](container); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected BindIn after Close to fail with ErrScopeClosed, got %v", err)
	}
	if err := ProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	}); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected ProvideIn after Close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := OverrideInContainer[*testService](container, func() (*testService, error) {
		return &testService{ID: "svc"}, nil
	}); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected Override after Close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := container.NewScope(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected NewScope after Close to fail with ErrScopeClosed, got %v", err)
	}
	if err := container.Validate(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected container.Validate after Close to fail with ErrScopeClosed, got %v", err)
	}
}

func TestHasReflectsOverrideAndClosedState(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	if HasInContainer[*testService](container) {
		t.Fatal("expected Has to report false before binding or override")
	}

	restore, err := OverrideInContainer[*testService](container, func() (*testService, error) {
		return &testService{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("OverrideE failed: %v", err)
	}
	defer restore()

	if !HasInContainer[*testService](container) {
		t.Fatal("expected Has to report true when an override is present")
	}

	if err := container.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if HasInContainer[*testService](container) {
		t.Fatal("expected Has to report false after container close")
	}
}

func TestResolveAllFromChildUsesNearestOverride(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testService](parent, func() *testService {
		return &testService{ID: "real"}
	}, WithGroup("services"))

	restore, err := OverrideInContainer[*testService](parent, func() (*testService, error) {
		return &testService{ID: "parent"}, nil
	})
	if err != nil {
		t.Fatalf("parent override failed: %v", err)
	}
	defer restore()

	child := MustNewOverlayContainer(parent)
	child.mu.Lock()
	child.runtimeState.Overrides[cacheKey(getType[*testService](), "")] = func(_ *Container, _ *scope) (reflect.Value, error) {
		return reflect.ValueOf(&testService{ID: "child"}), nil
	}
	child.mu.Unlock()

	values, err := resolveGroupFromContainer[*testService](child, child.scope, "services", &config{})
	if err != nil {
		t.Fatalf("child ResolveAll with overrides failed: %v", err)
	}
	if len(values) != 1 || values[0].ID != "child" {
		t.Fatalf("expected child override to take precedence, got %#v", values)
	}
}
