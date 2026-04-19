package di

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type nilOptionPtr struct{}

func (*nilOptionPtr) apply(cfg *config) {}

type countingOption struct {
	count *int
}

func (o countingOption) apply(cfg *config) {
	*o.count++
}

func TestNewContainerCreatesDistinctRoots(t *testing.T) {
	prepareTest(t)

	first := NewContainer()
	second := NewContainer()
	if first == nil || second == nil {
		t.Fatal("expected NewContainer to return a container")
	}
	if first == second {
		t.Fatal("expected NewContainer to return distinct root containers")
	}
}

func TestInvokeInjectorAndResolveAllResolveDependencies(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideGroupTo[*testDependent](container, "deps", func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	if err := InvokeOn(container, func(s *testService) {
		if s.ID != "svc" {
			t.Fatalf("unexpected service ID: %s", s.ID)
		}
	}); err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if _, err := container.Injector().Call(func(s *testService) {}); err != nil {
		t.Fatalf("Injector.Call failed: %v", err)
	}

	values, err := ResolveGroupFrom[*testDependent](container, "deps")
	if err != nil {
		t.Fatalf("ResolveAll failed: %v", err)
	}
	if len(values) != 1 || values[0].Service == nil || values[0].Service.ID != "svc" {
		t.Fatalf("unexpected ResolveAll result: %#v", values)
	}
}

func TestNilOptionsReturnErrInvalidOption(t *testing.T) {
	prepareTest(t)

	var (
		nilInterface Option
		nilPointer   *nilOptionPtr
		typedNil     Option = nilPointer
	)

	for _, opt := range []Option{nilInterface, typedNil} {
		container := newTestContainer()
		if _, err := BindTo[*testService](container, opt); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("expected Bind to reject nil option with ErrInvalidOption, got %v", err)
		}
		if _, err := ResolveFrom[*testService](container, opt); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("expected Resolve to reject nil option with ErrInvalidOption, got %v", err)
		}
		if _, err := DescribeInContainer[*testService](container, opt); !errors.Is(err, ErrInvalidOption) {
			t.Fatalf("expected Describe to reject nil option with ErrInvalidOption, got %v", err)
		}
		if HasInContainer[*testService](container, opt) {
			t.Fatal("expected Has to return false for nil options")
		}
		if NewInjector(container) == nil {
			t.Fatal("expected NewInjector to return an injector for a valid container")
		}
	}
}

func TestOptionsAreAppliedOnce(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	registrationCount := 0
	if _, err := BindTo[*testService](container, countingOption{count: &registrationCount}); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if registrationCount != 1 {
		t.Fatalf("expected Bind options to apply once, got %d", registrationCount)
	}

	MustProvideGroupTo[*testGroupValue](container, "counted", func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithLifetime(LifetimeTransient))

	aggregateCount := 0
	if _, err := ResolveGroupFrom[*testGroupValue](container, "counted", countingOption{count: &aggregateCount}); err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}
	if aggregateCount != 1 {
		t.Fatalf("expected ResolveGroup options to apply once, got %d", aggregateCount)
	}
}

func TestWithLifetimeRejectsUnknown(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	_, err := BindTo[*testService](container, WithLifetime(LifetimeUnknown))
	if err == nil {
		t.Fatal("expected Bind to reject LifetimeUnknown")
	}

	var diErr *Error
	if !errors.As(err, &diErr) || diErr.Code != ErrorCodeInvalidLifetime {
		t.Fatalf("expected invalid lifetime error, got %v", err)
	}

	if err := ProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	}, WithLifetime(LifetimeUnknown)); err == nil {
		t.Fatal("expected Provide to reject LifetimeUnknown")
	}

	if _, err := ResolveFrom[*testService](container); !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("expected failed registration to leave no binding behind, got %v", err)
	}
}

func TestConcurrentResolve(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	var wg sync.WaitGroup
	errs := make(chan error, 32)

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			value, err := ResolveFrom[*testDependent](container)
			if err != nil {
				errs <- err
				return
			}
			if value.Service == nil || value.Service.ID != "svc" {
				errs <- errors.New("invalid resolved dependency graph")
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent resolve failed: %v", err)
	}
}

func TestValidateConcurrentWithResolve(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	var wg sync.WaitGroup
	errs := make(chan error, 64)

	for i := 0; i < 32; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if _, err := ResolveFrom[*testDependent](container); err != nil {
				errs <- err
			}
		}()
		go func() {
			defer wg.Done()
			if err := container.Validate(); err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent validate/resolve failed: %v", err)
	}
}

func TestResolveInterfaceAlias(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	value, err := ResolveFrom[testAlias](container)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if value == nil || value.AliasID() != "alias" {
		t.Fatalf("unexpected alias resolution result: %#v", value)
	}
}

func TestFillStructInjectsFields(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "default"}
	})
	MustProvideNamedTo[*testService](container, "named", func() *testService {
		return &testService{ID: "named"}
	})

	var target fillTarget
	if err := container.Injector().FillStruct(&target); err != nil {
		t.Fatalf("FillStruct failed: %v", err)
	}

	if target.Default == nil || target.Default.ID != "default" {
		t.Fatalf("unexpected default field: %#v", target.Default)
	}
	if target.Named == nil || target.Named.ID != "named" {
		t.Fatalf("unexpected named field: %#v", target.Named)
	}
	if target.Container != container {
		t.Fatal("expected FillStruct to inject the current container")
	}
}

func TestContextAwareResolutionInvokeAndFillStruct(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "ctx"}
	})
	MustProvideTo[*testContextValue](container, func(ctx context.Context) *testContextValue {
		value, _ := ctx.Value(testContextKey{}).(string)
		return &testContextValue{Value: value}
	})

	ctx := context.WithValue(context.Background(), testContextKey{}, "req-123")

	value, err := ResolveFromContext[*testContextValue](ctx, container)
	if err != nil {
		t.Fatalf("ResolveContext failed: %v", err)
	}
	if value.Value != "req-123" {
		t.Fatalf("unexpected context-derived value: %#v", value)
	}

	if err := InvokeOnContext(ctx, container, func(injected context.Context, service *testService) error {
		if injected.Value(testContextKey{}) != "req-123" {
			t.Fatalf("unexpected injected context: %#v", injected)
		}
		if service.ID != "ctx" {
			t.Fatalf("unexpected injected service: %#v", service)
		}
		return nil
	}); err != nil {
		t.Fatalf("InvokeContext failed: %v", err)
	}

	var target contextFillTarget
	if err := container.Injector().FillStructContext(ctx, &target); err != nil {
		t.Fatalf("FillStructContext failed: %v", err)
	}
	if target.Ctx.Value(testContextKey{}) != "req-123" {
		t.Fatalf("unexpected filled context: %#v", target.Ctx)
	}
	if target.Service == nil || target.Service.ID != "ctx" {
		t.Fatalf("unexpected filled service: %#v", target.Service)
	}
}

func TestValidateRejectsSingletonContextDependency(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testContextValue](container, func(ctx context.Context) *testContextValue {
		value, _ := ctx.Value(testContextKey{}).(string)
		return &testContextValue{Value: value}
	})

	err := container.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail for singleton context dependency")
	}
	if !strings.Contains(err.Error(), "singleton depends on context.Context") {
		t.Fatalf("expected context lifetime validation error, got %v", err)
	}
}

func TestResolveContextRejectsClosedContainer(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	if err := container.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	_, err := ResolveFromContext[context.Context](context.WithValue(context.Background(), testContextKey{}, "req-123"), container)
	if !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected ResolveContext after Close to fail with ErrScopeClosed, got %v", err)
	}
}

func TestLifecycleHooksRun(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	var hookPre atomic.Int32
	var hookPost atomic.Int32
	var ifacePre atomic.Int32
	var ifacePost atomic.Int32

	MustProvideTo[*testLifecycle](container, func() *testLifecycle {
		return &testLifecycle{
			preCount:  &ifacePre,
			postCount: &ifacePost,
		}
	},
		WithPreConstruct(func(any) error {
			hookPre.Add(1)
			return nil
		}),
		WithPostConstruct(func(any) error {
			hookPost.Add(1)
			return nil
		}),
	)

	if _, err := ResolveFrom[*testLifecycle](container); err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if hookPre.Load() != 1 || hookPost.Load() != 1 {
		t.Fatalf("expected hooks to run once, got pre=%d post=%d", hookPre.Load(), hookPost.Load())
	}
	if ifacePre.Load() != 1 || ifacePost.Load() != 1 {
		t.Fatalf("expected lifecycle interfaces to run once, got pre=%d post=%d", ifacePre.Load(), ifacePost.Load())
	}
}

func TestInvokeReturnsCallbackError(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	want := errors.New("boom")
	err := InvokeOn(container, func() error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected Invoke to return callback error, got %v", err)
	}
}
