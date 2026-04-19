package di

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	diModel "github.com/pakasa-io/di/internal/model"
	diPlan "github.com/pakasa-io/di/internal/plan"
)

type resolutionState struct {
	active map[string]bool
	stack  []string
	ctx    context.Context
}

var (
	structPlanCache sync.Map
)

// ClearRuntimeCaches clears reflection-derived plan caches used by DI resolution.
func ClearRuntimeCaches() {
	structPlanCache = sync.Map{}
	diPlan.Clear()
}

func newResolutionState(ctx ...context.Context) *resolutionState {
	resolutionContext := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		resolutionContext = ctx[0]
	}

	return &resolutionState{
		active: make(map[string]bool),
		stack:  make([]string, 0),
		ctx:    resolutionContext,
	}
}

func (s *resolutionState) enter(key string) error {
	if s == nil {
		return nil
	}

	if s.active[key] {
		trace := append(s.trace(), key)
		return newErrorWithTrace(
			ErrCircularDependency.Code,
			ErrCircularDependency.Message,
			nil,
			trace,
		)
	}

	s.active[key] = true
	s.stack = append(s.stack, key)
	return nil
}

func (s *resolutionState) leave(key string) {
	if s == nil {
		return
	}

	delete(s.active, key)
	if n := len(s.stack); n > 0 && s.stack[n-1] == key {
		s.stack = s.stack[:n-1]
		return
	}

	panic(fmt.Sprintf("di: resolutionState.leave invariant violated for %q", key))
}

func (s *resolutionState) trace() []string {
	if s == nil || len(s.stack) == 0 {
		return nil
	}

	return append([]string(nil), s.stack...)
}

func (s *resolutionState) wrap(err error) error {
	return attachTrace(err, s.trace())
}

func (s *resolutionState) context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// getType returns the reflect.Type for type T
func getType[T any]() reflect.Type {
	var zero T
	return reflect.TypeOf(&zero).Elem()
}

// isFuncValidFactory validates that a function can be used as a factory
func isFuncValidFactory(fnType reflect.Type, returnType reflect.Type) bool {
	if fnType.Kind() != reflect.Func {
		return false
	}

	if fnType.NumOut() == 0 || fnType.NumOut() > 2 {
		return false
	}

	// First return value must be assignable to returnType
	if !fnType.Out(0).AssignableTo(returnType) {
		return false
	}

	// Optional second return value must be error
	if fnType.NumOut() == 2 && !fnType.Out(1).AssignableTo(errorType) {
		return false
	}

	return true
}

func cloneDependencies(deps []Dependency) []Dependency {
	if len(deps) == 0 {
		return nil
	}
	return append([]Dependency(nil), deps...)
}

func getFunctionPlan(fnType reflect.Type) functionPlan {
	compiled := diPlan.FunctionFor(fnType)
	return functionPlan{
		dependencies: cloneDependencies(compiled.Dependencies),
		returnsError: compiled.ReturnsError,
	}
}

// getDependenciesFromFunc analyzes function parameters to extract dependencies
func getDependenciesFromFunc(fnType reflect.Type) []Dependency {
	return cloneDependencies(diPlan.DependenciesForFunc(fnType))
}

func getStructInjectionPlan(typ reflect.Type) []structFieldPlan {
	if cached, ok := structPlanCache.Load(typ); ok {
		return cached.([]structFieldPlan)
	}

	compiled := diPlan.StructFields(typ)
	plan := make([]structFieldPlan, 0, len(compiled))
	for _, field := range compiled {
		plan = append(plan, structFieldPlan{
			index:      field.Index,
			dependency: normalizeDependency(field.Dependency),
		})
	}
	actual, _ := structPlanCache.LoadOrStore(typ, plan)
	return actual.([]structFieldPlan)
}

func dependenciesForStructType(typ reflect.Type) []Dependency {
	return cloneDependencies(diPlan.DependenciesForStruct(typ))
}

// resolveDependencies resolves all dependencies for a factory function.
func (c *Container) resolveDependencies(deps []Dependency, sc *scope, state *resolutionState) ([]reflect.Value, error) {
	values := make([]reflect.Value, len(deps))

	for i, dep := range deps {
		dep = normalizeDependency(dep)
		val, err := c.resolveDependency(dep, sc, state)
		if err != nil {
			if dep.Name != "" {
				return nil, state.wrap(newError(
					ErrDependencyResolution.Code,
					fmt.Sprintf("failed to resolve named dependency %s", dep.Name),
					err,
				))
			}
			return nil, state.wrap(newError(
				ErrDependencyResolution.Code,
				fmt.Sprintf("failed to resolve dependency %s", dependencyLabel(dep)),
				err,
			))
		}
		values[i] = val
	}

	return values, nil
}

// cacheKey generates a unique key for caching instances
func cacheKey(t reflect.Type, name string) string {
	return diModel.CacheKey(t, name)
}

// Predefined types for comparison
var (
	errorType        = reflect.TypeOf((*error)(nil)).Elem()
	closerType       = reflect.TypeOf((*interface{ Close() error })(nil)).Elem()
	cancelFuncType   = reflect.TypeOf((*func())(nil)).Elem()
	contextType      = reflect.TypeOf((*context.Context)(nil)).Elem()
	containerPtrType = reflect.TypeOf((*Container)(nil))
)

// extractCloseFunc extracts a single close function from a value.
// Priority order is AutoCloseable, then io.Closer, then context.CancelFunc.
func extractCloseFunc(value reflect.Value) CloseFunc {
	if !value.IsValid() {
		return nil
	}

	// Check for AutoCloseable
	if closer, ok := value.Interface().(AutoCloseable); ok {
		return closer.Close
	}

	// Check for io.Closer
	if value.Type().Implements(closerType) {
		if closer, ok := value.Interface().(interface{ Close() error }); ok {
			return closer.Close
		}
	}

	// Check for context.CancelFunc
	if value.Type() == cancelFuncType {
		if fn, ok := value.Interface().(func()); ok {
			return func() error {
				fn()
				return nil
			}
		}
	}

	return nil
}

type functionPlan struct {
	dependencies []Dependency
	returnsError bool
}

type structFieldPlan struct {
	index      int
	dependency Dependency
}

func parseStructTags(tag string) map[string]string {
	return diPlan.ParseStructTags(tag)
}
