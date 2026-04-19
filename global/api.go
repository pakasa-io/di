package global

import (
	"context"

	di "github.com/pakasa-io/di"
)

// Close closes the selected process-wide container.
func Close(name ...string) error {
	return Container(name...).Close()
}

// Injector returns an injector bound to the selected process-wide container.
func Injector(name ...string) *di.DepInjector {
	return Container(name...).Injector()
}

// NewScope creates a child scope from the selected process-wide container.
func NewScope(name ...string) (*di.Scope, error) {
	return Container(name...).NewScope()
}

// MustNewScope creates a child scope from the selected process-wide container and panics on error.
func MustNewScope(name ...string) *di.Scope {
	scope, err := NewScope(name...)
	if err != nil {
		panic(err)
	}
	return scope
}

// SetStructAutoWiring enables or disables implicit struct auto-wiring for the selected container.
func SetStructAutoWiring(enabled bool, name ...string) {
	Container(name...).SetStructAutoWiring(enabled)
}

// StructAutoWiringEnabled reports whether implicit struct auto-wiring is enabled for the selected container.
func StructAutoWiringEnabled(name ...string) bool {
	return Container(name...).StructAutoWiringEnabled()
}

// SetInstrumentation installs instrumentation callbacks on the selected container.
func SetInstrumentation(instrumentation di.Instrumentation, name ...string) {
	Container(name...).SetInstrumentation(instrumentation)
}

// Metrics returns a cumulative metrics snapshot for the selected container.
func Metrics(name ...string) di.ContainerMetrics {
	return Container(name...).Metrics()
}

// ResetMetrics resets cumulative metrics for the selected container.
func ResetMetrics(name ...string) {
	Container(name...).ResetMetrics()
}

// Bind starts a binding for type T in the default container.
func Bind[T any](opts ...di.Option) (*di.BindingBuilder[T], error) {
	return di.BindTo[T](Default(), opts...)
}

// BindIn starts a binding for type T in the named container.
func BindIn[T any](containerName string, opts ...di.Option) (*di.BindingBuilder[T], error) {
	return di.BindTo[T](Container(containerName), opts...)
}

// MustBind starts a binding for type T in the default container and panics on error.
func MustBind[T any](opts ...di.Option) *di.BindingBuilder[T] {
	return di.MustBindTo[T](Default(), opts...)
}

// MustBindIn starts a binding for type T in the named container and panics on error.
func MustBindIn[T any](containerName string, opts ...di.Option) *di.BindingBuilder[T] {
	return di.MustBindTo[T](Container(containerName), opts...)
}

// Provide registers a factory for type T in the default container.
func Provide[T any](factory any, opts ...di.Option) error {
	return di.ProvideTo[T](Default(), factory, opts...)
}

// ProvideIn registers a factory for type T in the named container.
func ProvideIn[T any](containerName string, factory any, opts ...di.Option) error {
	return di.ProvideTo[T](Container(containerName), factory, opts...)
}

// MustProvide registers a factory for type T in the default container and panics on error.
func MustProvide[T any](factory any, opts ...di.Option) {
	di.MustProvideTo[T](Default(), factory, opts...)
}

// MustProvideIn registers a factory for type T in the named container and panics on error.
func MustProvideIn[T any](containerName string, factory any, opts ...di.Option) {
	di.MustProvideTo[T](Container(containerName), factory, opts...)
}

// Resolve resolves type T from the default container.
func Resolve[T any](opts ...di.Option) (T, error) {
	return di.ResolveFrom[T](Default(), opts...)
}

// ResolveContext resolves type T from the default container and injects the provided context.
func ResolveContext[T any](ctx context.Context, opts ...di.Option) (T, error) {
	return di.ResolveFromContext[T](ctx, Default(), opts...)
}

// ResolveIn resolves type T from the named container.
func ResolveIn[T any](containerName string, opts ...di.Option) (T, error) {
	return di.ResolveFrom[T](Container(containerName), opts...)
}

// ResolveInContext resolves type T from the named container and injects the provided context.
func ResolveInContext[T any](ctx context.Context, containerName string, opts ...di.Option) (T, error) {
	return di.ResolveFromContext[T](ctx, Container(containerName), opts...)
}

// MustResolve resolves type T from the default container and panics on error.
func MustResolve[T any](opts ...di.Option) T {
	return di.MustResolveFrom[T](Default(), opts...)
}

// MustResolveIn resolves type T from the named container and panics on error.
func MustResolveIn[T any](containerName string, opts ...di.Option) T {
	return di.MustResolveFrom[T](Container(containerName), opts...)
}

// ResolveNamed resolves the named binding for type T from the default container.
func ResolveNamed[T any](name string, opts ...di.Option) (T, error) {
	return di.ResolveNamedFrom[T](Default(), name, opts...)
}

// ResolveNamedIn resolves the named binding for type T from the named container.
func ResolveNamedIn[T any](containerName string, name string, opts ...di.Option) (T, error) {
	return di.ResolveNamedFrom[T](Container(containerName), name, opts...)
}

// ResolveNamedContext resolves the named binding for type T from the default container and injects the provided context.
func ResolveNamedContext[T any](ctx context.Context, name string, opts ...di.Option) (T, error) {
	return di.ResolveNamedFromContext[T](ctx, Default(), name, opts...)
}

// ResolveNamedInContext resolves the named binding for type T from the named container and injects the provided context.
func ResolveNamedInContext[T any](ctx context.Context, containerName string, name string, opts ...di.Option) (T, error) {
	return di.ResolveNamedFromContext[T](ctx, Container(containerName), name, opts...)
}

// MustResolveNamed resolves the named binding for type T from the default container and panics on error.
func MustResolveNamed[T any](name string, opts ...di.Option) T {
	return di.MustResolveNamedFrom[T](Default(), name, opts...)
}

// MustResolveNamedIn resolves the named binding for type T from the named container and panics on error.
func MustResolveNamedIn[T any](containerName string, name string, opts ...di.Option) T {
	return di.MustResolveNamedFrom[T](Container(containerName), name, opts...)
}

// ResolveGroup resolves the named group from the default container.
func ResolveGroup[T any](group string, opts ...di.Option) ([]T, error) {
	return di.ResolveGroupFrom[T](Default(), group, opts...)
}

// ResolveGroupIn resolves the named group from the named container.
func ResolveGroupIn[T any](containerName string, group string, opts ...di.Option) ([]T, error) {
	return di.ResolveGroupFrom[T](Container(containerName), group, opts...)
}

// ResolveGroupContext resolves the named group from the default container with the provided context.
func ResolveGroupContext[T any](ctx context.Context, group string, opts ...di.Option) ([]T, error) {
	return di.ResolveGroupFromContext[T](ctx, Default(), group, opts...)
}

// ResolveGroupInContext resolves the named group from the named container with the provided context.
func ResolveGroupInContext[T any](ctx context.Context, containerName string, group string, opts ...di.Option) ([]T, error) {
	return di.ResolveGroupFromContext[T](ctx, Container(containerName), group, opts...)
}

// MustResolveGroup resolves the named group from the default container and panics on error.
func MustResolveGroup[T any](group string, opts ...di.Option) []T {
	return di.MustResolveGroupFrom[T](Default(), group, opts...)
}

// MustResolveGroupIn resolves the named group from the named container and panics on error.
func MustResolveGroupIn[T any](containerName string, group string, opts ...di.Option) []T {
	return di.MustResolveGroupFrom[T](Container(containerName), group, opts...)
}

// ResolveImplementations resolves interface implementations from the default container.
func ResolveImplementations[T any](opts ...di.Option) ([]T, error) {
	return di.ResolveImplementationsFrom[T](Default(), opts...)
}

// ResolveImplementationsIn resolves interface implementations from the named container.
func ResolveImplementationsIn[T any](containerName string, opts ...di.Option) ([]T, error) {
	return di.ResolveImplementationsFrom[T](Container(containerName), opts...)
}

// ResolveImplementationsContext resolves interface implementations from the default container with the provided context.
func ResolveImplementationsContext[T any](ctx context.Context, opts ...di.Option) ([]T, error) {
	return di.ResolveImplementationsFromContext[T](ctx, Default(), opts...)
}

// ResolveImplementationsInContext resolves interface implementations from the named container with the provided context.
func ResolveImplementationsInContext[T any](ctx context.Context, containerName string, opts ...di.Option) ([]T, error) {
	return di.ResolveImplementationsFromContext[T](ctx, Container(containerName), opts...)
}

// MustResolveImplementations resolves interface implementations from the default container and panics on error.
func MustResolveImplementations[T any](opts ...di.Option) []T {
	return di.MustResolveImplementationsFrom[T](Default(), opts...)
}

// MustResolveImplementationsIn resolves interface implementations from the named container and panics on error.
func MustResolveImplementationsIn[T any](containerName string, opts ...di.Option) []T {
	return di.MustResolveImplementationsFrom[T](Container(containerName), opts...)
}

// ProvideNamed registers a named factory in the default container.
func ProvideNamed[T any](name string, factory any, opts ...di.Option) error {
	return di.ProvideNamedTo[T](Default(), name, factory, opts...)
}

// ProvideNamedIn registers a named factory in the named container.
func ProvideNamedIn[T any](containerName string, name string, factory any, opts ...di.Option) error {
	return di.ProvideNamedTo[T](Container(containerName), name, factory, opts...)
}

// MustProvideNamed registers a named factory in the default container and panics on error.
func MustProvideNamed[T any](name string, factory any, opts ...di.Option) {
	di.MustProvideNamedTo[T](Default(), name, factory, opts...)
}

// MustProvideNamedIn registers a named factory in the named container and panics on error.
func MustProvideNamedIn[T any](containerName string, name string, factory any, opts ...di.Option) {
	di.MustProvideNamedTo[T](Container(containerName), name, factory, opts...)
}

// ProvideGroup registers a group member in the default container.
func ProvideGroup[T any](group string, factory any, opts ...di.Option) error {
	return di.ProvideGroupTo[T](Default(), group, factory, opts...)
}

// ProvideGroupIn registers a group member in the named container.
func ProvideGroupIn[T any](containerName string, group string, factory any, opts ...di.Option) error {
	return di.ProvideGroupTo[T](Container(containerName), group, factory, opts...)
}

// MustProvideGroup registers a group member in the default container and panics on error.
func MustProvideGroup[T any](group string, factory any, opts ...di.Option) {
	di.MustProvideGroupTo[T](Default(), group, factory, opts...)
}

// MustProvideGroupIn registers a group member in the named container and panics on error.
func MustProvideGroupIn[T any](containerName string, group string, factory any, opts ...di.Option) {
	di.MustProvideGroupTo[T](Container(containerName), group, factory, opts...)
}

// ProvideAs registers a concrete factory and aliases it to interface I in the default container.
func ProvideAs[T any, I any](factory any, opts ...di.Option) error {
	return di.ProvideAsTo[T, I](Default(), factory, opts...)
}

// ProvideAsIn registers a concrete factory and aliases it to interface I in the named container.
func ProvideAsIn[T any, I any](containerName string, factory any, opts ...di.Option) error {
	return di.ProvideAsTo[T, I](Container(containerName), factory, opts...)
}

// MustProvideAs registers a concrete factory and aliases it to interface I in the default container.
func MustProvideAs[T any, I any](factory any, opts ...di.Option) {
	di.MustProvideAsTo[T, I](Default(), factory, opts...)
}

// MustProvideAsIn registers a concrete factory and aliases it to interface I in the named container.
func MustProvideAsIn[T any, I any](containerName string, factory any, opts ...di.Option) {
	di.MustProvideAsTo[T, I](Container(containerName), factory, opts...)
}

// Invoke calls a function with dependency injection from the default container.
func Invoke(fn any) error {
	return di.InvokeOn(Default(), fn)
}

// InvokeIn calls a function with dependency injection from the named container.
func InvokeIn(containerName string, fn any) error {
	return di.InvokeOn(Container(containerName), fn)
}

// InvokeContext calls a function with dependency injection from the default container and injects the provided context.
func InvokeContext(ctx context.Context, fn any) error {
	return di.InvokeOnContext(ctx, Default(), fn)
}

// InvokeInContext calls a function with dependency injection from the named container and injects the provided context.
func InvokeInContext(ctx context.Context, containerName string, fn any) error {
	return di.InvokeOnContext(ctx, Container(containerName), fn)
}

// Override installs a runtime override in the default container.
func Override[T any](factory func() (T, error), opts ...di.Option) (func(), error) {
	return di.OverrideInContainer[T](Default(), factory, opts...)
}

// OverrideIn installs a runtime override in the named container.
func OverrideIn[T any](containerName string, factory func() (T, error), opts ...di.Option) (func(), error) {
	return di.OverrideInContainer[T](Container(containerName), factory, opts...)
}

// MustOverride installs a runtime override in the default container and panics on error.
func MustOverride[T any](factory func() (T, error), opts ...di.Option) func() {
	return di.MustOverrideInContainer[T](Default(), factory, opts...)
}

// MustOverrideIn installs a runtime override in the named container and panics on error.
func MustOverrideIn[T any](containerName string, factory func() (T, error), opts ...di.Option) func() {
	return di.MustOverrideInContainer[T](Container(containerName), factory, opts...)
}
