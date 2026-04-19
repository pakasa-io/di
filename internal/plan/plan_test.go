package plan

import (
	"context"
	"reflect"
	"testing"

	diModel "github.com/pakasa-io/di/internal/model"
)

type structInput struct {
	Service *int
	Named   string                `di:"name=primary"`
	Opt     diModel.Optional[int] `di:"name=optional"`
	Skip    bool                  `di:"-"`
	hidden  int
}

func TestFunctionAndStructPlans(t *testing.T) {
	Clear()
	_ = structInput{}.hidden

	fnType := reflect.TypeOf(func(context.Context, *int) (string, error) { return "", nil })
	fnPlan := FunctionFor(fnType)
	if !fnPlan.ReturnsError || len(fnPlan.Dependencies) != 2 {
		t.Fatalf("unexpected function plan: %#v", fnPlan)
	}
	deps := DependenciesForFunc(fnType)
	if len(deps) != 2 || deps[0].Type.Kind() != reflect.Interface || deps[1].Type.Kind() != reflect.Ptr {
		t.Fatalf("unexpected function dependencies: %#v", deps)
	}

	structType := reflect.TypeOf(structInput{})
	fields := StructFields(structType)
	if len(fields) != 3 {
		t.Fatalf("expected 3 injectable fields, got %#v", fields)
	}
	if fields[1].Dependency.Name != "primary" {
		t.Fatalf("expected named field dependency, got %#v", fields[1])
	}
	if !fields[2].Dependency.Optional || fields[2].Dependency.Name != "optional" || fields[2].Dependency.Type.Kind() != reflect.Int {
		t.Fatalf("expected optional field normalization, got %#v", fields[2])
	}

	structDeps := DependenciesForStruct(structType)
	if len(structDeps) != 3 {
		t.Fatalf("unexpected struct dependencies: %#v", structDeps)
	}
	if deps := DependenciesForStruct(reflect.TypeOf(0)); deps != nil {
		t.Fatalf("expected non-struct dependency list to be nil, got %#v", deps)
	}
}

func TestParseStructTags(t *testing.T) {
	if got := ParseStructTags(""); len(got) != 0 {
		t.Fatalf("expected empty tags, got %#v", got)
	}
	got := ParseStructTags("name=primary,group=cache,flag")
	if got["name"] != "primary" || got["group"] != "cache" || got["flag"] != "" {
		t.Fatalf("unexpected parsed tags: %#v", got)
	}
}
