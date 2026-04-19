package di

import (
	"context"
	"reflect"

	internalScope "github.com/pakasa-io/di/internal/scope"
)

type scope = internalScope.Scope

func newScope(parent *scope) *scope {
	return internalScope.New(parent, func(value reflect.Value) internalScope.CloseFunc {
		return internalScope.CloseFunc(extractCloseFunc(value))
	})
}

// Scope is a scoped resolution context backed by a dedicated scope runtime.
type Scope struct {
	container *Container
	state     *scope
}

func wrapScope(container *Container, state *scope) *Scope {
	if container == nil || state == nil {
		return nil
	}
	return &Scope{container: container, state: state}
}

// NewScope creates a child scope from this container.
func (c *Container) NewScope() (*Scope, error) {
	if err := c.ensureOpen(c.scope); err != nil {
		return nil, err
	}
	return wrapScope(c, c.scope.CreateChild()), nil
}

// MustNewScope creates a child scope from this container and panics on error.
func (c *Container) MustNewScope() *Scope {
	scope, err := c.NewScope()
	if err != nil {
		panic(err)
	}
	return scope
}

// ResolverContainer returns the container that resolves bindings for this scope.
func (s *Scope) ResolverContainer() *Container {
	if s == nil {
		return nil
	}
	return s.container
}

// Injector returns an injector bound to this scope.
func (s *Scope) Injector() *DepInjector {
	if s == nil || s.container == nil || s.state == nil {
		return nil
	}
	return newInjectorWithScope(s.container, s.state)
}

// Invoke calls fn with dependency injection inside this scope.
func (s *Scope) Invoke(fn any) error {
	return s.InvokeContext(context.Background(), fn)
}

// InvokeContext calls fn with dependency injection inside this scope and injects the provided context.
func (s *Scope) InvokeContext(ctx context.Context, fn any) error {
	if s == nil || s.container == nil || s.state == nil {
		return newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	return s.container.resolver().invokeInScope(ctx, s.state, fn)
}

// Close closes this scope.
func (s *Scope) Close() error {
	if s == nil || s.state == nil {
		return nil
	}
	return s.state.Close()
}

// NewScope creates a nested child scope from this scope.
func (s *Scope) NewScope() (*Scope, error) {
	if s == nil || s.container == nil || s.state == nil {
		return nil, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	if err := s.container.ensureOpen(s.state); err != nil {
		return nil, err
	}
	return wrapScope(s.container, s.state.CreateChild()), nil
}

// MustNewScope creates a nested child scope from this scope and panics on error.
func (s *Scope) MustNewScope() *Scope {
	scope, err := s.NewScope()
	if err != nil {
		panic(err)
	}
	return scope
}

// IsClosed reports whether this scope has been closed.
func (s *Scope) IsClosed() bool {
	if s == nil || s.container == nil || s.state == nil {
		return true
	}
	return s.container.isClosed() || s.state.IsClosed()
}

// ResolveInScope resolves type T from the provided scope.
func ResolveInScope[T any](s *Scope, opts ...Option) (T, error) {
	return ResolveInScopeContext[T](context.Background(), s, opts...)
}

// ResolveInScopeContext resolves type T from the provided scope and injects the provided context.
func ResolveInScopeContext[T any](ctx context.Context, s *Scope, opts ...Option) (T, error) {
	var zero T
	if s == nil || s.container == nil || s.state == nil {
		return zero, newError(ErrorCodeNilScope, "scope is nil", nil)
	}

	cfg, err := newConfig(opts...)
	if err != nil {
		return zero, err
	}
	dep := normalizeDependency(Dependency{Type: getType[T](), Name: cfg.name})
	state := newResolutionState(ctx)
	instance, err := s.container.resolveDependency(dep, s.state, state)
	if err != nil {
		return zero, err
	}
	return instance.Interface().(T), nil
}

// MustResolveInScope resolves type T from the provided scope and panics on error.
func MustResolveInScope[T any](s *Scope, opts ...Option) T {
	value, err := ResolveInScope[T](s, opts...)
	if err != nil {
		panic(err)
	}
	return value
}

// ResolveNamedInScope resolves the named binding for type T from the provided scope.
func ResolveNamedInScope[T any](s *Scope, name string, opts ...Option) (T, error) {
	return ResolveInScope[T](s, append(opts, WithName(name))...)
}

// ResolveNamedInScopeContext resolves the named binding for type T from the provided scope and injects the provided context.
func ResolveNamedInScopeContext[T any](ctx context.Context, s *Scope, name string, opts ...Option) (T, error) {
	return ResolveInScopeContext[T](ctx, s, append(opts, WithName(name))...)
}

// MustResolveNamedInScope resolves the named binding for type T from the provided scope and panics on error.
func MustResolveNamedInScope[T any](s *Scope, name string, opts ...Option) T {
	value, err := ResolveNamedInScope[T](s, name, opts...)
	if err != nil {
		panic(err)
	}
	return value
}

// ResolveGroupInScope resolves all bindings registered in the named group from the provided scope.
func ResolveGroupInScope[T any](s *Scope, group string, opts ...Option) ([]T, error) {
	if s == nil || s.container == nil || s.state == nil {
		return nil, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	return resolveGroupFromContainer[T](s.container, s.state, group, cfg)
}

// MustResolveGroupInScope resolves a named group from the provided scope and panics on error.
func MustResolveGroupInScope[T any](s *Scope, group string, opts ...Option) []T {
	values, err := ResolveGroupInScope[T](s, group, opts...)
	if err != nil {
		panic(err)
	}
	return values
}

// ResolveImplementationsInScope resolves all bindings registered for interface type T from the provided scope.
func ResolveImplementationsInScope[T any](s *Scope, opts ...Option) ([]T, error) {
	if s == nil || s.container == nil || s.state == nil {
		return nil, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	return resolveImplementationsFromContainer[T](s.container, s.state, cfg)
}

// MustResolveImplementationsInScope resolves interface implementations from the provided scope and panics on error.
func MustResolveImplementationsInScope[T any](s *Scope, opts ...Option) []T {
	values, err := ResolveImplementationsInScope[T](s, opts...)
	if err != nil {
		panic(err)
	}
	return values
}

// ExplainInScope returns the selected binding or override for type T from the provided scope.
func ExplainInScope[T any](s *Scope, opts ...Option) (*ResolutionExplanation, error) {
	if s == nil || s.container == nil || s.state == nil {
		return nil, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	explanation, err := s.container.introspection().explainInScope(s.state, getType[T](), cfg.name)
	if err != nil {
		return nil, err
	}
	return &explanation, nil
}

// ExplainNamedInScope returns the selected binding or override for the named binding of type T from the provided scope.
func ExplainNamedInScope[T any](s *Scope, name string, opts ...Option) (*ResolutionExplanation, error) {
	return ExplainInScope[T](s, append(opts, WithName(name))...)
}

// ValidateBindings validates the bindings visible from this scope's container.
func (s *Scope) ValidateBindings() error {
	if s == nil || s.container == nil || s.state == nil {
		return newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	return s.container.validateVisibleInScope(s.state)
}

// ListBindings returns the bindings local to this scope's container.
func (s *Scope) ListBindings() []BindingInfo {
	if s == nil || s.container == nil {
		return nil
	}
	return s.container.ListBindings()
}

// DescribeContainer returns a summary of this scope's container.
func (s *Scope) DescribeContainer() ContainerInfo {
	if s == nil || s.container == nil {
		return ContainerInfo{}
	}
	return s.container.DescribeContainer()
}

// Graph returns the effective dependency graph visible from this scope.
func (s *Scope) Graph() (Graph, error) {
	if s == nil || s.container == nil || s.state == nil {
		return Graph{}, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	return s.container.graphInScope(s.state)
}

// DumpGraph returns a readable text dump for this scope.
func (s *Scope) DumpGraph() (string, error) {
	graph, err := s.Graph()
	if err != nil {
		return "", err
	}
	return graph.String(), nil
}

// DumpGraphDOT returns a Graphviz DOT dump for this scope.
func (s *Scope) DumpGraphDOT() (string, error) {
	graph, err := s.Graph()
	if err != nil {
		return "", err
	}
	return graph.DOT(), nil
}
