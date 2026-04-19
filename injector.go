package di

import (
	"context"
	"fmt"
	"reflect"
)

// DepInjector handles automatic dependency injection into functions and structs
type DepInjector struct {
	container *Container
	scope     *scope
}

func newInjectorWithScope(container *Container, scope *scope) *DepInjector {
	return &DepInjector{
		container: container,
		scope:     scope,
	}
}

// NewInjector creates a new injector for the given container's default scope.
func NewInjector(container *Container) *DepInjector {
	if container == nil {
		return nil
	}
	return newInjectorWithScope(container, container.scope)
}

// NewScopeInjector creates a new injector bound to the provided scope.
func NewScopeInjector(sc *Scope) *DepInjector {
	if sc == nil || sc.container == nil || sc.state == nil {
		return nil
	}
	return newInjectorWithScope(sc.container, sc.state)
}

// Call injects dependencies into a function and calls it
func (inj *DepInjector) Call(fn any) ([]reflect.Value, error) {
	return inj.CallContext(context.Background(), fn)
}

// CallContext injects dependencies into a function, calls it, and injects the provided context.
func (inj *DepInjector) CallContext(ctx context.Context, fn any) ([]reflect.Value, error) {
	fnValue := reflect.ValueOf(fn)
	if !fnValue.IsValid() {
		return nil, newError(ErrorCodeNotAFunction, "provided value is not a function", nil)
	}

	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, newError(ErrorCodeNotAFunction, "provided value is not a function", nil)
	}

	args, err := inj.injectFunctionArgs(fnType, newResolutionState(ctx))
	if err != nil {
		return nil, err
	}

	return fnValue.Call(args), nil
}

// injectFunctionArgs resolves dependencies for function parameters.
func (inj *DepInjector) injectFunctionArgs(fnType reflect.Type, state *resolutionState) ([]reflect.Value, error) {
	args, err := inj.container.resolveDependencies(getDependenciesFromFunc(fnType), inj.scope, state)
	if err != nil {
		return nil, newError(ErrorCodeDependencyInjectionFailed, "failed to inject function arguments", err)
	}
	return args, nil
}

// FillStruct injects dependencies into a struct's fields.
func (inj *DepInjector) FillStruct(structPtr any) error {
	return inj.FillStructContext(context.Background(), structPtr)
}

// FillStructContext injects dependencies into a struct's fields and injects the provided context.
func (inj *DepInjector) FillStructContext(ctx context.Context, structPtr any) error {
	value := reflect.ValueOf(structPtr)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return newError(ErrorCodeInvalidStruct, "FillStruct requires a non-nil pointer to a struct", nil)
	}

	target := value.Elem()
	if target.Kind() != reflect.Struct {
		return newError(ErrorCodeInvalidStruct, "FillStruct requires a pointer to a struct", nil)
	}

	targetType := target.Type()
	for _, fieldPlan := range getStructInjectionPlan(targetType) {
		field := target.Field(fieldPlan.index)
		if !field.CanSet() {
			continue
		}

		var (
			instance reflect.Value
			err      error
		)
		state := newResolutionState(ctx)
		instance, err = inj.container.resolveDependency(fieldPlan.dependency, inj.scope, state)
		if err != nil {
			return newError(ErrorCodeDependencyInjectionFailed,
				fmt.Sprintf("failed to inject field %s of type %v", targetType.Field(fieldPlan.index).Name, fieldPlan.dependency.Type), err)
		}

		field.Set(instance)
	}

	return nil
}
