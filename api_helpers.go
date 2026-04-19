package di

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func validateAggregateElementType(typ reflect.Type) error {
	if inner, ok := optionalInnerType(typ); ok {
		return newError(
			ErrUnsupportedAPIShape.Code,
			fmt.Sprintf("aggregate resolution does not support Optional[%s]; resolve %s aggregates directly", inner, inner),
			nil,
		)
	}
	return nil
}

func resolveAggregateSelection[T any](c *Container, sc *scope, selection aggregateSelection, cfg *config) ([]T, error) {
	return resolveAggregateSelectionContext[T](context.Background(), c, sc, selection, cfg)
}

func resolveAggregateSelectionContext[T any](ctx context.Context, c *Container, sc *scope, selection aggregateSelection, cfg *config) ([]T, error) {
	if err := c.ensureOpen(sc); err != nil {
		return nil, err
	}
	requestedType := getType[T]()
	if err := validateAggregateElementType(requestedType); err != nil {
		return nil, err
	}
	if len(selection.refs) == 0 {
		return nil, newError(ErrorCodeNoBindingsFound, "no bindings found for aggregate query", nil)
	}

	values := make([]T, 0, len(selection.refs))
	state := newResolutionState(ctx)
	for _, ref := range selection.refs {
		overrideKeys := make([]string, 0, 1)
		if selection.interfaceLookup {
			overrideKeys = append(overrideKeys, cacheKey(requestedType, ref.binding.name))
		}
		value, err := c.resolver().resolveAggregateBinding(ref, sc, state, overrideKeys...)
		if err != nil {
			return nil, err
		}
		values = append(values, value.Interface().(T))
	}

	return values, nil
}

func resolveGroupFromContainer[T any](c *Container, sc *scope, group string, cfg *config) ([]T, error) {
	selection := c.resolver().selectAggregateBindings(aggregateQuery{
		mode:          aggregateQueryGroup,
		selector:      group,
		requestedType: getType[T](),
		name:          cfg.name,
	})
	if len(selection.refs) == 0 {
		return nil, newError(ErrorCodeNoBindingsFound, fmt.Sprintf("no bindings found for group: %s", group), nil)
	}
	return resolveAggregateSelection[T](c, sc, selection, cfg)
}

func resolveImplementationsFromContainer[T any](c *Container, sc *scope, cfg *config) ([]T, error) {
	interfaceType := getType[T]()
	if err := validateAggregateElementType(interfaceType); err != nil {
		return nil, err
	}
	if interfaceType.Kind() != reflect.Interface {
		return nil, newError(ErrInvalidOption.Code, fmt.Sprintf("ResolveImplementations requires an interface type, got %s", interfaceType), nil)
	}
	selection := c.resolver().selectAggregateBindings(aggregateQuery{
		mode:          aggregateQueryInterface,
		requestedType: interfaceType,
		name:          cfg.name,
	})
	if len(selection.refs) == 0 {
		return nil, newError(ErrorCodeNoBindingsFound, fmt.Sprintf("no bindings found for interface: %s", interfaceType), nil)
	}
	return resolveAggregateSelection[T](c, sc, selection, cfg)
}

// ResolveNamedFrom resolves the named binding for type T from the provided container.
func ResolveNamedFrom[T any](c *Container, name string, opts ...Option) (T, error) {
	return ResolveNamedFromContext[T](context.Background(), c, name, opts...)
}

// ResolveNamedFromContext resolves the named binding for type T from the provided container and injects the provided context.
func ResolveNamedFromContext[T any](ctx context.Context, c *Container, name string, opts ...Option) (T, error) {
	return ResolveFromContext[T](ctx, c, append(opts, WithName(name))...)
}

// MustResolveFrom resolves type T from the provided container and panics on error.
func MustResolveFrom[T any](c *Container, opts ...Option) T {
	value, err := ResolveFrom[T](c, opts...)
	if err != nil {
		panic(err)
	}
	return value
}

// MustResolveNamedFrom resolves the named binding for type T from the provided container and panics on error.
func MustResolveNamedFrom[T any](c *Container, name string, opts ...Option) T {
	value, err := ResolveNamedFrom[T](c, name, opts...)
	if err != nil {
		panic(err)
	}
	return value
}

// ResolveGroupFrom resolves all bindings registered in the named group from the provided container.
func ResolveGroupFrom[T any](c *Container, group string, opts ...Option) ([]T, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	return resolveGroupFromContainer[T](c, c.scope, group, cfg)
}

// ResolveGroupFromContext resolves all bindings registered in the named group from the provided container and injects the provided context.
func ResolveGroupFromContext[T any](ctx context.Context, c *Container, group string, opts ...Option) ([]T, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	selection := c.resolver().selectAggregateBindings(aggregateQuery{
		mode:          aggregateQueryGroup,
		selector:      group,
		requestedType: getType[T](),
		name:          cfg.name,
	})
	if len(selection.refs) == 0 {
		return nil, newError(ErrorCodeNoBindingsFound, fmt.Sprintf("no bindings found for group: %s", group), nil)
	}
	return resolveAggregateSelectionContext[T](ctx, c, c.scope, selection, cfg)
}

// MustResolveGroupFrom resolves a group from the provided container and panics on error.
func MustResolveGroupFrom[T any](c *Container, group string, opts ...Option) []T {
	values, err := ResolveGroupFrom[T](c, group, opts...)
	if err != nil {
		panic(err)
	}
	return values
}

// ResolveImplementationsFrom resolves all bindings registered for interface type T from the provided container.
func ResolveImplementationsFrom[T any](c *Container, opts ...Option) ([]T, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	return resolveImplementationsFromContainer[T](c, c.scope, cfg)
}

// ResolveImplementationsFromContext resolves all bindings registered for interface type T from the provided container and injects the provided context.
func ResolveImplementationsFromContext[T any](ctx context.Context, c *Container, opts ...Option) ([]T, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	interfaceType := getType[T]()
	if interfaceType.Kind() != reflect.Interface {
		return nil, newError(ErrInvalidOption.Code, fmt.Sprintf("ResolveImplementations requires an interface type, got %s", interfaceType), nil)
	}
	selection := c.resolver().selectAggregateBindings(aggregateQuery{
		mode:          aggregateQueryInterface,
		requestedType: interfaceType,
		name:          cfg.name,
	})
	if len(selection.refs) == 0 {
		return nil, newError(ErrorCodeNoBindingsFound, fmt.Sprintf("no bindings found for interface: %s", interfaceType), nil)
	}
	return resolveAggregateSelectionContext[T](ctx, c, c.scope, selection, cfg)
}

// MustResolveImplementationsFrom resolves interface implementations from the provided container and panics on error.
func MustResolveImplementationsFrom[T any](c *Container, opts ...Option) []T {
	values, err := ResolveImplementationsFrom[T](c, opts...)
	if err != nil {
		panic(err)
	}
	return values
}

// HasNamedInContainer reports whether the named binding for type T is resolvable from the provided container.
func HasNamedInContainer[T any](c *Container, name string, opts ...Option) bool {
	return HasInContainer[T](c, append(opts, WithName(name))...)
}

// DescribeNamedInContainer returns binding metadata for the named binding of type T from the provided container.
func DescribeNamedInContainer[T any](c *Container, name string, opts ...Option) (*BindingInfo, error) {
	return DescribeInContainer[T](c, append(opts, WithName(name))...)
}

// ProvideNamedTo registers a named factory for type T in the provided container.
func ProvideNamedTo[T any](c *Container, name string, factory any, opts ...Option) error {
	return ProvideTo[T](c, factory, append(opts, WithName(name))...)
}

// MustProvideNamedTo registers a named factory for type T in the provided container and panics on error.
func MustProvideNamedTo[T any](c *Container, name string, factory any, opts ...Option) {
	if err := ProvideNamedTo[T](c, name, factory, opts...); err != nil {
		panic(err)
	}
}

// ProvideGroupTo registers a group member factory for type T in the provided container.
func ProvideGroupTo[T any](c *Container, group string, factory any, opts ...Option) error {
	return ProvideTo[T](c, factory, append(opts, WithGroup(group))...)
}

// MustProvideGroupTo registers a group member factory for type T in the provided container and panics on error.
func MustProvideGroupTo[T any](c *Container, group string, factory any, opts ...Option) {
	if err := ProvideGroupTo[T](c, group, factory, opts...); err != nil {
		panic(err)
	}
}

// ProvideAsTo registers a factory for concrete type T and aliases it to interface type I in the provided container.
func ProvideAsTo[T any, I any](c *Container, factory any, opts ...Option) error {
	alias, err := withInterfaceTypeE(getType[I]())
	if err != nil {
		return err
	}
	return ProvideTo[T](c, factory, append(opts, alias)...)
}

// MustProvideAsTo registers a factory for concrete type T and aliases it to interface type I, panicking on error.
func MustProvideAsTo[T any, I any](c *Container, factory any, opts ...Option) {
	if err := ProvideAsTo[T, I](c, factory, opts...); err != nil {
		panic(err)
	}
}

// ResolutionExplanation describes the binding or override selected for a resolution request.
type ResolutionExplanation struct {
	Requested    Dependency
	Container    *Container
	Owner        *Container
	Key          string
	Optional     bool
	Override     bool
	AutoWired    bool
	Lifetime     Lifetime
	HasFactory   bool
	Missing      bool
	Dependencies []ResolutionExplanation
}

func (e ResolutionExplanation) String() string {
	var b strings.Builder
	e.writeTo(&b, 0)
	return b.String()
}

func (e ResolutionExplanation) writeTo(b *strings.Builder, depth int) {
	indent := strings.Repeat("  ", depth)
	label := e.Key
	if label == "" {
		label = cacheKey(e.Requested.Type, e.Requested.Name)
	}

	status := "binding"
	switch {
	case e.Optional && e.Missing:
		status = "optional-missing"
	case e.Optional && e.Override:
		status = "optional-override"
	case e.Optional && e.AutoWired:
		status = "optional-autowired"
	case e.Optional:
		status = "optional"
	case e.Override:
		status = "override"
	case e.AutoWired:
		status = "autowired"
	case e.Missing:
		status = "missing"
	}

	fmt.Fprintf(b, "%s- %s (%s)", indent, label, status)
	if e.Owner != nil {
		fmt.Fprintf(b, " owner=%p", e.Owner)
	}
	if e.Lifetime != LifetimeUnknown && !e.Override && !e.Missing {
		fmt.Fprintf(b, " lifetime=%s", e.Lifetime)
	}
	b.WriteByte('\n')
	for _, dep := range e.Dependencies {
		dep.writeTo(b, depth+1)
	}
}

// ExplainBinding returns the selected binding or override for the provided type and optional name.
func (c *Container) ExplainBinding(typ reflect.Type, name string) (*ResolutionExplanation, error) {
	explanation, err := c.introspection().explain(typ, name)
	if err != nil {
		return nil, err
	}
	return &explanation, nil
}

// ExplainFrom returns the selected binding or override for type T from the provided container.
func ExplainFrom[T any](c *Container, opts ...Option) (*ResolutionExplanation, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	explanation, err := c.introspection().explain(getType[T](), cfg.name)
	if err != nil {
		return nil, err
	}
	return &explanation, nil
}

// ExplainNamedFrom returns the selected binding or override for the named binding of type T from the provided container.
func ExplainNamedFrom[T any](c *Container, name string, opts ...Option) (*ResolutionExplanation, error) {
	return ExplainFrom[T](c, append(opts, WithName(name))...)
}

// ExplainBinding returns the selected binding or override for the provided type and optional name within this scope.
func (s *Scope) ExplainBinding(typ reflect.Type, name string) (*ResolutionExplanation, error) {
	if s == nil || s.container == nil {
		return nil, newError(ErrorCodeNilScope, "scope is nil", nil)
	}
	return s.container.ExplainBinding(typ, name)
}
