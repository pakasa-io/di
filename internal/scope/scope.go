package scope

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	internalState "github.com/pakasa-io/di/internal/state"
)

type CloseFunc func() error

type Scope struct {
	parent      *Scope
	children    map[*Scope]bool
	instances   map[string]reflect.Value
	closeFuncs  []CloseFunc
	closeOrder  []string
	extractFunc func(reflect.Value) CloseFunc
	mu          sync.RWMutex
	closed      internalState.CloseState
}

func New(parent *Scope, extract func(reflect.Value) CloseFunc) *Scope {
	s := &Scope{
		parent:      parent,
		children:    make(map[*Scope]bool),
		instances:   make(map[string]reflect.Value),
		closeFuncs:  make([]CloseFunc, 0),
		closeOrder:  make([]string, 0),
		extractFunc: extract,
	}
	if parent != nil {
		parent.AddChild(s)
	}
	return s
}

func (s *Scope) Get(key string) (reflect.Value, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed.IsClosed() {
		return reflect.Value{}, false
	}
	value, ok := s.instances[key]
	return value, ok
}

func (s *Scope) Set(key string, instance reflect.Value) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.IsClosed() {
		return
	}

	s.instances[key] = instance
	s.closeOrder = append(s.closeOrder, key)
	if s.extractFunc != nil {
		if closeFn := s.extractFunc(instance); closeFn != nil {
			s.closeFuncs = append(s.closeFuncs, closeFn)
		}
	}
}

func (s *Scope) AddChild(child *Scope) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.IsClosed() {
		return
	}
	if s.children == nil {
		s.children = make(map[*Scope]bool)
	}
	s.children[child] = true
}

func (s *Scope) RemoveChild(child *Scope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.children, child)
}

func (s *Scope) CreateChild() *Scope {
	return New(s, s.extractFunc)
}

func (s *Scope) Close() error {
	s.mu.Lock()
	if !s.closed.CloseOnce() {
		s.mu.Unlock()
		return nil
	}

	children := make([]*Scope, 0, len(s.children))
	for child := range s.children {
		children = append(children, child)
	}
	closeFuncs := append([]CloseFunc(nil), s.closeFuncs...)

	s.instances = nil
	s.closeFuncs = nil
	s.children = nil
	s.closeOrder = nil
	s.mu.Unlock()

	var errs []error
	for _, child := range children {
		if err := child.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for i := len(closeFuncs) - 1; i >= 0; i-- {
		if err := closeFuncs[i](); err != nil {
			errs = append(errs, err)
		}
	}
	if s.parent != nil {
		s.parent.RemoveChild(s)
	}
	if len(errs) > 0 {
		return fmt.Errorf("scope close failed: %w", errors.Join(errs...))
	}
	return nil
}

func (s *Scope) IsClosed() bool {
	return s.closed.IsClosed()
}
