package builtin

import (
	"context"
	"reflect"
	"testing"

	diModel "github.com/pakasa-io/di/internal/model"
)

type testContainer struct{}

func TestForDependencyAndResolve(t *testing.T) {
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	containerType := reflect.TypeOf((*testContainer)(nil))

	if got := ForType(contextType, contextType, containerType); got != Context {
		t.Fatalf("expected context kind, got %v", got)
	}
	if got := ForType(containerType, contextType, containerType); got != Container {
		t.Fatalf("expected container kind, got %v", got)
	}
	if got := ForType(reflect.TypeOf(""), contextType, containerType); got != None {
		t.Fatalf("expected none kind, got %v", got)
	}

	if got := ForDependency(diModel.Dependency{Type: reflect.TypeOf(diModel.Optional[context.Context]{})}, contextType, containerType); got != Context {
		t.Fatalf("expected optional context dependency to normalize to context kind, got %v", got)
	}

	ctxValue := Resolve(Context, nil, nil)
	if !ctxValue.IsValid() || ctxValue.Interface().(context.Context) == nil {
		t.Fatalf("expected Resolve(Context) to produce a context, got %#v", ctxValue)
	}

	container := &testContainer{}
	containerValue := Resolve(Container, context.Background(), container)
	if !containerValue.IsValid() || containerValue.Interface().(*testContainer) != container {
		t.Fatalf("expected Resolve(Container) to return the provided container, got %#v", containerValue)
	}

	if value := Resolve(None, context.Background(), nil); value.IsValid() {
		t.Fatalf("expected Resolve(None) to return an invalid value, got %#v", value)
	}
}
