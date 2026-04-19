package global

import (
	"errors"
	"sync"

	di "github.com/pakasa-io/di"
)

var (
	mu    sync.RWMutex
	named = make(map[string]*di.Container)
	def   *di.Container
)

// Default returns the process-wide default container.
func Default() *di.Container {
	mu.RLock()
	c := def
	mu.RUnlock()
	if c != nil && !c.IsClosed() {
		return c
	}

	mu.Lock()
	defer mu.Unlock()
	if def == nil || def.IsClosed() {
		def = di.NewContainer()
	}
	return def
}

// Named returns the named process-wide container, creating it on first use.
func Named(name string) *di.Container {
	if name == "" {
		return Default()
	}

	mu.RLock()
	c := named[name]
	mu.RUnlock()
	if c != nil && !c.IsClosed() {
		return c
	}

	mu.Lock()
	defer mu.Unlock()
	if c = named[name]; c != nil && !c.IsClosed() {
		return c
	}
	c = di.NewContainer()
	named[name] = c
	return c
}

// Container returns a named container, falling back to Default when name is empty or omitted.
func Container(name ...string) *di.Container {
	if len(name) == 0 {
		return Default()
	}
	return Named(name[0])
}

// Reset closes and clears all process-wide containers managed by this package.
func Reset() error {
	mu.Lock()
	containers := make([]*di.Container, 0, len(named)+1)
	for _, container := range named {
		if container != nil {
			containers = append(containers, container)
		}
	}
	if def != nil {
		containers = append(containers, def)
	}
	named = make(map[string]*di.Container)
	def = nil
	mu.Unlock()

	var errs []error
	seen := make(map[*di.Container]bool, len(containers))
	for _, container := range containers {
		if container == nil || seen[container] {
			continue
		}
		seen[container] = true
		if err := container.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
