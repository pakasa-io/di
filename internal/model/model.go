package model

import (
	"fmt"
	"reflect"
	"strings"
)

// Lifetime controls how a dependency instance is cached.
type Lifetime int

const (
	LifetimeUnknown Lifetime = iota
	LifetimeSingleton
	LifetimeTransient
	LifetimeScoped
)

func (l Lifetime) String() string {
	switch l {
	case LifetimeSingleton:
		return "singleton"
	case LifetimeTransient:
		return "transient"
	case LifetimeScoped:
		return "scoped"
	default:
		return "unknown"
	}
}

// Dependency describes a dependency with optional name and optionality.
type Dependency struct {
	Type         reflect.Type
	Name         string
	Optional     bool
	DeclaredType reflect.Type
}

// Optional wraps a dependency that may or may not be present.
type Optional[T any] struct {
	Value T
	OK    bool
}

// CloseFunc is a function that can be registered for cleanup.
type CloseFunc func() error

// PreConstruct is called before the instance is returned from the container.
type PreConstruct interface {
	PreConstruct()
}

// PostConstruct is called after all dependencies are injected.
type PostConstruct interface {
	PostConstruct()
}

// AutoCloseable is called when the container or scope is closing.
type AutoCloseable interface {
	Close() error
}

// LifecycleHooks holds lifecycle hooks for a binding.
type LifecycleHooks struct {
	PreConstruct  func(instance any) error
	PostConstruct func(instance any) error
	CloseFunc     CloseFunc
}

var (
	optionalSentinelType = reflect.TypeOf(Optional[struct{}]{})
	boolType             = reflect.TypeOf(false)
)

// OptionalInnerType reports the wrapped type for Optional[T].
func OptionalInnerType(typ reflect.Type) (reflect.Type, bool) {
	if typ == nil || typ.Kind() != reflect.Struct || typ.PkgPath() != optionalSentinelType.PkgPath() {
		return nil, false
	}
	if !strings.HasPrefix(typ.Name(), "Optional[") || typ.NumField() != 2 {
		return nil, false
	}
	if typ.Field(0).Name != "Value" || typ.Field(1).Name != "OK" || typ.Field(1).Type != boolType {
		return nil, false
	}
	return typ.Field(0).Type, true
}

// NormalizeDependency normalizes Optional[T] declarations to T + Optional=true.
func NormalizeDependency(dep Dependency) Dependency {
	if dep.Type == nil {
		return dep
	}
	if dep.DeclaredType == nil {
		dep.DeclaredType = dep.Type
	}
	if inner, ok := OptionalInnerType(dep.Type); ok {
		dep.Optional = true
		dep.DeclaredType = dep.Type
		dep.Type = inner
	}
	return dep
}

// CacheKey formats a dependency key from type and name.
func CacheKey(t reflect.Type, name string) string {
	typeLabel := "<nil>"
	if t != nil {
		typeLabel = t.String()
	}
	if name != "" {
		return fmt.Sprintf("%s@%s", typeLabel, name)
	}
	return typeLabel
}

// DependencyKey formats a dependency key from a dependency declaration.
func DependencyKey(dep Dependency) string {
	dep = NormalizeDependency(dep)
	return CacheKey(dep.Type, dep.Name)
}

// DependencyLabel formats a human-readable dependency label.
func DependencyLabel(dep Dependency) string {
	key := DependencyKey(dep)
	if dep.Optional {
		return "optional:" + key
	}
	return key
}
