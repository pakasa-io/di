package scope

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestScopeLifecycleAndCloseOrder(t *testing.T) {
	var order []string
	extract := func(value reflect.Value) CloseFunc {
		switch value.Interface().(string) {
		case "one":
			return func() error {
				order = append(order, "one")
				return nil
			}
		case "two":
			return func() error {
				order = append(order, "two")
				return errors.New("two failed")
			}
		default:
			return nil
		}
	}

	parent := New(nil, extract)
	child := parent.CreateChild()
	grandchild := child.CreateChild()

	child.Set("one", reflect.ValueOf("one"))
	child.Set("two", reflect.ValueOf("two"))
	if value, ok := child.Get("one"); !ok || value.Interface().(string) != "one" {
		t.Fatalf("unexpected scope get result: %#v %v", value, ok)
	}

	parent.RemoveChild(grandchild)
	parent.AddChild(grandchild)

	if err := child.Close(); err == nil || !strings.Contains(err.Error(), "two failed") {
		t.Fatalf("expected close to aggregate child errors, got %v", err)
	}
	if !child.IsClosed() || !grandchild.IsClosed() {
		t.Fatal("expected child close to close descendants")
	}
	if _, ok := child.Get("one"); ok {
		t.Fatal("expected closed scope get to fail")
	}
	child.Set("three", reflect.ValueOf("three"))
	if len(order) != 2 || order[0] != "two" || order[1] != "one" {
		t.Fatalf("expected reverse close order, got %v", order)
	}
	if err := child.Close(); err != nil {
		t.Fatalf("expected second close to be a no-op, got %v", err)
	}
}
