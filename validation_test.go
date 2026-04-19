package di

import (
	"errors"
	"strings"
	"testing"
)

func TestProvideReturnsErrorWithoutRegisteringBinding(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	err := ProvideTo[*testService](container, func() string {
		return "wrong"
	})
	if err == nil {
		t.Fatal("expected Provide to return an error for an invalid factory")
	}

	if _, err := ResolveFrom[*testService](container); !errors.Is(err, ErrBindingNotFound) {
		t.Fatalf("expected invalid Provide to leave no binding behind, got %v", err)
	}
}

func TestWithInterfaceReturnsErrorForInvalidInput(t *testing.T) {
	if _, err := WithInterface(testAliasImpl{}); err == nil {
		t.Fatal("expected WithInterface to reject non-interface input")
	}
}

func TestValidateReportsMissingDependencyTrace(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	err := container.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail for a missing dependency")
	}

	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || len(validationErr.Issues) == 0 {
		t.Fatalf("expected ValidationError with issues, got %v", err)
	}

	issue := validationErr.Issues[0].Error()
	if !strings.Contains(issue, "missing dependency") {
		t.Fatalf("expected missing dependency issue, got %q", issue)
	}
	if !strings.Contains(issue, getType[*testDependent]().String()) || !strings.Contains(issue, getType[*testService]().String()) {
		t.Fatalf("expected trace to include dependency path, got %q", issue)
	}
}

func TestValidateReportsSingletonScopedLifetimeMismatch(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "scoped"}
	}, WithLifetime(LifetimeScoped))
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	err := container.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail for singleton-to-scoped dependency")
	}
	if !strings.Contains(err.Error(), "singleton depends on scoped dependency") {
		t.Fatalf("expected lifetime mismatch error, got %v", err)
	}
}

func TestValidateReportsDuplicateAliasCollision(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})
	MustProvideAsTo[*testAliasImpl2, testAlias](container, func() *testAliasImpl2 {
		return &testAliasImpl2{}
	})

	err := container.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail for duplicate interface aliases")
	}
	if !errors.Is(err, ErrMultipleBindings) {
		t.Fatalf("expected duplicate alias to report ErrMultipleBindings, got %v", err)
	}
}

func TestResolveErrorIncludesTrace(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	_, err := ResolveFrom[*testDependent](container)
	if err == nil {
		t.Fatal("expected Resolve to fail")
	}
	message := err.Error()
	if !strings.Contains(message, "trace:") {
		t.Fatalf("expected resolution trace in error, got %q", message)
	}
	if !strings.Contains(message, getType[*testDependent]().String()) || !strings.Contains(message, getType[*testService]().String()) {
		t.Fatalf("expected trace path in error, got %q", message)
	}
}

func TestScopeValidateReportsInheritedBindingFailures(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testDependent](parent, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	scope := parent.MustNewScope()
	err := scope.ValidateBindings()
	if err == nil {
		t.Fatal("expected Scope.ValidateBindings to fail for a broken inherited binding")
	}
	if !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("expected missing dependency from inherited binding, got %v", err)
	}
	if !strings.Contains(err.Error(), getType[*testDependent]().String()) || !strings.Contains(err.Error(), getType[*testService]().String()) {
		t.Fatalf("expected inherited validation trace to include dependency path, got %v", err)
	}
}

func TestScopeValidateUsesNearestVisibleOverride(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testDependent](parent, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	sc := parent.MustNewScope()
	restore := MustOverrideInContainer[*testDependent](parent, func() (*testDependent, error) {
		return &testDependent{}, nil
	})
	defer restore()

	if err := sc.ValidateBindings(); err != nil {
		t.Fatalf("expected Scope.ValidateBindings to honor a visible container override, got %v", err)
	}

	value, err := ResolveInScope[*testDependent](sc)
	if err != nil {
		t.Fatalf("expected runtime resolution through the child override to succeed, got %v", err)
	}
	if value == nil {
		t.Fatal("expected override-backed resolution to return a value")
	}
}

func TestScopeValidateIgnoresParentAliasCollisionShadowedByChildOverride(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideAsTo[*testAliasImpl, testAlias](parent, func() *testAliasImpl {
		return &testAliasImpl{}
	}, WithName("shared"))
	MustProvideAsTo[*testAliasImpl2, testAlias](parent, func() *testAliasImpl2 {
		return &testAliasImpl2{}
	}, WithName("shared"))

	if err := parent.Validate(); err == nil {
		t.Fatal("expected parent validation to fail for duplicate alias bindings")
	}

	child := MustNewOverlayContainer(parent)
	restore := MustOverrideInContainer[testAlias](child, func() (testAlias, error) {
		return &overrideAlias{id: "override-shared"}, nil
	}, WithName("shared"))
	defer restore()

	scope := child.MustNewScope()
	defer scope.Close()

	if err := scope.ValidateBindings(); err != nil {
		t.Fatalf("expected scope validation to honor the nearer alias override, got %v", err)
	}

	value, err := ResolveNamedInScope[testAlias](scope, "shared")
	if err != nil {
		t.Fatalf("ResolveNamedInScope failed: %v", err)
	}
	if value.AliasID() != "override-shared" {
		t.Fatalf("expected scope resolution to use the child alias override, got %#v", value)
	}
}
