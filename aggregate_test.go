package di

import (
	"errors"
	"testing"
)

func TestResolveAllFromChildInheritsParentGroups(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideTo[*testGroupValue](parent, func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithGroup("shared"), WithName("parent"), WithLifetime(LifetimeTransient))

	child := MustNewOverlayContainer(parent)
	childBinding := &BindingBuilder[*testGroupValue]{
		container: child,
		binding: child.bindType(
			getType[*testGroupValue](),
			WithGroup("shared"),
			WithName("child"),
			WithLifetime(LifetimeTransient),
		),
	}
	childBinding.MustToFactory(func() *testGroupValue {
		return &testGroupValue{ID: 2}
	})

	values, err := resolveGroupFromContainer[*testGroupValue](child, child.scope, "shared", &config{})
	if err != nil {
		t.Fatalf("child ResolveAll failed: %v", err)
	}
	if len(values) != 2 || values[0].ID != 2 || values[1].ID != 1 {
		t.Fatalf("expected child then parent group bindings, got %#v", values)
	}
}

func TestResolveAllFromChildInheritsParentInterfaceAliases(t *testing.T) {
	prepareTest(t)

	parent := newTestContainer()
	MustProvideAsTo[*testAliasImpl, testAlias](parent, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	child := MustNewOverlayContainer(parent)
	childBinding := &BindingBuilder[*testAliasImpl2]{
		container: child,
		binding: child.bindType(
			getType[*testAliasImpl2](),
			MustWithInterface((*testAlias)(nil)),
			WithName("child"),
			WithLifetime(LifetimeTransient),
		),
	}
	childBinding.MustToFactory(func() *testAliasImpl2 {
		return &testAliasImpl2{}
	})

	values, err := resolveImplementationsFromContainer[testAlias](child, child.scope, &config{})
	if err != nil {
		t.Fatalf("child ResolveAll for interface aliases failed: %v", err)
	}
	if len(values) != 2 || values[0].AliasID() != "alias-2" || values[1].AliasID() != "alias" {
		t.Fatalf("expected child then parent interface aliases, got %#v", values)
	}
}

func TestResolveAllUsesInterfaceOverride(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	})

	restore, err := OverrideInContainer[testAlias](container, func() (testAlias, error) {
		return &testAliasImpl2{}, nil
	})
	if err != nil {
		t.Fatalf("OverrideE failed: %v", err)
	}
	defer restore()

	values, err := ResolveImplementationsFrom[testAlias](container)
	if err != nil {
		t.Fatalf("ResolveAll failed: %v", err)
	}
	if len(values) != 1 || values[0].AliasID() != "alias-2" {
		t.Fatalf("expected interface override to apply to ResolveAll, got %#v", values)
	}
}

func TestResolveAllFiltersGroupMembersByName(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 1}
	}, WithName("one"), WithLifetime(LifetimeTransient))
	MustProvideGroupTo[*testGroupValue](container, "helpers", func() *testGroupValue {
		return &testGroupValue{ID: 2}
	}, WithName("two"), WithLifetime(LifetimeTransient))

	values, err := ResolveGroupFrom[*testGroupValue](container, "helpers", WithName("two"))
	if err != nil {
		t.Fatalf("ResolveGroup with name filter failed: %v", err)
	}
	if len(values) != 1 || values[0].ID != 2 {
		t.Fatalf("expected named group filter to keep only the matching member, got %#v", values)
	}

	_, err = ResolveGroupFrom[*testGroupValue](container, "helpers", WithName("missing"))
	var diErr *Error
	if !errors.As(err, &diErr) || diErr.Code != ErrorCodeNoBindingsFound {
		t.Fatalf("expected missing named group filter to report no bindings found, got %v", err)
	}
}

type overrideAlias struct {
	id string
}

func (a *overrideAlias) AliasID() string { return a.id }

func TestResolveAllNamedInterfaceOverrideOnlyShadowsMatchingAlias(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideAsTo[*testAliasImpl, testAlias](container, func() *testAliasImpl {
		return &testAliasImpl{}
	}, WithName("one"))
	MustProvideAsTo[*testAliasImpl2, testAlias](container, func() *testAliasImpl2 {
		return &testAliasImpl2{}
	}, WithName("two"))

	restore, err := OverrideInContainer[testAlias](container, func() (testAlias, error) {
		return &overrideAlias{id: "override-two"}, nil
	}, WithName("two"))
	if err != nil {
		t.Fatalf("Override failed: %v", err)
	}
	defer restore()

	filtered, err := ResolveImplementationsFrom[testAlias](container, WithName("two"))
	if err != nil {
		t.Fatalf("ResolveImplementations with name filter failed: %v", err)
	}
	if len(filtered) != 1 || filtered[0].AliasID() != "override-two" {
		t.Fatalf("expected named interface override to replace only the matching alias, got %#v", filtered)
	}

	values, err := ResolveImplementationsFrom[testAlias](container)
	if err != nil {
		t.Fatalf("ResolveImplementations failed: %v", err)
	}
	if len(values) != 2 || values[0].AliasID() != "alias" || values[1].AliasID() != "override-two" {
		t.Fatalf("expected unmatched aliases to keep their binding while the named alias uses the override, got %#v", values)
	}
}
