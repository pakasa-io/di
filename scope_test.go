package di

import (
	"errors"
	"testing"
)

func TestNewScopeAPI(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})

	scope := container.MustNewScope()
	if scope == nil || scope.ResolverContainer() == nil {
		t.Fatal("expected NewScope to return a scope wrapper with a container")
	}
	if scope.IsClosed() {
		t.Fatal("expected a fresh scope to be open")
	}

	value, err := ResolveInScope[*testService](scope)
	if err != nil {
		t.Fatalf("ResolveInScope failed: %v", err)
	}
	if value.ID != "svc" {
		t.Fatalf("unexpected ResolveInScope value: %#v", value)
	}

	if err := scope.Invoke(func(s *testService) error {
		if s.ID != "svc" {
			t.Fatalf("unexpected scope invocation service: %s", s.ID)
		}
		return nil
	}); err != nil {
		t.Fatalf("Scope.Invoke failed: %v", err)
	}

	child := scope.MustNewScope()
	if child == nil || child.ResolverContainer() == nil {
		t.Fatal("expected nested scope to be created")
	}

	if err := scope.Close(); err != nil {
		t.Fatalf("Scope.Close failed: %v", err)
	}
	if !scope.IsClosed() {
		t.Fatal("expected scope to report closed after Close")
	}
	if _, err := ResolveInScope[*testService](scope); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected ResolveInScope after close to fail with ErrScopeClosed, got %v", err)
	}
	if err := scope.ValidateBindings(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected ValidateBindings after close to fail with ErrScopeClosed, got %v", err)
	}
}

func TestNewInjectorUsesDefaultScope(t *testing.T) {
	prepareTest(t)

	container := newTestContainer()
	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})

	injector := NewInjector(container)
	if injector == nil {
		t.Fatal("expected NewInjector to return an injector")
	}

	if _, err := injector.Call(func(s *testService) {
		if s.ID != "svc" {
			t.Fatalf("unexpected injected service: %s", s.ID)
		}
	}); err != nil {
		t.Fatalf("NewInjector.Call failed: %v", err)
	}

	scope := container.MustNewScope()
	scopeInjector := NewScopeInjector(scope)
	if scopeInjector == nil {
		t.Fatal("expected NewScopeInjector to return an injector")
	}
	if _, err := scopeInjector.Call(func(s *testService) {
		if s.ID != "svc" {
			t.Fatalf("unexpected scope-injected service: %s", s.ID)
		}
	}); err != nil {
		t.Fatalf("NewScopeInjector.Call failed: %v", err)
	}
}

func TestScopeResolverContainerReturnsBaseContainer(t *testing.T) {
	prepareTest(t)

	base := newTestContainer()
	scope := base.MustNewScope()
	defer scope.Close()

	if scope.ResolverContainer() != base {
		t.Fatalf("expected scope to resolve against its base container, got %p want %p", scope.ResolverContainer(), base)
	}
}

func TestOverlayContainerKeepsRegistrationsIndependentFromBaseContainer(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testService](parent, func() *testService {
		return &testService{ID: "svc"}
	})

	overlay := MustNewOverlayContainer(parent)
	bindInContainer[*testDependent](overlay).MustToFactory(func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	value, err := resolveFromContainer[*testDependent](overlay)
	if err != nil {
		t.Fatalf("overlay resolve failed: %v", err)
	}
	if value.Service == nil || value.Service.ID != "svc" {
		t.Fatalf("expected overlay binding to inherit parent registrations, got %#v", value)
	}

	if _, err := resolveFromContainer[*testDependent](parent); !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("expected parent container to remain unaware of overlay-only registration, got %v", err)
	}
}

func TestNewOverlayContainerRejectsNilAndClosedParents(t *testing.T) {
	prepareTest(t)

	if _, err := NewOverlayContainer(nil); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected nil parent to fail with ErrInvalidOption, got %v", err)
	}

	parent := newTestContainer()
	if err := parent.Close(); err != nil {
		t.Fatalf("close parent: %v", err)
	}

	if _, err := NewOverlayContainer(parent); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected closed parent to fail with ErrScopeClosed, got %v", err)
	}
}
