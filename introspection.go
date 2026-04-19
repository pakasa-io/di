package di

import (
	"errors"
	"reflect"

	diBuiltin "github.com/pakasa-io/di/internal/builtin"
)

// BindingInfo describes a registered binding.
type BindingInfo struct {
	Key          string
	Type         reflect.Type
	Name         string
	Group        string
	InterfaceFor reflect.Type
	Lifetime     Lifetime
	HasFactory   bool
	Dependencies []Dependency
	Tags         map[string]string
	Owner        *Container
	Inherited    bool
}

// ContainerInfo describes a container and its locally registered bindings.
type ContainerInfo struct {
	Container    *Container
	Parent       *Container
	Closed       bool
	BindingCount int
	ChildCount   int
	Bindings     []BindingInfo
}

// HasInContainer reports whether type T has an explicit binding or override in an open container.
func HasInContainer[T any](c *Container, opts ...Option) bool {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return false
	}
	return c.HasBinding(getType[T](), cfg.name)
}

// DescribeInContainer returns binding metadata for type T from the provided container.
func DescribeInContainer[T any](c *Container, opts ...Option) (*BindingInfo, error) {
	cfg, err := containerConfig(c, opts...)
	if err != nil {
		return nil, err
	}
	return c.DescribeBinding(getType[T](), cfg.name)
}

// HasBinding reports whether an open container has an explicit binding or override for the given type and optional name.
func (c *Container) HasBinding(typ reflect.Type, name string) bool {
	return c.introspection().hasBinding(typ, name)
}

// DescribeBinding returns metadata for the binding selected by type and optional name.
func (c *Container) DescribeBinding(typ reflect.Type, name string) (*BindingInfo, error) {
	return c.introspection().describeBinding(typ, name)
}

// ListBindings returns locally registered bindings in registration order.
func (c *Container) ListBindings() []BindingInfo {
	return c.introspection().listBindings()
}

// DescribeContainer returns a summary of the container and its local bindings.
func (c *Container) DescribeContainer() ContainerInfo {
	return c.introspection().describeContainer()
}

type introspectionComponent struct {
	container *Container
}

func validateBindingLookupType(typ reflect.Type) error {
	if typ == nil {
		return newError(ErrInvalidOption.Code, "dependency type cannot be nil", nil)
	}
	return nil
}

func cloneTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(tags))
	for key, value := range tags {
		cloned[key] = value
	}
	return cloned
}

func bindingDependenciesForInfo(b *binding) []Dependency {
	if b == nil {
		return nil
	}
	return plannedDependenciesForBinding(b)
}

func bindingInfoFrom(owner, current *Container, b *binding) BindingInfo {
	return BindingInfo{
		Key:          b.key(),
		Type:         b.typ,
		Name:         b.name,
		Group:        b.group,
		InterfaceFor: b.interfaceFor,
		Lifetime:     b.lifetime,
		HasFactory:   b.factory.IsValid(),
		Dependencies: bindingDependenciesForInfo(b),
		Tags:         cloneTags(b.tags),
		Owner:        owner,
		Inherited:    owner != nil && current != nil && owner != current,
	}
}

func (i introspectionComponent) hasBinding(typ reflect.Type, name string) bool {
	c := i.container
	if err := c.ensureOpen(c.scope); err != nil {
		return false
	}
	if err := validateBindingLookupType(typ); err != nil {
		return false
	}

	analysis := analyzeDependency(c, Dependency{Type: typ, Name: name})
	return analysis.source.override != nil || analysis.source.binding != nil
}

func (i introspectionComponent) describeBinding(typ reflect.Type, name string) (*BindingInfo, error) {
	c := i.container
	if err := validateBindingLookupType(typ); err != nil {
		return nil, err
	}
	dep := normalizeDependency(Dependency{Type: typ, Name: name})
	b, owner, err := c.registry().findBindingSourceOnly(dep.Type, dep.Name)
	if err != nil {
		return nil, err
	}

	info := bindingInfoFrom(owner, c, b)
	return &info, nil
}

func (i introspectionComponent) listBindings() []BindingInfo {
	c := i.container
	c.mu.RLock()
	keys := append([]string(nil), c.registryState.BindingOrder...)
	bindings := make([]BindingInfo, 0, len(keys))
	for _, key := range keys {
		b, ok := c.registryState.Bindings[key]
		if !ok || b == nil {
			continue
		}
		bindings = append(bindings, bindingInfoFrom(c, c, b))
	}
	c.mu.RUnlock()

	return bindings
}

func (i introspectionComponent) describeContainer() ContainerInfo {
	c := i.container
	c.mu.RLock()
	childCount := len(c.hierarchyState.Children)
	parent := c.hierarchyState.Parent
	c.mu.RUnlock()

	bindings := i.listBindings()
	return ContainerInfo{
		Container:    c,
		Parent:       parent,
		Closed:       c.isClosed(),
		BindingCount: len(bindings),
		ChildCount:   childCount,
		Bindings:     bindings,
	}
}

func (i introspectionComponent) explain(typ reflect.Type, name string) (ResolutionExplanation, error) {
	return i.explainInScope(i.container.scope, typ, name)
}

func (i introspectionComponent) explainInScope(sc *scope, typ reflect.Type, name string) (ResolutionExplanation, error) {
	c := i.container
	if err := c.ensureOpen(sc); err != nil {
		return ResolutionExplanation{}, err
	}
	if err := validateBindingLookupType(typ); err != nil {
		return ResolutionExplanation{}, err
	}
	return i.explainDependencyFrom(c, normalizeDependency(Dependency{Type: typ, Name: name}), make(map[string]bool))
}

func (i introspectionComponent) explainDependencyFrom(lookup *Container, dep Dependency, seen map[string]bool) (ResolutionExplanation, error) {
	dep = normalizeDependency(dep)
	explanation := ResolutionExplanation{
		Requested: dep,
		Container: lookup,
		Optional:  dep.Optional,
	}

	if dep.Type == nil {
		explanation.Missing = true
		return explanation, newError(ErrInvalidOption.Code, "dependency type cannot be nil", nil)
	}
	explanation.Key = dependencyKey(dep)

	analysis := analyzeDependency(lookup, dep)
	if analysis.builtin != diBuiltin.None {
		explanation.Owner = lookup
		return explanation, nil
	}

	if analysis.sourceErr != nil {
		if analysis.autoWired {
			explanation.Owner = lookup
			explanation.AutoWired = true

			autoNodeID := autoWiredGraphNodeID(lookup, dep)
			if seen[autoNodeID] {
				return explanation, nil
			}
			seen[autoNodeID] = true
			defer delete(seen, autoNodeID)

			for _, childDep := range dependenciesForStructType(dep.Type) {
				child, childErr := i.explainDependencyFrom(lookup, childDep, seen)
				if childErr != nil && !child.Missing {
					child.Missing = true
				}
				explanation.Dependencies = append(explanation.Dependencies, child)
			}
			return explanation, nil
		}
		if dep.Optional && errors.Is(analysis.sourceErr, ErrBindingNotFound) {
			explanation.Missing = true
			return explanation, nil
		}
		explanation.Missing = true
		return explanation, analysis.sourceErr
	}

	if analysis.source.override != nil {
		explanation.Owner = analysis.source.overrideOwner
		explanation.Override = true
		explanation.HasFactory = true
		return explanation, nil
	}

	binding := analysis.source.binding
	owner := analysis.source.owner
	explanation.Owner = owner
	explanation.Key = binding.key()
	explanation.Lifetime = binding.lifetime
	explanation.HasFactory = binding.factory.IsValid()

	nodeID := graphNodeID(owner, explanation.Key)
	if seen[nodeID] {
		return explanation, nil
	}
	seen[nodeID] = true
	defer delete(seen, nodeID)

	for _, childDep := range plannedDependenciesForBinding(binding) {
		child, childErr := i.explainDependencyFrom(owner, childDep, seen)
		if childErr != nil && !child.Missing {
			child.Missing = true
		}
		explanation.Dependencies = append(explanation.Dependencies, child)
	}

	return explanation, nil
}

func (i introspectionComponent) visibleGraphItems() []graphItem {
	c := i.container
	seen := make(map[string]bool)
	items := make([]graphItem, 0)

	for current := c; current != nil; current = current.hierarchyState.Parent {
		current.mu.RLock()
		keys := append([]string(nil), current.registryState.BindingOrder...)
		current.mu.RUnlock()
		for _, key := range keys {
			if seen[key] {
				continue
			}
			current.mu.RLock()
			b, ok := current.registryState.Bindings[key]
			current.mu.RUnlock()
			if !ok || b == nil {
				continue
			}
			items = append(items, graphItem{binding: b, owner: current})
			seen[key] = true
		}
	}

	return items
}

func (i introspectionComponent) visibleOverrides() map[string]*Container {
	c := i.container
	owners := make(map[string]*Container)
	for current := c; current != nil; current = current.hierarchyState.Parent {
		current.mu.RLock()
		for key := range current.runtimeState.Overrides {
			if _, exists := owners[key]; exists {
				continue
			}
			owners[key] = current
		}
		current.mu.RUnlock()
	}
	return owners
}

type graphWalkState struct {
	current        *Container
	overrideOwners map[string]*Container
	nodeMap        map[string]GraphNode
	edgeSet        map[string]bool
	edges          []GraphEdge
	queue          []graphItem
	processed      map[string]bool
}

func newGraphWalkState(current *Container, items []graphItem, overrideOwners map[string]*Container) *graphWalkState {
	return &graphWalkState{
		current:        current,
		overrideOwners: overrideOwners,
		nodeMap:        make(map[string]GraphNode),
		edgeSet:        make(map[string]bool),
		edges:          make([]GraphEdge, 0),
		queue:          append([]graphItem(nil), items...),
		processed:      make(map[string]bool),
	}
}

func (g *graphWalkState) addNode(node GraphNode) {
	g.nodeMap[node.ID] = node
}

func (g *graphWalkState) enqueue(item graphItem) {
	g.queue = append(g.queue, item)
}

func (g *graphWalkState) addEdge(edge GraphEdge) {
	edgeKey := edge.From + "->" + edge.To
	if g.edgeSet[edgeKey] {
		return
	}
	g.edgeSet[edgeKey] = true
	g.edges = append(g.edges, edge)
}

func (g *graphWalkState) drainOverrideNodes() {
	for key, owner := range g.overrideOwners {
		overrideNode := overrideGraphNode(owner, key)
		overrideNode.Inherited = owner != g.current
		g.nodeMap[overrideNode.ID] = overrideNode
	}
}

func (g *graphWalkState) nodes() []GraphNode {
	nodes := make([]GraphNode, 0, len(g.nodeMap))
	for _, node := range g.nodeMap {
		nodes = append(nodes, node)
	}
	return nodes
}

func (i introspectionComponent) graphNodeForItem(item graphItem, state *graphWalkState) (GraphNode, []Dependency, bool) {
	if item.owner == nil {
		return GraphNode{}, nil, false
	}
	if item.binding != nil {
		node := graphNodeFrom(item.owner, state.current, item.binding)
		if overrideOwner, ok := state.overrideOwners[item.binding.key()]; ok {
			node = overrideGraphNode(overrideOwner, item.binding.key())
			node.Type = item.binding.typ
			node.Name = item.binding.name
			node.Group = item.binding.group
			node.InterfaceFor = item.binding.interfaceFor
			node.Inherited = overrideOwner != state.current
		}
		return node, bindingDependenciesForInfo(item.binding), true
	}
	if !shouldAutoWireStruct(item.owner, item.dep) {
		return GraphNode{}, nil, false
	}
	return autoWiredGraphNode(item.owner, state.current, item.dep), dependenciesForStructType(item.dep.Type), true
}

func (i introspectionComponent) graphEdgeForDependency(item graphItem, from GraphNode, dep Dependency, state *graphWalkState) (GraphEdge, bool) {
	analysis := analyzeDependency(item.owner, dep)
	if analysis.builtin != diBuiltin.None {
		return GraphEdge{}, false
	}

	edge := GraphEdge{
		From:       from.ID,
		Dependency: analysis.dependency,
	}

	switch {
	case analysis.source.override != nil:
		overrideNode := overrideGraphNode(analysis.source.overrideOwner, analysis.key)
		overrideNode.Inherited = analysis.source.overrideOwner != state.current
		state.addNode(overrideNode)
		edge.To = overrideNode.ID
	case analysis.autoWired:
		target := autoWiredGraphNode(item.owner, state.current, analysis.dependency)
		state.addNode(target)
		edge.To = target.ID
		state.enqueue(graphItem{owner: item.owner, dep: analysis.dependency})
	case analysis.sourceErr != nil || analysis.source.binding == nil:
		missing := missingGraphNode(analysis.dependency)
		state.addNode(missing)
		edge.To = missing.ID
		edge.Missing = true
	default:
		target := graphNodeFrom(analysis.source.owner, state.current, analysis.source.binding)
		if overrideOwner, ok := state.overrideOwners[analysis.source.binding.key()]; ok {
			target.Override = true
			target.Inherited = overrideOwner != state.current
		}
		state.addNode(target)
		edge.To = target.ID
		state.enqueue(graphItem{binding: analysis.source.binding, owner: analysis.source.owner})
	}

	return edge, true
}

func (i introspectionComponent) graphInScope(sc *scope) (Graph, error) {
	c := i.container
	if err := c.ensureOpen(sc); err != nil {
		return Graph{}, err
	}
	state := newGraphWalkState(c, i.visibleGraphItems(), i.visibleOverrides())

	for len(state.queue) > 0 {
		item := state.queue[0]
		state.queue = state.queue[1:]

		node, deps, ok := i.graphNodeForItem(item, state)
		if !ok {
			continue
		}
		state.addNode(node)
		if state.processed[node.ID] {
			continue
		}
		state.processed[node.ID] = true
		if node.Override {
			continue
		}

		for _, dep := range deps {
			edge, ok := i.graphEdgeForDependency(item, node, dep, state)
			if !ok {
				continue
			}
			state.addEdge(edge)
		}
	}
	state.drainOverrideNodes()
	return Graph{Nodes: state.nodes(), Edges: state.edges}, nil
}
