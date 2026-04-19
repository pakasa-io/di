package state

import (
	"reflect"
	"sync/atomic"
)

type CloseState struct {
	closed atomic.Bool
}

func (s *CloseState) IsClosed() bool {
	if s == nil {
		return false
	}
	return s.closed.Load()
}

func (s *CloseState) CloseOnce() bool {
	if s == nil {
		return false
	}
	return s.closed.CompareAndSwap(false, true)
}

type RegistryState[B any] struct {
	Bindings     map[string]B
	BindingOrder []string
	Interfaces   map[reflect.Type][]string
	Groups       map[string][]string
}

func NewRegistryState[B any]() RegistryState[B] {
	return RegistryState[B]{
		Bindings:     make(map[string]B),
		BindingOrder: make([]string, 0),
		Interfaces:   make(map[reflect.Type][]string),
		Groups:       make(map[string][]string),
	}
}

type HierarchyState[C comparable] struct {
	Parent   C
	Children map[C]bool
}

func NewHierarchyState[C comparable](parent C) HierarchyState[C] {
	return HierarchyState[C]{
		Parent:   parent,
		Children: make(map[C]bool),
	}
}

type RuntimeState[O any, M any] struct {
	StructAutoWire  atomic.Bool
	Overrides       map[string]O
	Instrumentation atomic.Value
	Metrics         M
	Closed          CloseState
}

func NewRuntimeState[O any, M any]() RuntimeState[O, M] {
	return RuntimeState[O, M]{
		Overrides: make(map[string]O),
	}
}
