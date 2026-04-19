package di

import (
	"reflect"
	"sync"

	internalState "github.com/pakasa-io/di/internal/state"
)

type overrideFunc func(*Container, *scope) (reflect.Value, error)

type containerRegistryState = internalState.RegistryState[*binding]
type containerHierarchyState = internalState.HierarchyState[*Container]
type containerRuntimeState = internalState.RuntimeState[overrideFunc, containerMetrics]

// Container is the main dependency injection container.
type Container struct {
	registryState  containerRegistryState
	hierarchyState containerHierarchyState
	runtimeState   containerRuntimeState
	scope          *scope
	mu             sync.RWMutex
}

func (c *Container) registry() registryComponent {
	return registryComponent{container: c}
}

func (c *Container) resolver() resolverComponent {
	return resolverComponent{container: c}
}

func (c *Container) lifecycle() lifecycleComponent {
	return lifecycleComponent{container: c}
}

func (c *Container) introspection() introspectionComponent {
	return introspectionComponent{container: c}
}

func newContainer(parent *Container, parentScope *scope) *Container {
	c := &Container{
		registryState:  internalState.NewRegistryState[*binding](),
		hierarchyState: internalState.NewHierarchyState(parent),
		runtimeState:   internalState.NewRuntimeState[overrideFunc, containerMetrics](),
	}
	if parent != nil {
		c.runtimeState.StructAutoWire.Store(parent.StructAutoWiringEnabled())
	} else {
		c.runtimeState.StructAutoWire.Store(defaultStructAutoWiringEnabled())
	}

	c.scope = newScope(parentScope)
	if parent != nil {
		parent.lifecycle().addChild(c)
	}

	return c
}

// NewContainer creates a new root container.
func NewContainer() *Container {
	return newContainer(nil, nil)
}

// NewOverlayContainer creates a child container that inherits registrations and overrides from the parent.
// Unlike Scope, an overlay container is a registration/runtime overlay, not a scoped lifetime context.
func NewOverlayContainer(parent *Container) (*Container, error) {
	if parent == nil {
		return nil, newError(ErrInvalidOption.Code, "overlay parent container is nil", nil)
	}
	if err := parent.ensureOpen(parent.scope); err != nil {
		return nil, err
	}
	return newContainer(parent, parent.scope), nil
}

// MustNewOverlayContainer creates a child overlay container and panics on error.
func MustNewOverlayContainer(parent *Container) *Container {
	container, err := NewOverlayContainer(parent)
	if err != nil {
		panic(err)
	}
	return container
}

// Injector returns a dependency injector bound to this container's default scope.
func (c *Container) Injector() *DepInjector {
	return newInjectorWithScope(c, c.scope)
}

// BindingBuilder provides a fluent API for configuring bindings.
type BindingBuilder[T any] struct {
	container *Container
	binding   *binding
}

// ToSingleton configures the binding as a singleton.
func (bb *BindingBuilder[T]) ToSingleton() *BindingBuilder[T] {
	bb.binding.lifetime = LifetimeSingleton
	return bb
}

// ToTransient configures the binding as transient.
func (bb *BindingBuilder[T]) ToTransient() *BindingBuilder[T] {
	bb.binding.lifetime = LifetimeTransient
	return bb
}

// ToScoped configures the binding as scoped.
func (bb *BindingBuilder[T]) ToScoped() *BindingBuilder[T] {
	bb.binding.lifetime = LifetimeScoped
	return bb
}

// ToFactory configures the factory function and returns an error on invalid factories.
func (bb *BindingBuilder[T]) ToFactory(factory any) (*BindingBuilder[T], error) {
	if err := validateFactoryForBinding(bb.binding, factory); err != nil {
		return nil, err
	}

	factoryValue := reflect.ValueOf(factory)
	bb.binding.factory = factoryValue
	bb.binding.factoryPlan = getFunctionPlan(factoryValue.Type())
	return bb, nil
}

// MustToFactory configures the factory function and panics on invalid factories.
func (bb *BindingBuilder[T]) MustToFactory(factory any) *BindingBuilder[T] {
	builder, err := bb.ToFactory(factory)
	if err != nil {
		panic(err)
	}
	return builder
}

// WithHooks configures lifecycle hooks.
func (bb *BindingBuilder[T]) WithHooks(hooks *LifecycleHooks) *BindingBuilder[T] {
	bb.binding.hooks = hooks
	return bb
}
