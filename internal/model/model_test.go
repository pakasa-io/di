package model

import (
	"reflect"
	"testing"
)

func TestLifetimeStringAndDependencyHelpers(t *testing.T) {
	if LifetimeUnknown.String() != "unknown" || LifetimeSingleton.String() != "singleton" || LifetimeTransient.String() != "transient" || LifetimeScoped.String() != "scoped" {
		t.Fatal("unexpected lifetime string mapping")
	}

	optionalType := reflect.TypeOf(Optional[int]{})
	inner, ok := OptionalInnerType(optionalType)
	if !ok || inner.Kind() != reflect.Int {
		t.Fatalf("expected OptionalInnerType to unwrap int, got %v %v", inner, ok)
	}
	if _, ok := OptionalInnerType(reflect.TypeOf(struct{ Value int }{})); ok {
		t.Fatal("expected non-optional struct to be rejected")
	}

	dep := NormalizeDependency(Dependency{Type: optionalType, Name: "named"})
	if dep.Type.Kind() != reflect.Int || !dep.Optional || dep.DeclaredType != optionalType || dep.Name != "named" {
		t.Fatalf("unexpected normalized dependency: %#v", dep)
	}

	if got := CacheKey(nil, "name"); got != "<nil>@name" {
		t.Fatalf("unexpected nil cache key: %q", got)
	}
	if got := CacheKey(reflect.TypeOf(""), ""); got != "string" {
		t.Fatalf("unexpected cache key: %q", got)
	}
	if got := DependencyKey(Dependency{Type: optionalType}); got != "int" {
		t.Fatalf("unexpected dependency key: %q", got)
	}
	if got := DependencyLabel(Dependency{Type: optionalType}); got != "int" {
		t.Fatalf("unexpected dependency label for optional wrapper type: %q", got)
	}
	if got := DependencyLabel(Dependency{Type: reflect.TypeOf(0), Optional: true}); got != "optional:int" {
		t.Fatalf("unexpected dependency label: %q", got)
	}
}
