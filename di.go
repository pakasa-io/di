package di

import (
	"context"
	"fmt"
	"reflect"
)

func requireContainer(c *Container) error {
	if c == nil {
		return newError(ErrInvalidOption.Code, "container is nil", nil)
	}
	return nil
}

func containerConfig(c *Container, opts ...Option) (*config, error) {
	if err := requireContainer(c); err != nil {
		return nil, err
	}
	return newConfig(opts...)
}

// BindTo starts a binding for type T in the provided container.
func BindTo[T any](c *Container, opts ...Option) (*BindingBuilder[T], error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	b, err := c.bindTypeWithConfig(getType[T](), cfg)
	if err != nil {
		return nil, err
	}
	return &BindingBuilder[T]{container: c, binding: b}, nil
}

// MustBindTo starts a binding for type T and panics on error.
func MustBindTo[T any](c *Container, opts ...Option) *BindingBuilder[T] {
	builder, err := BindTo[T](c, opts...)
	if err != nil {
		panic(err)
	}
	return builder
}

// ProvideTo registers a factory for type T in the provided container.
func ProvideTo[T any](c *Container, factory any, opts ...Option) error {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return err
	}
	binding, err := c.bindTypeWithConfig(getType[T](), cfg)
	if err != nil {
		return err
	}
	builder := &BindingBuilder[T]{container: c, binding: binding}
	if _, err = builder.ToFactory(factory); err != nil {
		builder.container.unregisterBinding(builder.binding.key())
		return err
	}
	return nil
}

// MustProvideTo registers a factory for type T and panics on error.
func MustProvideTo[T any](c *Container, factory any, opts ...Option) {
	if err := ProvideTo[T](c, factory, opts...); err != nil {
		panic(err)
	}
}

// ResolveFrom resolves type T from the provided container.
func ResolveFrom[T any](c *Container, opts ...Option) (T, error) {
	return ResolveFromContext[T](context.Background(), c, opts...)
}

// ResolveFromContext resolves type T from the provided container and injects the provided context.
func ResolveFromContext[T any](ctx context.Context, c *Container, opts ...Option) (T, error) {
	var zero T

	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return zero, err
	}
	dep := normalizeDependency(Dependency{Type: getType[T](), Name: cfg.name})
	state := newResolutionState(ctx)

	instance, err := c.resolveDependency(dep, c.scope, state)
	if err != nil {
		return zero, err
	}

	return instance.Interface().(T), nil
}

// InvokeOn calls a function with dependency injection from the provided container.
func InvokeOn(c *Container, fn any) error {
	if err := requireContainer(c); err != nil {
		return err
	}
	return c.Invoke(fn)
}

// InvokeOnContext calls a function with dependency injection from the provided container and injects the provided context.
func InvokeOnContext(ctx context.Context, c *Container, fn any) error {
	if err := requireContainer(c); err != nil {
		return err
	}
	return c.InvokeContext(ctx, fn)
}

// OverrideInContainer installs a runtime override for type T on the provided container.
func OverrideInContainer[T any](c *Container, factory func() (T, error), opts ...Option) (func(), error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	if err := c.ensureOpen(c.scope); err != nil {
		return nil, err
	}

	typ := getType[T]()
	if err := validateRegisteredType(typ); err != nil {
		return nil, err
	}
	key := cacheKey(typ, cfg.name)

	c.mu.Lock()
	originalOverride, hadOverride := c.runtimeState.Overrides[key]
	c.runtimeState.Overrides[key] = func(_ *Container, _ *scope) (reflect.Value, error) {
		instance, err := factory()
		if err != nil {
			return reflect.Value{}, err
		}
		value := reflect.ValueOf(instance)
		if !value.IsValid() {
			return reflect.Zero(typ), nil
		}
		if !value.Type().AssignableTo(typ) {
			return reflect.Value{}, newError(
				ErrInvalidFactory.Code,
				fmt.Sprintf("override for %v returned %v", typ, value.Type()),
				nil,
			)
		}
		return value, nil
	}
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		if hadOverride {
			c.runtimeState.Overrides[key] = originalOverride
		} else {
			delete(c.runtimeState.Overrides, key)
		}
		c.mu.Unlock()
	}, nil
}

// MustOverrideInContainer installs a runtime override for type T and panics on error.
func MustOverrideInContainer[T any](c *Container, factory func() (T, error), opts ...Option) func() {
	restore, err := OverrideInContainer[T](c, factory, opts...)
	if err != nil {
		panic(err)
	}
	return restore
}
