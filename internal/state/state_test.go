package state

import (
	"reflect"
	"testing"
)

func TestCloseStateAndConstructors(t *testing.T) {
	var nilState *CloseState
	if nilState.IsClosed() {
		t.Fatal("expected nil close state to be open")
	}
	if nilState.CloseOnce() {
		t.Fatal("expected nil close state close to return false")
	}

	var state CloseState
	if state.IsClosed() {
		t.Fatal("expected fresh close state to be open")
	}
	if !state.CloseOnce() || !state.IsClosed() {
		t.Fatal("expected close state to close once")
	}
	if state.CloseOnce() {
		t.Fatal("expected second close attempt to return false")
	}

	registry := NewRegistryState[int]()
	if registry.Bindings == nil || registry.Interfaces == nil || registry.Groups == nil || len(registry.BindingOrder) != 0 {
		t.Fatalf("unexpected registry state: %#v", registry)
	}

	parentType := reflect.TypeOf("parent")
	hierarchy := NewHierarchyState(parentType)
	if hierarchy.Parent != parentType || hierarchy.Children == nil {
		t.Fatalf("unexpected hierarchy state: %#v", hierarchy)
	}

	runtime := NewRuntimeState[func(), int]()
	if runtime.Overrides == nil || runtime.Closed.IsClosed() {
		t.Fatal("unexpected runtime state")
	}
}
