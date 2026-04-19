package di

import "errors"

type lifecycleComponent struct {
	container *Container
}

func (c *Container) ensureOpen(sc *scope) error {
	return c.lifecycle().ensureOpen(sc)
}

func (c *Container) isClosed() bool {
	return c.lifecycle().isClosed()
}

// IsClosed reports whether this container has been closed.
func (c *Container) IsClosed() bool {
	if c == nil {
		return true
	}
	return c.isClosed()
}

// Close closes the container and all descendant scopes.
func (c *Container) Close() error {
	return c.lifecycle().close()
}

func (l lifecycleComponent) isClosed() bool {
	return l.container.runtimeState.Closed.IsClosed()
}

func (l lifecycleComponent) ensureOpen(sc *scope) error {
	if l.isClosed() {
		return ErrScopeClosed
	}
	if sc != nil && sc.IsClosed() {
		return ErrScopeClosed
	}
	return nil
}

func (l lifecycleComponent) addChild(child *Container) {
	c := l.container
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.hierarchyState.Children == nil {
		c.hierarchyState.Children = make(map[*Container]bool)
	}
	c.hierarchyState.Children[child] = true
}

func (l lifecycleComponent) removeChild(child *Container) {
	c := l.container
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.hierarchyState.Children, child)
}

func (l lifecycleComponent) close() error {
	c := l.container
	if !c.runtimeState.Closed.CloseOnce() {
		return nil
	}

	var errs []error

	var children []*Container
	c.mu.RLock()
	for child := range c.hierarchyState.Children {
		children = append(children, child)
	}
	c.mu.RUnlock()

	for _, child := range children {
		if err := child.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if err := c.scope.Close(); err != nil {
		errs = append(errs, err)
	}

	c.mu.RLock()
	keys := append([]string(nil), c.registryState.BindingOrder...)
	c.mu.RUnlock()

	for i := len(keys) - 1; i >= 0; i-- {
		c.mu.RLock()
		b, ok := c.registryState.Bindings[keys[i]]
		c.mu.RUnlock()

		if ok && b != nil {
			if err := b.close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if c.hierarchyState.Parent != nil {
		c.hierarchyState.Parent.lifecycle().removeChild(c)
	}

	if len(errs) > 0 {
		return newError(
			ErrorCodeContainerClose,
			"errors occurred while closing container",
			errors.Join(errs...),
		)
	}

	return nil
}
