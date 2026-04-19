package di

import (
	"fmt"
	"reflect"

	diModel "github.com/pakasa-io/di/internal/model"
)

// Dependency describes a typed dependency request with an optional name and
// optionality.
type Dependency = diModel.Dependency

// Optional is a resolution-time wrapper for dependencies that may be absent.
type Optional[T any] = diModel.Optional[T]

func optionalInnerType(typ reflect.Type) (reflect.Type, bool) {
	return diModel.OptionalInnerType(typ)
}

func normalizeDependency(dep Dependency) Dependency {
	return diModel.NormalizeDependency(dep)
}

func dependencyKey(dep Dependency) string {
	return diModel.DependencyKey(dep)
}

func dependencyLabel(dep Dependency) string {
	return diModel.DependencyLabel(dep)
}

func optionalValueFor(dep Dependency, value reflect.Value, ok bool) (reflect.Value, error) {
	dep = normalizeDependency(dep)
	if !dep.Optional {
		return value, nil
	}

	declaredType := dep.DeclaredType
	if declaredType == nil {
		return reflect.Value{}, newError(
			ErrUnsupportedAPIShape.Code,
			fmt.Sprintf("optional dependency %s must be declared using Optional[T]", dependencyKey(dep)),
			nil,
		)
	}
	if _, wrapped := optionalInnerType(declaredType); !wrapped {
		return reflect.Value{}, newError(
			ErrUnsupportedAPIShape.Code,
			fmt.Sprintf("optional dependency %s must be declared using Optional[T]", dependencyKey(dep)),
			nil,
		)
	}

	result := reflect.New(declaredType).Elem()
	valueField := result.FieldByName("Value")
	okField := result.FieldByName("OK")
	if !ok {
		valueField.Set(reflect.Zero(valueField.Type()))
		okField.SetBool(false)
		return result, nil
	}

	if !value.IsValid() {
		value = reflect.Zero(dep.Type)
	}
	valueField.Set(value)
	okField.SetBool(true)
	return result, nil
}

func validateRegisteredType(typ reflect.Type) error {
	if inner, ok := optionalInnerType(typ); ok {
		return newError(
			ErrUnsupportedAPIShape.Code,
			fmt.Sprintf("cannot register Optional[%s]; register %s directly", inner, inner),
			nil,
		)
	}
	return nil
}
