package di

import "testing"

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
