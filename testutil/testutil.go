package testutil

import (
	"fmt"
	"sync/atomic"
	"testing"

	di "github.com/pakasa-io/di"
	diglobal "github.com/pakasa-io/di/global"
)

var containerSeq atomic.Uint64

// Fixture describes an isolated container and optional scope for tests.
type Fixture struct {
	Name      string
	Container *di.Container
	Scope     *di.Scope
}

func uniqueName(tb testing.TB) string {
	tb.Helper()
	return fmt.Sprintf("testutil-%s-%d", tb.Name(), containerSeq.Add(1))
}

// Reset clears global DI container and reflection cache state for isolated tests.
func Reset(tb testing.TB) {
	tb.Helper()
	if err := diglobal.Reset(); err != nil {
		tb.Fatalf("reset containers: %v", err)
	}
	di.ClearRuntimeCaches()
}

// Container returns an isolated named container and closes it during cleanup.
func Container(tb testing.TB) (string, *di.Container) {
	tb.Helper()

	name := uniqueName(tb)
	container := diglobal.Named(name)
	tb.Cleanup(func() {
		_ = container.Close()
	})

	return name, container
}

// FixtureFor returns an isolated container fixture and closes it during cleanup.
func FixtureFor(tb testing.TB) *Fixture {
	tb.Helper()

	name, container := Container(tb)
	return &Fixture{
		Name:      name,
		Container: container,
	}
}

// Scope returns a scope backed by an isolated named container and closes it during cleanup.
func Scope(tb testing.TB) *di.Scope {
	tb.Helper()

	fixture := FixtureFor(tb)
	scope, err := fixture.Container.NewScope()
	if err != nil {
		tb.Fatalf("new scope: %v", err)
	}
	fixture.Scope = scope
	tb.Cleanup(func() {
		_ = scope.Close()
	})

	return scope
}

// ScopedFixture returns an isolated container fixture with an attached scope.
func ScopedFixture(tb testing.TB) *Fixture {
	tb.Helper()

	fixture := FixtureFor(tb)
	scope, err := fixture.Container.NewScope()
	if err != nil {
		tb.Fatalf("new scope: %v", err)
	}
	fixture.Scope = scope
	tb.Cleanup(func() {
		_ = scope.Close()
	})
	return fixture
}

// Override installs a temporary override on the provided container and restores it during cleanup.
func Override[T any](tb testing.TB, container *di.Container, factory func() (T, error), opts ...di.Option) func() {
	tb.Helper()

	if container == nil {
		tb.Fatal("override: container is nil")
	}

	restore, err := di.OverrideInContainer(container, factory, opts...)
	if err != nil {
		tb.Fatalf("override: %v", err)
	}
	tb.Cleanup(restore)
	return restore
}

// MustProvide registers a binding in the given container and fails the test on error.
func MustProvide[T any](tb testing.TB, containerName string, factory any, opts ...di.Option) {
	tb.Helper()

	if err := diglobal.ProvideIn[T](containerName, factory, opts...); err != nil {
		tb.Fatalf("provide: %v", err)
	}
}

// MustProvideNamed registers a named binding in the given container and fails the test on error.
func MustProvideNamed[T any](tb testing.TB, containerName string, name string, factory any, opts ...di.Option) {
	tb.Helper()

	if err := diglobal.ProvideNamedIn[T](containerName, name, factory, opts...); err != nil {
		tb.Fatalf("provide named: %v", err)
	}
}

// MustProvideGroup registers a group binding in the given container and fails the test on error.
func MustProvideGroup[T any](tb testing.TB, containerName string, group string, factory any, opts ...di.Option) {
	tb.Helper()

	if err := diglobal.ProvideGroupIn[T](containerName, group, factory, opts...); err != nil {
		tb.Fatalf("provide group: %v", err)
	}
}

// MustProvideAs registers an aliased binding in the given container and fails the test on error.
func MustProvideAs[T any, I any](tb testing.TB, containerName string, factory any, opts ...di.Option) {
	tb.Helper()

	if err := diglobal.ProvideAsIn[T, I](containerName, factory, opts...); err != nil {
		tb.Fatalf("provide alias: %v", err)
	}
}

// Instrument sets container instrumentation and resets it during cleanup.
func Instrument(tb testing.TB, container *di.Container, instrumentation di.Instrumentation) {
	tb.Helper()

	container.SetInstrumentation(instrumentation)
	tb.Cleanup(func() {
		container.SetInstrumentation(di.Instrumentation{})
	})
}
