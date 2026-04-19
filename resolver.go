package di

import (
	"context"
	"errors"
	"reflect"
	"time"

	diBuiltin "github.com/pakasa-io/di/internal/builtin"
)

type resolverComponent struct {
	container *Container
}

type aggregateBindingRef struct {
	key     string
	binding *binding
	owner   *Container
}

type aggregateQueryMode int

const (
	aggregateQueryGroup aggregateQueryMode = iota
	aggregateQueryInterface
)

type aggregateQuery struct {
	mode          aggregateQueryMode
	selector      string
	requestedType reflect.Type
	name          string
}

type aggregateSelection struct {
	refs            []aggregateBindingRef
	interfaceLookup bool
}

type resolutionSource struct {
	binding       *binding
	owner         *Container
	override      func(*Container, *scope) (reflect.Value, error)
	overrideOwner *Container
}

type dependencyAnalysis struct {
	dependency Dependency
	key        string
	builtin    diBuiltin.Kind
	source     resolutionSource
	sourceErr  error
	autoWired  bool
}

func analyzeDependency(lookup *Container, dep Dependency) dependencyAnalysis {
	return analyzeDependencyWithOverrides(lookup, dep, true)
}

func analyzeRegisteredDependency(lookup *Container, dep Dependency) dependencyAnalysis {
	return analyzeDependencyWithOverrides(lookup, dep, false)
}

func analyzeDependencyWithOverrides(lookup *Container, dep Dependency, includeOverrides bool) dependencyAnalysis {
	dep = normalizeDependency(dep)
	analysis := dependencyAnalysis{
		dependency: dep,
		builtin:    diBuiltin.ForDependency(dep, contextType, containerPtrType),
	}
	if dep.Type == nil {
		return analysis
	}
	analysis.key = dependencyKey(dep)
	if analysis.builtin != diBuiltin.None {
		return analysis
	}

	var (
		source resolutionSource
		err    error
	)
	if includeOverrides {
		source, err = lookup.resolver().lookupResolutionSource(dep.Type, dep.Name)
	} else {
		var owner *Container
		source.binding, owner, err = lookup.registry().findBindingSourceOnly(dep.Type, dep.Name)
		source.owner = owner
	}
	if err == nil {
		analysis.source = source
		return analysis
	}

	analysis.sourceErr = err
	analysis.autoWired = shouldAutoWireStruct(lookup, dep)
	return analysis
}

func (a dependencyAnalysis) hasSource() bool {
	return a.source.override != nil || a.source.binding != nil
}

func (c *Container) resolveByType(typ reflect.Type, sc *scope, state *resolutionState) (reflect.Value, error) {
	return c.resolver().resolveByType(typ, sc, state)
}

func (c *Container) resolveDependency(dep Dependency, sc *scope, state *resolutionState) (reflect.Value, error) {
	return c.resolver().resolveDependency(dep, sc, state)
}

// Invoke calls a function with dependency injection using context.Background().
func (c *Container) Invoke(fn any) error {
	return c.InvokeContext(context.Background(), fn)
}

// InvokeContext calls a function with dependency injection and injects the provided context.
func (c *Container) InvokeContext(ctx context.Context, fn any) error {
	return c.resolver().invoke(ctx, fn)
}

func filteredAggregateRefs(refs []aggregateBindingRef, name string) []aggregateBindingRef {
	if name == "" || len(refs) == 0 {
		return refs
	}

	filtered := make([]aggregateBindingRef, 0, len(refs))
	for _, ref := range refs {
		if ref.binding == nil || ref.binding.name != name {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func instanceScope(owner *Container, requested *scope, lifetime Lifetime) *scope {
	if lifetime == LifetimeSingleton {
		return owner.scope
	}
	return requested
}

func (r resolverComponent) findOverrideSource(key string) (func(*Container, *scope) (reflect.Value, error), *Container, bool) {
	c := r.container
	c.mu.RLock()
	override, ok := c.runtimeState.Overrides[key]
	parent := c.hierarchyState.Parent
	c.mu.RUnlock()

	if ok {
		return override, c, true
	}
	if parent != nil {
		return parent.resolver().findOverrideSource(key)
	}
	return nil, nil, false
}

func (r resolverComponent) lookupResolutionSource(typ reflect.Type, name string) (resolutionSource, error) {
	c := r.container
	c.mu.RLock()
	key := cacheKey(typ, name)
	if override, ok := c.runtimeState.Overrides[key]; ok {
		c.mu.RUnlock()
		return resolutionSource{override: override, overrideOwner: c}, nil
	}
	if b, ok, err := c.registry().findLocalBindingLocked(typ, name); ok {
		c.mu.RUnlock()
		if err != nil {
			return resolutionSource{}, err
		}
		return resolutionSource{binding: b, owner: c}, nil
	}
	parent := c.hierarchyState.Parent
	c.mu.RUnlock()

	if parent != nil {
		return parent.resolver().lookupResolutionSource(typ, name)
	}

	return resolutionSource{}, ErrBindingNotFound
}

func (r resolverComponent) collectGroupBindings(group string) []aggregateBindingRef {
	seen := make(map[string]bool)
	refs := make([]aggregateBindingRef, 0)

	for current := r.container; current != nil; current = current.hierarchyState.Parent {
		current.mu.RLock()
		for _, key := range current.registryState.Groups[group] {
			if seen[key] {
				continue
			}
			b, ok := current.registryState.Bindings[key]
			if !ok || b == nil {
				continue
			}
			refs = append(refs, aggregateBindingRef{key: key, binding: b, owner: current})
			seen[key] = true
		}
		current.mu.RUnlock()
	}

	return refs
}

func (r resolverComponent) collectInterfaceBindings(iface reflect.Type) []aggregateBindingRef {
	seen := make(map[string]bool)
	refs := make([]aggregateBindingRef, 0)

	for current := r.container; current != nil; current = current.hierarchyState.Parent {
		current.mu.RLock()
		for _, key := range current.registryState.Interfaces[iface] {
			if seen[key] {
				continue
			}
			b, ok := current.registryState.Bindings[key]
			if !ok || b == nil {
				continue
			}
			refs = append(refs, aggregateBindingRef{key: key, binding: b, owner: current})
			seen[key] = true
		}
		current.mu.RUnlock()
	}

	return refs
}

func (r resolverComponent) selectAggregateBindings(query aggregateQuery) aggregateSelection {
	switch query.mode {
	case aggregateQueryGroup:
		return aggregateSelection{
			refs:            filteredAggregateRefs(r.collectGroupBindings(query.selector), query.name),
			interfaceLookup: false,
		}
	case aggregateQueryInterface:
		return aggregateSelection{
			refs:            filteredAggregateRefs(r.collectInterfaceBindings(query.requestedType), query.name),
			interfaceLookup: true,
		}
	}

	return aggregateSelection{}
}

func (r resolverComponent) resolveAggregateBinding(ref aggregateBindingRef, sc *scope, state *resolutionState, overrideKeys ...string) (value reflect.Value, err error) {
	c := r.container
	start := time.Now()
	overrideUsed := false
	resolutionKey := ref.key
	defer func() {
		c.recordResolve(ResolveEvent{
			Key:       resolutionKey,
			Type:      ref.binding.typ,
			Name:      ref.binding.name,
			Duration:  time.Since(start),
			Err:       err,
			Override:  overrideUsed,
			Container: c,
			Owner:     ref.owner,
		})
	}()

	if err = state.enter(ref.key); err != nil {
		return reflect.Value{}, err
	}
	defer state.leave(ref.key)

	keys := append([]string(nil), overrideKeys...)
	keys = append(keys, ref.key)
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if override, overrideOwner, ok := c.resolver().findOverrideSource(key); ok {
			overrideUsed = true
			resolutionKey = key
			value, err = override(overrideOwner, sc)
			if err != nil {
				return reflect.Value{}, state.wrap(err)
			}
			return value, nil
		}
	}

	value, err = ref.binding.getInstance(ref.owner, instanceScope(ref.owner, sc, ref.binding.lifetime), state)
	if err != nil {
		return reflect.Value{}, state.wrap(err)
	}
	return value, nil
}

func (r resolverComponent) resolveDependency(dep Dependency, sc *scope, state *resolutionState) (reflect.Value, error) {
	return r.resolveNormalizedDependency(normalizeDependency(dep), sc, state)
}

func (r resolverComponent) resolveByType(typ reflect.Type, sc *scope, state *resolutionState) (reflect.Value, error) {
	return r.resolveNormalizedDependency(normalizeDependency(Dependency{Type: typ}), sc, state)
}

type resolvedDependency struct {
	value        reflect.Value
	owner        *Container
	overrideUsed bool
}

func (r resolverComponent) resolveNormalizedDependency(dep Dependency, sc *scope, state *resolutionState) (value reflect.Value, err error) {
	c := r.container
	analysis := analyzeDependency(c, dep)
	key := analysis.key
	start := time.Now()
	var (
		overrideUsed bool
		owner        *Container
	)
	defer func() {
		c.recordResolve(ResolveEvent{
			Key:       key,
			Type:      dep.Type,
			Name:      dep.Name,
			Duration:  time.Since(start),
			Err:       err,
			Override:  overrideUsed,
			Container: c,
			Owner:     owner,
		})
	}()

	if err = c.lifecycle().ensureOpen(sc); err != nil {
		return reflect.Value{}, err
	}

	resolved, resolveErr := r.resolveWithAnalysis(dep, analysis, sc, state)
	if resolveErr != nil {
		err = resolveErr
		return reflect.Value{}, err
	}
	overrideUsed = resolved.overrideUsed
	owner = resolved.owner
	value = resolved.value
	return value, nil
}

func (r resolverComponent) resolveWithAnalysis(dep Dependency, analysis dependencyAnalysis, sc *scope, state *resolutionState) (resolved resolvedDependency, err error) {
	if analysis.builtin != diBuiltin.None {
		resolved.owner = r.container
		resolved.value = diBuiltin.Resolve(analysis.builtin, state.context(), r.container)
		if dep.Optional {
			resolved.value, err = optionalValueFor(dep, resolved.value, true)
		}
		return resolved, err
	}

	if err = state.enter(analysis.key); err != nil {
		return resolvedDependency{}, err
	}
	defer state.leave(analysis.key)

	if analysis.sourceErr != nil {
		switch {
		case analysis.autoWired:
			resolved.owner = r.container
			resolved.value, err = r.autoWireStruct(dep.Type, sc, state)
			if err != nil {
				return resolvedDependency{}, state.wrap(err)
			}
			if dep.Optional {
				resolved.value, err = optionalValueFor(dep, resolved.value, true)
			}
			return resolved, err
		case dep.Optional && errors.Is(analysis.sourceErr, ErrBindingNotFound):
			resolved.value, err = optionalValueFor(dep, reflect.Zero(dep.Type), false)
			return resolved, err
		default:
			return resolvedDependency{}, state.wrap(analysis.sourceErr)
		}
	}

	if analysis.source.override != nil {
		if analysis.source.overrideOwner != nil && analysis.source.overrideOwner.isClosed() {
			return resolvedDependency{}, ErrScopeClosed
		}
		resolved.overrideUsed = true
		resolved.owner = analysis.source.overrideOwner
		resolved.value, err = analysis.source.override(analysis.source.overrideOwner, sc)
		if err != nil {
			return resolvedDependency{}, state.wrap(err)
		}
		if dep.Optional {
			resolved.value, err = optionalValueFor(dep, resolved.value, true)
		}
		return resolved, err
	}

	if analysis.source.binding == nil {
		if dep.Optional {
			resolved.value, err = optionalValueFor(dep, reflect.Zero(dep.Type), false)
			return resolved, err
		}
		return resolvedDependency{}, state.wrap(ErrBindingNotFound)
	}

	resolved.owner = analysis.source.owner
	if resolved.owner != nil && resolved.owner.isClosed() {
		return resolvedDependency{}, ErrScopeClosed
	}
	resolved.value, err = analysis.source.binding.getInstance(
		analysis.source.owner,
		instanceScope(analysis.source.owner, sc, analysis.source.binding.lifetime),
		state,
	)
	if err != nil {
		return resolvedDependency{}, state.wrap(err)
	}
	if dep.Optional {
		resolved.value, err = optionalValueFor(dep, resolved.value, true)
	}
	return resolved, err
}

func (r resolverComponent) invoke(ctx context.Context, fn any) error {
	return r.invokeInScope(ctx, r.container.scope, fn)
}

func (r resolverComponent) invokeInScope(ctx context.Context, sc *scope, fn any) error {
	c := r.container
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		return newError(ErrorCodeInvalidFunction, "Invoke requires a function", nil)
	}
	if err := c.lifecycle().ensureOpen(sc); err != nil {
		return err
	}

	state := newResolutionState(ctx)
	args, err := c.resolveDependencies(getDependenciesFromFunc(fnType), sc, state)
	if err != nil {
		return err
	}

	results := reflect.ValueOf(fn).Call(args)
	if getFunctionPlan(fnType).returnsError {
		last := results[len(results)-1]
		if !last.IsNil() {
			return last.Interface().(error)
		}
	}

	return nil
}

func (r resolverComponent) autoWireStruct(typ reflect.Type, sc *scope, state *resolutionState) (reflect.Value, error) {
	instance := reflect.New(typ).Elem()

	for _, fieldPlan := range getStructInjectionPlan(typ) {
		fieldValue, err := r.resolveDependency(fieldPlan.dependency, sc, state)
		if err != nil {
			return reflect.Value{}, state.wrap(err)
		}

		instance.Field(fieldPlan.index).Set(fieldValue)
	}

	return instance, nil
}
