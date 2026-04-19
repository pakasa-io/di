package testutil_test

import (
	"sync/atomic"
	"testing"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
	testutil "github.com/pakasa-io/di/testutil"
)

type service struct {
	ID string
}

type groupValue struct {
	ID int
}

type alias interface {
	AliasID() string
}

type aliasImpl struct{}

func (*aliasImpl) AliasID() string { return "alias" }

func TestFixtureHelpersAndOverride(t *testing.T) {
	testutil.Reset(t)

	name1, container1 := testutil.Container(t)
	name2, _ := testutil.Container(t)
	if name1 == name2 || container1 == nil {
		t.Fatalf("expected unique named containers, got %q and %q", name1, name2)
	}

	fixture := testutil.FixtureFor(t)
	testutil.MustProvide[*service](t, fixture.Name, func() *service {
		return &service{ID: "svc"}
	})
	testutil.MustProvideNamed[*service](t, fixture.Name, "named", func() *service {
		return &service{ID: "named"}
	})
	testutil.MustProvideGroup[*groupValue](t, fixture.Name, "helpers", func() *groupValue {
		return &groupValue{ID: 1}
	}, di.WithName("one"), di.WithLifetime(di.LifetimeTransient))
	testutil.MustProvideAs[*aliasImpl, alias](t, fixture.Name, func() *aliasImpl {
		return &aliasImpl{}
	})

	var resolveEvents atomic.Int32
	testutil.Instrument(t, fixture.Container, di.Instrumentation{
		OnResolve: func(di.ResolveEvent) { resolveEvents.Add(1) },
	})

	testutil.Override[*service](t, fixture.Container, func() (*service, error) {
		return &service{ID: "override"}, nil
	})

	if value, err := di.ResolveFrom[*service](fixture.Container); err != nil || value.ID != "override" {
		t.Fatalf("unexpected overridden service: %#v %v", value, err)
	}
	if value, err := diglobal.ResolveNamedIn[*service](fixture.Name, "named"); err != nil || value.ID != "named" {
		t.Fatalf("unexpected named service: %#v %v", value, err)
	}
	if values, err := diglobal.ResolveGroupIn[*groupValue](fixture.Name, "helpers"); err != nil || len(values) != 1 {
		t.Fatalf("unexpected group values: %#v %v", values, err)
	}
	if values, err := diglobal.ResolveImplementationsIn[alias](fixture.Name); err != nil || len(values) != 1 {
		t.Fatalf("unexpected implementation values: %#v %v", values, err)
	}
	if resolveEvents.Load() == 0 {
		t.Fatal("expected instrumentation to observe resolution")
	}
}

func TestScopeHelpers(t *testing.T) {
	testutil.Reset(t)

	scope := testutil.Scope(t)
	if scope == nil {
		t.Fatal("expected Scope helper to return a scope")
	}
	scoped := testutil.ScopedFixture(t)
	if scoped.Scope == nil || scoped.Container == nil {
		t.Fatalf("unexpected scoped fixture: %#v", scoped)
	}
}
