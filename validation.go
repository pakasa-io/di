package di

import (
	"fmt"

	diBuiltin "github.com/pakasa-io/di/internal/builtin"
)

type validationWalkState struct {
	trace  []string
	active map[string]bool
}

func newValidationWalkState(rootNode, rootLabel string) *validationWalkState {
	return &validationWalkState{
		trace:  []string{rootLabel},
		active: map[string]bool{rootNode: true},
	}
}

func (s *validationWalkState) push(nodeID, label string) {
	s.trace = append(s.trace, label)
	s.active[nodeID] = true
}

func (s *validationWalkState) pop(nodeID string) {
	delete(s.active, nodeID)
	if n := len(s.trace); n > 0 {
		s.trace = s.trace[:n-1]
	}
}

func (s *validationWalkState) traceWith(next string) []string {
	trace := append([]string(nil), s.trace...)
	if next != "" {
		trace = append(trace, next)
	}
	return trace
}

type validationWalker struct {
	rootLifetime Lifetime
	analyzer     func(*Container, Dependency) dependencyAnalysis
	state        *validationWalkState
}

func newValidationWalker(owner *Container, root *binding, rootLifetime Lifetime, analyzer func(*Container, Dependency) dependencyAnalysis) *validationWalker {
	return &validationWalker{
		rootLifetime: rootLifetime,
		analyzer:     analyzer,
		state:        newValidationWalkState(bindingNodeID(owner, root), bindingTraceLabel(root)),
	}
}

func (w *validationWalker) validate(owner *Container, deps []Dependency) []error {
	var issues []error
	for _, dep := range deps {
		issues = append(issues, w.validateDependency(owner, dep)...)
	}
	return issues
}

func (w *validationWalker) validateDependency(owner *Container, dep Dependency) []error {
	analysis := w.analyzer(owner, dep)
	dep = analysis.dependency

	switch analysis.builtin {
	case diBuiltin.Container:
		return nil
	case diBuiltin.Context:
		return w.validateContextDependency(dep)
	}

	if dep.Type == nil {
		return []error{newErrorWithTrace(
			ErrInvalidOption.Code,
			"dependency type cannot be nil",
			nil,
			w.state.traceWith("nil"),
		)}
	}

	label := dependencyLabel(dep)
	if analysis.sourceErr != nil {
		return w.validateMissingDependency(owner, dep, analysis, label)
	}
	if !analysis.hasSource() || analysis.source.binding == nil {
		return nil
	}
	return w.validateResolvedBinding(dep, analysis.source.owner, analysis.source.binding, label)
}

func (w *validationWalker) validateContextDependency(dep Dependency) []error {
	if w.rootLifetime != LifetimeSingleton {
		return nil
	}
	return []error{newErrorWithTrace(
		ErrorCodeInvalidLifetimeGraph,
		"singleton depends on context.Context",
		nil,
		w.state.traceWith(dependencyLabel(dep)),
	)}
}

func (w *validationWalker) validateMissingDependency(owner *Container, dep Dependency, analysis dependencyAnalysis, label string) []error {
	if analysis.autoWired {
		depNodeID := autoWireValidationNodeID(owner, dep)
		if w.state.active[depNodeID] {
			return []error{newErrorWithTrace(
				ErrCircularDependency.Code,
				ErrCircularDependency.Message,
				nil,
				w.state.traceWith(label),
			)}
		}
		w.state.push(depNodeID, label)
		defer w.state.pop(depNodeID)
		return w.validate(owner, dependenciesForStructType(dep.Type))
	}
	if dep.Optional {
		return nil
	}
	return []error{attachTrace(
		newError(
			ErrDependencyResolution.Code,
			fmt.Sprintf("missing dependency %s", label),
			analysis.sourceErr,
		),
		w.state.traceWith(label),
	)}
}

func (w *validationWalker) validateResolvedBinding(dep Dependency, owner *Container, b *binding, label string) []error {
	var issues []error
	if w.rootLifetime == LifetimeSingleton && b.lifetime == LifetimeScoped {
		issues = append(issues, newErrorWithTrace(
			ErrorCodeInvalidLifetimeGraph,
			fmt.Sprintf("singleton depends on scoped dependency %s", label),
			nil,
			w.state.traceWith(label),
		))
	}

	depNodeID := bindingNodeID(owner, b)
	if w.state.active[depNodeID] {
		issues = append(issues, newErrorWithTrace(
			ErrCircularDependency.Code,
			ErrCircularDependency.Message,
			nil,
			w.state.traceWith(label),
		))
		return issues
	}

	deps, err := bindingDependencies(b)
	if err != nil {
		issues = append(issues, attachTrace(err, w.state.traceWith(label)))
		return issues
	}

	w.state.push(depNodeID, label)
	defer w.state.pop(depNodeID)
	issues = append(issues, w.validate(owner, deps)...)
	return issues
}

func bindingNodeID(owner *Container, b *binding) string {
	return fmt.Sprintf("%p:%s", owner, cacheKey(b.typ, b.name))
}

func bindingTraceLabel(b *binding) string {
	return cacheKey(b.typ, b.name)
}

func autoWireValidationNodeID(owner *Container, dep Dependency) string {
	return fmt.Sprintf("%p:auto:%s", owner, cacheKey(dep.Type, dep.Name))
}

func bindingDependencies(b *binding) ([]Dependency, error) {
	if b == nil {
		return nil, ErrBindingNotFound
	}
	if !b.factory.IsValid() {
		return nil, newError(ErrorCodeNoFactory, fmt.Sprintf("no factory registered for type %v", b.typ), nil)
	}
	if err := validateFactoryForBinding(b, b.factory.Interface()); err != nil {
		return nil, err
	}
	return plannedDependenciesForBinding(b), nil
}

// Validate validates the bindings reachable from the container without
// constructing instances.
func (c *Container) Validate() error {
	if c == nil {
		return nil
	}
	if err := c.ensureOpen(c.scope); err != nil {
		return err
	}
	issues := c.collectValidationIssues(make(map[*Container]bool))
	if len(issues) == 0 {
		return nil
	}

	return &ValidationError{Issues: issues}
}

func (c *Container) validateVisibleInScope(sc *scope) error {
	if c == nil {
		return nil
	}
	if err := c.ensureOpen(sc); err != nil {
		return err
	}
	return validationResult(c.collectVisibleValidationIssues())
}

func validationResult(issues []error) error {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}

func (c *Container) collectValidationIssues(visited map[*Container]bool) []error {
	if c == nil || visited[c] {
		return nil
	}
	visited[c] = true

	var issues []error

	c.mu.RLock()
	keys := append([]string(nil), c.registryState.BindingOrder...)
	children := make([]*Container, 0, len(c.hierarchyState.Children))
	for child := range c.hierarchyState.Children {
		children = append(children, child)
	}
	bindings := make([]*binding, 0, len(keys))
	for _, key := range keys {
		if b, ok := c.registryState.Bindings[key]; ok && b != nil {
			bindings = append(bindings, b)
		}
	}
	c.mu.RUnlock()

	issues = append(issues, c.validateAliasCollisions(bindings)...)
	for _, b := range bindings {
		issues = append(issues, c.validateBinding(b)...)
	}
	for _, child := range children {
		issues = append(issues, child.collectValidationIssues(visited)...)
	}

	return issues
}

func (c *Container) collectVisibleValidationIssues() []error {
	if c == nil {
		return nil
	}

	items := c.introspection().visibleGraphItems()
	overrideKeys := c.collectVisibleOverrideKeys()
	visitedBindings := make(map[string]bool)
	issues := c.collectVisibleAliasValidationIssues()

	for _, item := range items {
		if item.binding == nil || item.owner == nil {
			continue
		}

		key := item.binding.key()
		if overrideKeys[key] {
			continue
		}

		nodeID := bindingNodeID(item.owner, item.binding)
		if visitedBindings[nodeID] {
			continue
		}
		visitedBindings[nodeID] = true
		issues = append(issues, c.validateVisibleBinding(item.owner, item.binding)...)
	}

	return issues
}

func (c *Container) collectVisibleOverrideKeys() map[string]bool {
	keys := make(map[string]bool)
	for current := c; current != nil; current = current.hierarchyState.Parent {
		current.mu.RLock()
		for key := range current.runtimeState.Overrides {
			if !keys[key] {
				keys[key] = true
			}
		}
		current.mu.RUnlock()
	}
	return keys
}

func (c *Container) collectVisibleAliasValidationIssues() []error {
	type aliasKey struct {
		iface string
		name  string
	}

	resolved := make(map[aliasKey]bool)
	var issues []error

	for current := c; current != nil; current = current.hierarchyState.Parent {
		type localAliasCount struct {
			ifaceType string
			bindings  []string
		}

		current.mu.RLock()
		local := make(map[aliasKey]localAliasCount)
		for _, key := range current.registryState.BindingOrder {
			b, ok := current.registryState.Bindings[key]
			if !ok || b == nil || b.interfaceFor == nil {
				continue
			}

			alias := aliasKey{iface: b.interfaceFor.String(), name: b.name}
			if resolved[alias] {
				continue
			}

			overrideKey := cacheKey(b.interfaceFor, b.name)
			if _, overridden := current.runtimeState.Overrides[overrideKey]; overridden {
				resolved[alias] = true
				continue
			}

			entry := local[alias]
			entry.ifaceType = b.interfaceFor.String()
			entry.bindings = append(entry.bindings, bindingTraceLabel(b))
			local[alias] = entry
		}
		current.mu.RUnlock()

		for alias, entry := range local {
			resolved[alias] = true
			if len(entry.bindings) < 2 {
				continue
			}
			issues = append(issues, newErrorWithTrace(
				ErrMultipleBindings.Code,
				fmt.Sprintf("multiple bindings registered for interface %s", entry.ifaceType),
				nil,
				entry.bindings,
			))
		}
	}

	return issues
}

func (c *Container) validateAliasCollisions(bindings []*binding) []error {
	type aliasKey struct {
		iface string
		name  string
	}

	counts := make(map[aliasKey][]string)
	for _, b := range bindings {
		if b == nil || b.interfaceFor == nil {
			continue
		}
		key := aliasKey{iface: b.interfaceFor.String(), name: b.name}
		counts[key] = append(counts[key], bindingTraceLabel(b))
	}

	var issues []error
	for key, bindings := range counts {
		if len(bindings) < 2 {
			continue
		}

		issues = append(issues, newErrorWithTrace(
			ErrMultipleBindings.Code,
			fmt.Sprintf("multiple bindings registered for interface %s", key.iface),
			nil,
			bindings,
		))
	}

	return issues
}

func (c *Container) validateBinding(b *binding) []error {
	return c.validateBindingWithAnalyzer(c, b, analyzeRegisteredDependency)
}

func (c *Container) validateVisibleBinding(owner *Container, b *binding) []error {
	return c.validateBindingWithAnalyzer(owner, b, analyzeDependency)
}

func (c *Container) validateBindingWithAnalyzer(owner *Container, b *binding, analyzer func(*Container, Dependency) dependencyAnalysis) []error {
	var issues []error

	if err := validateBindingConfiguration(b); err != nil {
		issues = append(issues, attachTrace(err, []string{bindingTraceLabel(b)}))
		return issues
	}

	deps, err := bindingDependencies(b)
	if err != nil {
		issues = append(issues, attachTrace(err, []string{bindingTraceLabel(b)}))
		return issues
	}

	walker := newValidationWalker(owner, b, b.lifetime, analyzer)
	issues = append(issues, walker.validate(owner, deps)...)
	return issues
}
