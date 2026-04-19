package plan

import (
	"reflect"
	"strings"
	"sync"

	diModel "github.com/pakasa-io/di/internal/model"
)

type Function struct {
	Dependencies []diModel.Dependency
	ReturnsError bool
}

type Field struct {
	Index      int
	Dependency diModel.Dependency
}

var (
	functionCache sync.Map
	structCache   sync.Map
	errorType     = reflect.TypeOf((*error)(nil)).Elem()
)

func Clear() {
	functionCache = sync.Map{}
	structCache = sync.Map{}
}

func cloneDependencies(deps []diModel.Dependency) []diModel.Dependency {
	if len(deps) == 0 {
		return nil
	}
	return append([]diModel.Dependency(nil), deps...)
}

// FunctionFor returns the compiled dependency plan for a function type.
func FunctionFor(fnType reflect.Type) Function {
	if cached, ok := functionCache.Load(fnType); ok {
		return cached.(Function)
	}

	compiled := Function{
		Dependencies: make([]diModel.Dependency, fnType.NumIn()),
		ReturnsError: fnType.NumOut() > 0 && fnType.Out(fnType.NumOut()-1).AssignableTo(errorType),
	}
	for i := 0; i < fnType.NumIn(); i++ {
		compiled.Dependencies[i] = diModel.NormalizeDependency(diModel.Dependency{Type: fnType.In(i)})
	}

	actual, _ := functionCache.LoadOrStore(fnType, compiled)
	return actual.(Function)
}

// DependenciesForFunc returns a cloned dependency slice for the function.
func DependenciesForFunc(fnType reflect.Type) []diModel.Dependency {
	return cloneDependencies(FunctionFor(fnType).Dependencies)
}

// StructFields returns the compiled struct field injection plan.
func StructFields(typ reflect.Type) []Field {
	if cached, ok := structCache.Load(typ); ok {
		return cached.([]Field)
	}

	fields := make([]Field, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		tag := field.Tag.Get("di")
		if tag == "-" {
			continue
		}

		settings := ParseStructTags(tag)
		fields = append(fields, Field{
			Index:      i,
			Dependency: diModel.NormalizeDependency(diModel.Dependency{Type: field.Type, Name: settings["name"]}),
		})
	}

	actual, _ := structCache.LoadOrStore(typ, fields)
	return actual.([]Field)
}

// DependenciesForStruct returns a cloned dependency list for struct auto-wiring.
func DependenciesForStruct(typ reflect.Type) []diModel.Dependency {
	if typ == nil || typ.Kind() != reflect.Struct {
		return nil
	}

	plan := StructFields(typ)
	deps := make([]diModel.Dependency, 0, len(plan))
	for _, field := range plan {
		deps = append(deps, diModel.NormalizeDependency(field.Dependency))
	}
	return deps
}

// ParseStructTags parses `di` struct tags into key/value settings.
func ParseStructTags(tag string) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}

	for _, pair := range strings.Split(tag, ",") {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = kv[1]
		} else if len(kv) == 1 {
			result[kv[0]] = ""
		}
	}

	return result
}
