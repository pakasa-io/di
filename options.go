package di

import (
	"reflect"

	diModel "github.com/pakasa-io/di/internal/model"
)

// Lifetime controls how instances are cached by the container.
type Lifetime = diModel.Lifetime

// Lifetime constants describe singleton, transient, and scoped caching behavior.
const (
	LifetimeUnknown   = diModel.LifetimeUnknown
	LifetimeSingleton = diModel.LifetimeSingleton
	LifetimeTransient = diModel.LifetimeTransient
	LifetimeScoped    = diModel.LifetimeScoped
)

// Option configures binding or resolution behavior
type Option interface {
	apply(cfg *config)
}

type config struct {
	name         string
	group        string
	interfaceFor reflect.Type
	dependencies []Dependency
	lifetime     Lifetime
	lifetimeSet  bool
	hooks        *LifecycleHooks
	tags         map[string]string
}

// Dep declares an unnamed dependency on type T for use with WithDeps.
func Dep[T any]() Dependency {
	return normalizeDependency(Dependency{Type: getType[T]()})
}

// Named declares a named dependency on type T for use with WithDeps.
func Named[T any](name string) Dependency {
	return normalizeDependency(Dependency{Type: getType[T](), Name: name})
}

// optionFunc is a function adapter for Option
type optionFunc func(cfg *config)

func (f optionFunc) apply(cfg *config) { f(cfg) }

// WithName names a binding for retrieval by name
func WithName(name string) Option {
	return optionFunc(func(cfg *config) {
		cfg.name = name
	})
}

// WithGroup adds binding to a group
func WithGroup(group string) Option {
	return optionFunc(func(cfg *config) {
		cfg.group = group
	})
}

func withInterfaceTypeE(ifaceType reflect.Type) (Option, error) {
	if ifaceType == nil || ifaceType.Kind() != reflect.Interface {
		return nil, newError(ErrInvalidOption.Code, "WithInterface requires a pointer to an interface type", nil)
	}

	return optionFunc(func(cfg *config) {
		cfg.interfaceFor = ifaceType
	}), nil
}

// WithInterface specifies that this binding implements an interface and returns an error on invalid input.
func WithInterface(iface any) (Option, error) {
	if iface == nil {
		return nil, newError(ErrInvalidOption.Code, "WithInterface requires a non-nil pointer to an interface type", nil)
	}

	ifaceType := reflect.TypeOf(iface)
	if ifaceType.Kind() != reflect.Ptr || ifaceType.Elem().Kind() != reflect.Interface {
		return nil, newError(ErrInvalidOption.Code, "WithInterface requires a pointer to an interface type", nil)
	}

	return withInterfaceTypeE(ifaceType.Elem())
}

// MustWithInterface specifies that this binding implements an interface and panics on invalid input.
func MustWithInterface(iface any) Option {
	opt, err := WithInterface(iface)
	if err != nil {
		panic(err)
	}
	return opt
}

// WithDeps specifies explicit dependencies for a factory.
func WithDeps(deps ...Dependency) Option {
	return optionFunc(func(cfg *config) {
		cfg.dependencies = make([]Dependency, len(deps))
		for i, dep := range deps {
			cfg.dependencies[i] = normalizeDependency(dep)
		}
	})
}

// WithLifetime sets the lifetime of a binding
func WithLifetime(lifetime Lifetime) Option {
	return optionFunc(func(cfg *config) {
		cfg.lifetime = lifetime
		cfg.lifetimeSet = true
	})
}

// WithPreConstruct adds a pre-construct hook
func WithPreConstruct(hook func(any) error) Option {
	return optionFunc(func(cfg *config) {
		if cfg.hooks == nil {
			cfg.hooks = &LifecycleHooks{}
		}
		cfg.hooks.PreConstruct = hook
	})
}

// WithPostConstruct adds a post-construct hook
func WithPostConstruct(hook func(any) error) Option {
	return optionFunc(func(cfg *config) {
		if cfg.hooks == nil {
			cfg.hooks = &LifecycleHooks{}
		}
		cfg.hooks.PostConstruct = hook
	})
}

// WithCloseFunc adds a close function
func WithCloseFunc(closeFn CloseFunc) Option {
	return optionFunc(func(cfg *config) {
		if cfg.hooks == nil {
			cfg.hooks = &LifecycleHooks{}
		}
		cfg.hooks.CloseFunc = closeFn
	})
}

func newConfig(opts ...Option) (*config, error) {
	cfg := &config{}
	for _, opt := range opts {
		if isNilOption(opt) {
			return nil, newError(ErrInvalidOption.Code, "option cannot be nil", nil)
		}
		opt.apply(cfg)
	}
	return cfg, nil
}

func isNilOption(opt Option) bool {
	if opt == nil {
		return true
	}

	value := reflect.ValueOf(opt)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
