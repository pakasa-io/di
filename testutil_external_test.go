package di_test

import (
	"errors"
	"sync/atomic"
	"testing"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
	testutil "github.com/pakasa-io/di/testutil"
)

type fixtureService struct {
	ID string
}

type fixtureGroupValue struct {
	ID int
}

type fixtureAlias interface {
	AliasID() string
}

type fixtureAliasImpl struct{}

func (*fixtureAliasImpl) AliasID() string {
	return "alias"
}

func TestTestutilFixtureHelpers(t *testing.T) {
	testutil.Reset(t)
	fixture := testutil.ScopedFixture(t)

	testutil.MustProvideNamed[*fixtureService](t, fixture.Name, "named", func() *fixtureService {
		return &fixtureService{ID: "named"}
	})
	testutil.MustProvideGroup[*fixtureGroupValue](t, fixture.Name, "helpers", func() *fixtureGroupValue {
		return &fixtureGroupValue{ID: 9}
	}, di.WithLifetime(di.LifetimeTransient))
	testutil.MustProvideAs[*fixtureAliasImpl, fixtureAlias](t, fixture.Name, func() *fixtureAliasImpl {
		return &fixtureAliasImpl{}
	})

	value, err := diglobal.ResolveNamedIn[*fixtureService](fixture.Name, "named")
	if err != nil {
		t.Fatalf("ResolveNamedIn failed: %v", err)
	}
	if value.ID != "named" {
		t.Fatalf("unexpected named value: %#v", value)
	}

	groupValues, err := diglobal.ResolveGroupIn[*fixtureGroupValue](fixture.Name, "helpers")
	if err != nil {
		t.Fatalf("ResolveGroupIn failed: %v", err)
	}
	if len(groupValues) != 1 || groupValues[0].ID != 9 {
		t.Fatalf("unexpected group values: %#v", groupValues)
	}

	impls, err := diglobal.ResolveImplementationsIn[fixtureAlias](fixture.Name)
	if err != nil {
		t.Fatalf("ResolveImplementationsIn failed: %v", err)
	}
	if len(impls) != 1 || impls[0].AliasID() != "alias" {
		t.Fatalf("unexpected implementation values: %#v", impls)
	}

	var resolveEvents atomic.Int32
	testutil.Instrument(t, fixture.Container, di.Instrumentation{
		OnResolve: func(event di.ResolveEvent) {
			if event.Err == nil {
				resolveEvents.Add(1)
			}
		},
	})

	if _, err := diglobal.ResolveNamedIn[*fixtureService](fixture.Name, "named"); err != nil {
		t.Fatalf("ResolveNamedIn failed: %v", err)
	}
	if resolveEvents.Load() == 0 {
		t.Fatal("expected fixture instrumentation to observe resolution")
	}
}

func TestGlobalContainersRecreateAfterClose(t *testing.T) {
	testutil.Reset(t)

	default1 := diglobal.Default()
	if err := default1.Close(); err != nil {
		t.Fatalf("close default: %v", err)
	}
	default2 := diglobal.Default()
	if default1 == default2 {
		t.Fatal("expected Default to recreate a closed container")
	}
	if default2.IsClosed() {
		t.Fatal("expected recreated default container to be open")
	}

	named1 := diglobal.Named("recreate")
	if err := named1.Close(); err != nil {
		t.Fatalf("close named: %v", err)
	}
	named2 := diglobal.Named("recreate")
	if named1 == named2 {
		t.Fatal("expected Named to recreate a closed container")
	}
	if named2.IsClosed() {
		t.Fatal("expected recreated named container to be open")
	}
	if _, err := di.ResolveFrom[*fixtureService](named2); !errors.Is(err, di.ErrBindingNotFound) {
		t.Fatalf("expected recreated named container to be fresh, got %v", err)
	}
}

func TestTestutilOverrideTargetsProvidedContainer(t *testing.T) {
	testutil.Reset(t)
	fixture := testutil.FixtureFor(t)

	if err := di.ProvideTo[*fixtureService](diglobal.Default(), func() *fixtureService {
		return &fixtureService{ID: "default"}
	}); err != nil {
		t.Fatalf("provide default binding: %v", err)
	}
	testutil.MustProvide[*fixtureService](t, fixture.Name, func() *fixtureService {
		return &fixtureService{ID: "fixture"}
	})

	testutil.Override[*fixtureService](t, fixture.Container, func() (*fixtureService, error) {
		return &fixtureService{ID: "override"}, nil
	})

	fixtureValue, err := di.ResolveFrom[*fixtureService](fixture.Container)
	if err != nil {
		t.Fatalf("resolve fixture service: %v", err)
	}
	if fixtureValue.ID != "override" {
		t.Fatalf("expected override to apply to fixture container, got %#v", fixtureValue)
	}

	defaultValue, err := di.ResolveFrom[*fixtureService](diglobal.Default())
	if err != nil {
		t.Fatalf("resolve default service: %v", err)
	}
	if defaultValue.ID != "default" {
		t.Fatalf("expected default container to remain unaffected, got %#v", defaultValue)
	}
}
