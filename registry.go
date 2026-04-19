package di

import "reflect"

type registryComponent struct {
	container *Container
}

func removeBindingKey(keys []string, key string) []string {
	filtered := keys[:0]
	for _, current := range keys {
		if current != key {
			filtered = append(filtered, current)
		}
	}
	return filtered
}

func appendBindingKey(keys []string, key string) []string {
	keys = removeBindingKey(keys, key)
	return append(keys, key)
}

func (c *Container) bindType(typ reflect.Type, opts ...Option) *binding {
	b, err := c.bindTypeE(typ, opts...)
	if err != nil {
		panic(err)
	}
	return b
}

func (c *Container) bindTypeE(typ reflect.Type, opts ...Option) (*binding, error) {
	cfg, err := newConfig(opts...)
	if err != nil {
		return nil, err
	}
	return c.bindTypeWithConfig(typ, cfg)
}

func (c *Container) bindTypeWithConfig(typ reflect.Type, cfg *config) (*binding, error) {
	return c.registry().bindTypeWithConfig(typ, cfg)
}

func (c *Container) unregisterBinding(key string) {
	c.registry().unregisterBinding(key)
}

func (r registryComponent) removeBindingIndexesLocked(b *binding, key string) {
	c := r.container
	if b == nil {
		return
	}

	if b.interfaceFor != nil {
		keys := removeBindingKey(c.registryState.Interfaces[b.interfaceFor], key)
		if len(keys) == 0 {
			delete(c.registryState.Interfaces, b.interfaceFor)
		} else {
			c.registryState.Interfaces[b.interfaceFor] = keys
		}
	}

	if b.group != "" {
		keys := removeBindingKey(c.registryState.Groups[b.group], key)
		if len(keys) == 0 {
			delete(c.registryState.Groups, b.group)
		} else {
			c.registryState.Groups[b.group] = keys
		}
	}
}

func (r registryComponent) unregisterBinding(key string) {
	c := r.container

	c.mu.Lock()
	defer c.mu.Unlock()

	b, ok := c.registryState.Bindings[key]
	if !ok {
		return
	}

	r.removeBindingIndexesLocked(b, key)
	delete(c.registryState.Bindings, key)
	c.registryState.BindingOrder = removeBindingKey(c.registryState.BindingOrder, key)
}

func (r registryComponent) bindTypeWithConfig(typ reflect.Type, cfg *config) (*binding, error) {
	c := r.container
	if c.lifecycle().isClosed() {
		return nil, ErrScopeClosed
	}

	b, err := newBindingWithConfig(typ, cfg)
	if err != nil {
		return nil, err
	}
	key := b.key()

	c.mu.Lock()
	defer c.mu.Unlock()

	if existing, ok := c.registryState.Bindings[key]; ok {
		r.removeBindingIndexesLocked(existing, key)
		c.registryState.BindingOrder = removeBindingKey(c.registryState.BindingOrder, key)
	}
	c.registryState.Bindings[key] = b
	c.registryState.BindingOrder = append(c.registryState.BindingOrder, key)

	if b.interfaceFor != nil {
		c.registryState.Interfaces[b.interfaceFor] = appendBindingKey(c.registryState.Interfaces[b.interfaceFor], key)
	}

	if b.group != "" {
		c.registryState.Groups[b.group] = appendBindingKey(c.registryState.Groups[b.group], key)
	}

	return b, nil
}

func (r registryComponent) findInterfaceBindingLocked(typ reflect.Type, name string) (*binding, bool, error) {
	c := r.container
	keys := c.registryState.Interfaces[typ]
	if len(keys) == 0 {
		return nil, false, nil
	}

	matches := make([]*binding, 0, len(keys))
	for _, key := range keys {
		b, ok := c.registryState.Bindings[key]
		if !ok || b == nil || b.name != name {
			continue
		}
		matches = append(matches, b)
	}

	switch len(matches) {
	case 0:
		return nil, false, nil
	case 1:
		return matches[0], true, nil
	default:
		return nil, true, ErrMultipleBindings
	}
}

func (r registryComponent) findLocalBindingLocked(typ reflect.Type, name string) (*binding, bool, error) {
	c := r.container
	key := cacheKey(typ, name)
	if b, ok := c.registryState.Bindings[key]; ok {
		return b, true, nil
	}

	return r.findInterfaceBindingLocked(typ, name)
}

func (r registryComponent) findBindingSourceOnly(typ reflect.Type, name string) (*binding, *Container, error) {
	c := r.container

	c.mu.RLock()
	if b, ok, err := r.findLocalBindingLocked(typ, name); ok {
		c.mu.RUnlock()
		return b, c, err
	}
	parent := c.hierarchyState.Parent
	c.mu.RUnlock()

	if parent != nil {
		return parent.registry().findBindingSourceOnly(typ, name)
	}

	return nil, nil, ErrBindingNotFound
}
