package di

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestIntrospectionAPIs(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testService](parent, func() *testService {
		return &testService{ID: "svc"}
	}, WithName("named"), WithGroup("services"), WithLifetime(LifetimeTransient))

	child := MustNewOverlayContainer(parent)

	if !HasInContainer[*testService](parent, WithName("named")) {
		t.Fatal("expected HasIn to report the named binding")
	}
	if HasInContainer[*testClosable](parent) {
		t.Fatal("expected HasIn to return false for an unbound type")
	}

	info, err := DescribeInContainer[*testService](parent, WithName("named"))
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}
	if info.Name != "named" || info.Group != "services" || info.Lifetime != LifetimeTransient || !info.HasFactory {
		t.Fatalf("unexpected binding info: %#v", info)
	}

	inherited, err := child.DescribeBinding(getType[*testService](), "named")
	if err != nil {
		t.Fatalf("child DescribeBinding failed: %v", err)
	}
	if !inherited.Inherited || inherited.Owner != parent {
		t.Fatalf("expected inherited binding metadata, got %#v", inherited)
	}

	bindings := parent.ListBindings()
	if len(bindings) != 1 || bindings[0].Name != "named" {
		t.Fatalf("unexpected ListBindings result: %#v", bindings)
	}

	containerInfo := parent.DescribeContainer()
	if containerInfo.BindingCount != 1 || containerInfo.ChildCount != 1 || containerInfo.Closed {
		t.Fatalf("unexpected container info: %#v", containerInfo)
	}
}

func TestLowLevelIntrospectionAPIsRejectNilType(t *testing.T) {
	prepareTest(t)

	c := newTestContainer()
	var nilType reflect.Type

	if c.HasBinding(nilType, "") {
		t.Fatal("expected HasBinding(nil) to return false")
	}

	if _, err := c.DescribeBinding(nilType, ""); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected DescribeBinding(nil) to reject invalid input, got %v", err)
	}

	if _, err := c.ExplainBinding(nilType, ""); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("expected ExplainBinding(nil) to reject invalid input, got %v", err)
	}
}

func TestNamedDependencyHelpersResolveFactoryDeps(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	type namedDependent struct {
		Service *testService
	}

	MustProvideNamedTo[*testService](container, "named", func() *testService {
		return &testService{ID: "named"}
	})
	MustProvideTo[*namedDependent](container, func(s *testService) *namedDependent {
		return &namedDependent{Service: s}
	}, WithDeps(Named[*testService]("named")))

	value, err := ResolveFrom[*namedDependent](container)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if value.Service == nil || value.Service.ID != "named" {
		t.Fatalf("expected named dependency injection, got %#v", value)
	}

	info, err := DescribeInContainer[*namedDependent](container)
	if err != nil {
		t.Fatalf("Describe failed: %v", err)
	}
	if len(info.Dependencies) != 1 || info.Dependencies[0].Name != "named" || info.Dependencies[0].Type != getType[*testService]() {
		t.Fatalf("unexpected dependency metadata: %#v", info.Dependencies)
	}
}

func TestDeveloperUXHelpers(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideNamedTo[*testService](container, "named", func() *testService {
		return &testService{ID: "named"}
	})
	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 7}
	}, WithLifetime(LifetimeTransient))
	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	named, err := ResolveNamedFrom[*testService](container, "named")
	if err != nil {
		t.Fatalf("ResolveNamed failed: %v", err)
	}
	if named.ID != "named" {
		t.Fatalf("unexpected named resolution result: %#v", named)
	}
	if !HasNamedInContainer[*testService](container, "named") {
		t.Fatal("expected HasNamed to report true")
	}

	info, err := DescribeNamedInContainer[*testService](container, "named")
	if err != nil {
		t.Fatalf("DescribeNamed failed: %v", err)
	}
	if info.Name != "named" {
		t.Fatalf("unexpected named binding info: %#v", info)
	}

	groupValues, err := ResolveGroupFrom[*testGroupValue](container, "helpers")
	if err != nil {
		t.Fatalf("ResolveGroup failed: %v", err)
	}
	if len(groupValues) != 1 || groupValues[0].ID != 7 {
		t.Fatalf("unexpected group resolution result: %#v", groupValues)
	}

	impls, err := ResolveImplementationsFrom[testAlias](container)
	if err != nil {
		t.Fatalf("ResolveImplementations failed: %v", err)
	}
	if len(impls) != 1 || impls[0].AliasID() != "alias" {
		t.Fatalf("unexpected implementation resolution result: %#v", impls)
	}
}

func TestScopeHelperAPIs(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideNamedTo[*testService](container, "named", func() *testService {
		return &testService{ID: "named"}
	})
	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 11}
	}, WithLifetime(LifetimeTransient))
	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	scope := container.MustNewScope()
	defer scope.Close()

	named, err := ResolveNamedInScope[*testService](scope, "named")
	if err != nil {
		t.Fatalf("ResolveNamedInScope failed: %v", err)
	}
	if named.ID != "named" {
		t.Fatalf("unexpected scoped named resolution result: %#v", named)
	}

	groupValues, err := ResolveGroupInScope[*testGroupValue](scope, "helpers")
	if err != nil {
		t.Fatalf("ResolveGroupInScope failed: %v", err)
	}
	if len(groupValues) != 1 || groupValues[0].ID != 11 {
		t.Fatalf("unexpected scoped group resolution result: %#v", groupValues)
	}

	impls, err := ResolveImplementationsInScope[testAlias](scope)
	if err != nil {
		t.Fatalf("ResolveImplementationsInScope failed: %v", err)
	}
	if len(impls) != 1 || impls[0].AliasID() != "alias" {
		t.Fatalf("unexpected scoped implementation resolution result: %#v", impls)
	}

	explanation, err := ExplainNamedInScope[*testService](scope, "named")
	if err != nil {
		t.Fatalf("ExplainNamedInScope failed: %v", err)
	}
	if explanation.Key != cacheKey(getType[*testService](), "named") {
		t.Fatalf("unexpected explanation key: %#v", explanation)
	}
	if err := scope.ValidateBindings(); err != nil {
		t.Fatalf("scope ValidateBindings failed: %v", err)
	}
}

func TestExplainAndFormatValidation(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	explanation, err := ExplainFrom[*testDependent](container)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if explanation.Key != cacheKey(getType[*testDependent](), "") || len(explanation.Dependencies) != 1 {
		t.Fatalf("unexpected explanation: %#v", explanation)
	}
	if explanation.Dependencies[0].Key != cacheKey(getType[*testService](), "") {
		t.Fatalf("unexpected explanation dependency: %#v", explanation.Dependencies)
	}

	container = newTestContainer()
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	formatted := FormatValidation(container.Validate())
	if !strings.Contains(formatted, "validation failed") || !strings.Contains(formatted, getType[*testDependent]().String()) {
		t.Fatalf("unexpected validation format output: %q", formatted)
	}
	if !strings.Contains(formatted, "graph:") {
		t.Fatalf("expected validation format to include a graph snapshot, got %q", formatted)
	}
}
