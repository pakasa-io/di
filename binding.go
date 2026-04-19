package di

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

// binding holds configuration for a registered type
type binding struct {
	typ           reflect.Type
	factory       reflect.Value
	factoryPlan   functionPlan
	lifetime      Lifetime
	hooks         *LifecycleHooks
	interfaceFor  reflect.Type // If this binding implements an interface
	dependencies  []Dependency
	name          string
	group         string
	instance      atomic.Value // For singleton lifetime
	instanceMutex sync.Mutex   // For singleton thread safety
	tags          map[string]string
}

func newBindingWithConfig(typ reflect.Type, cfg *config) (*binding, error) {
	if err := validateRegisteredType(typ); err != nil {
		return nil, err
	}

	b := &binding{
		typ:      typ,
		lifetime: LifetimeSingleton, // Default lifetime
	}

	if cfg != nil {
		b.applyConfig(cfg)
	}
	if err := validateBindingConfiguration(b); err != nil {
		return nil, err
	}
	return b, nil
}

// newBindingE creates a new binding and validates its static configuration.
// applyConfig applies configuration options to the binding
func (b *binding) applyConfig(cfg *config) {
	if cfg.name != "" {
		b.name = cfg.name
	}
	if cfg.group != "" {
		b.group = cfg.group
	}
	if cfg.interfaceFor != nil {
		b.interfaceFor = cfg.interfaceFor
	}
	if cfg.lifetimeSet {
		b.lifetime = cfg.lifetime
	}
	if cfg.hooks != nil {
		b.hooks = cfg.hooks
	}
	if cfg.dependencies != nil {
		b.dependencies = cloneDependencies(cfg.dependencies)
	}
	if cfg.tags != nil {
		b.tags = cfg.tags
	}
}

func validateBindingConfiguration(b *binding) error {
	if b == nil {
		return newError(ErrInvalidOption.Code, ErrInvalidOption.Message, nil)
	}

	switch b.lifetime {
	case LifetimeSingleton, LifetimeTransient, LifetimeScoped:
	default:
		return newError(ErrorCodeInvalidLifetime, fmt.Sprintf("unknown lifetime: %s", b.lifetime), nil)
	}

	if b.interfaceFor != nil {
		if b.interfaceFor.Kind() != reflect.Interface {
			return newError(ErrInvalidOption.Code, "WithInterface requires a pointer to an interface type", nil)
		}
		if !b.typ.Implements(b.interfaceFor) {
			return newError(
				ErrInvalidOption.Code,
				fmt.Sprintf("type %v does not implement interface %v", b.typ, b.interfaceFor),
				nil,
			)
		}
	}

	for _, dep := range b.dependencies {
		dep = normalizeDependency(dep)
		if dep.Type == nil {
			return newError(ErrInvalidOption.Code, "dependency type cannot be nil", nil)
		}
	}

	return nil
}

func validateFactoryForBinding(b *binding, factory any) error {
	if b == nil {
		return ErrInvalidFactory
	}

	factoryType := reflect.TypeOf(factory)
	if factoryType == nil || !isFuncValidFactory(factoryType, b.typ) {
		return newError(
			ErrInvalidFactory.Code,
			fmt.Sprintf("factory for %v must return (T) or (T, error)", b.typ),
			nil,
		)
	}

	if len(b.dependencies) > 0 && factoryType.NumIn() != len(b.dependencies) {
		return newError(
			ErrInvalidOption.Code,
			fmt.Sprintf("factory for %v expects %d parameters, but %d explicit dependencies were provided", b.typ, factoryType.NumIn(), len(b.dependencies)),
			nil,
		)
	}

	return nil
}

func plannedDependenciesForBinding(b *binding) []Dependency {
	if b == nil {
		return nil
	}
	return cloneDependencies(factoryPlanForBinding(b).dependencies)
}

func factoryPlanForBinding(b *binding) functionPlan {
	if b == nil || !b.factory.IsValid() {
		return functionPlan{}
	}

	plan := b.factoryPlan
	if len(plan.dependencies) == 0 && !plan.returnsError {
		plan = getFunctionPlan(b.factory.Type())
	}
	if len(b.dependencies) > 0 {
		plan.dependencies = cloneDependencies(b.dependencies)
		for i := range plan.dependencies {
			plan.dependencies[i] = normalizeDependency(plan.dependencies[i])
		}
	}
	return plan
}

// getInstance creates or returns a cached instance based on lifetime.
func (b *binding) getInstance(c *Container, sc *scope, state *resolutionState) (reflect.Value, error) {
	switch b.lifetime {
	case LifetimeSingleton:
		return b.getSingleton(c, sc, state)
	case LifetimeScoped:
		return b.getScoped(c, sc, state)
	case LifetimeTransient:
		return b.createInstance(c, sc, state)
	default:
		return reflect.Value{}, newError(ErrorCodeInvalidLifetime,
			fmt.Sprintf("unknown lifetime: %s", b.lifetime), nil)
	}
}

// getSingleton returns or creates a singleton instance.
func (b *binding) getSingleton(c *Container, sc *scope, state *resolutionState) (reflect.Value, error) {
	// Fast path: check if already created
	if instance := b.instance.Load(); instance != nil {
		return instance.(reflect.Value), nil
	}

	b.instanceMutex.Lock()
	defer b.instanceMutex.Unlock()

	// Double-check after acquiring lock
	if instance := b.instance.Load(); instance != nil {
		return instance.(reflect.Value), nil
	}

	instance, err := b.createInstance(c, sc, state)
	if err != nil {
		return reflect.Value{}, err
	}

	b.instance.Store(instance)
	return instance, nil
}

// getScoped returns or creates a scoped instance.
func (b *binding) getScoped(c *Container, sc *scope, state *resolutionState) (reflect.Value, error) {
	if sc == nil {
		return reflect.Value{}, newError(ErrorCodeScopeRequired,
			"scope required for scoped lifetime", nil)
	}
	if sc.IsClosed() {
		return reflect.Value{}, ErrScopeClosed
	}

	// Check if instance exists in scope
	if instance, ok := sc.Get(b.key()); ok {
		return instance, nil
	}

	instance, err := b.createInstance(c, sc, state)
	if err != nil {
		return reflect.Value{}, err
	}

	sc.Set(b.key(), instance)
	return instance, nil
}

// createInstance creates a new instance using the factory function.
func (b *binding) createInstance(c *Container, sc *scope, state *resolutionState) (instance reflect.Value, err error) {
	start := time.Now()
	defer func() {
		c.recordInstance(InstanceEvent{
			Key:       b.key(),
			Type:      b.typ,
			Name:      b.name,
			Lifetime:  b.lifetime,
			Duration:  time.Since(start),
			Err:       err,
			Container: c,
		})
	}()

	if !b.factory.IsValid() {
		return reflect.Value{}, newError(ErrorCodeNoFactory,
			fmt.Sprintf("no factory registered for type %v", b.typ), nil)
	}

	// Resolve dependencies
	args, err := c.resolveDependencies(plannedDependenciesForBinding(b), sc, state)
	if err != nil {
		return reflect.Value{}, err
	}

	// Call factory
	results := b.factory.Call(args)

	plan := factoryPlanForBinding(b)

	// Check for error
	if plan.returnsError && len(results) > 1 && !results[1].IsNil() {
		return reflect.Value{}, results[1].Interface().(error)
	}

	instance = results[0]

	// Call pre-construct hook.
	if b.hooks != nil && b.hooks.PreConstruct != nil {
		if err := b.hooks.PreConstruct(instance.Interface()); err != nil {
			return reflect.Value{}, err
		}
	}

	if pre, ok := instance.Interface().(PreConstruct); ok {
		pre.PreConstruct()
	}

	// Call post-construct hook
	if b.hooks != nil && b.hooks.PostConstruct != nil {
		if err := b.hooks.PostConstruct(instance.Interface()); err != nil {
			return reflect.Value{}, err
		}
	}

	// Call interface PostConstruct
	if post, ok := instance.Interface().(PostConstruct); ok {
		post.PostConstruct()
	}

	return instance, nil
}

// key returns a unique key for this binding
func (b *binding) key() string {
	return cacheKey(b.typ, b.name)
}

// close cleans up the binding if it has a close function.
func (b *binding) close() error {
	var errs []error

	if b.hooks != nil && b.hooks.CloseFunc != nil {
		if err := b.hooks.CloseFunc(); err != nil {
			errs = append(errs, err)
		}
	} else if b.lifetime == LifetimeSingleton {
		if instance := b.instance.Load(); instance != nil {
			if closeFn := extractCloseFunc(instance.(reflect.Value)); closeFn != nil {
				if err := closeFn(); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
