package di

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Graph describes the effective dependency graph visible from a container.
type Graph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

// GraphNode describes a binding or missing dependency in the graph.
type GraphNode struct {
	ID           string
	Key          string
	Type         reflect.Type
	Name         string
	Group        string
	InterfaceFor reflect.Type
	Lifetime     Lifetime
	Owner        *Container
	Inherited    bool
	HasFactory   bool
	Optional     bool
	Missing      bool
	Override     bool
	AutoWired    bool
}

// GraphEdge describes a dependency edge between two graph nodes.
type GraphEdge struct {
	From       string
	To         string
	Dependency Dependency
	Missing    bool
}

type graphItem struct {
	binding *binding
	owner   *Container
	dep     Dependency
}

func graphNodeID(owner *Container, key string) string {
	return fmt.Sprintf("%p:%s", owner, key)
}

func overrideGraphNodeID(owner *Container, key string) string {
	return fmt.Sprintf("%p:override:%s", owner, key)
}

func missingGraphNodeID(dep Dependency) string {
	dep = normalizeDependency(dep)
	return "missing:" + cacheKey(dep.Type, dep.Name)
}

func missingGraphNode(dep Dependency) GraphNode {
	dep = normalizeDependency(dep)
	return GraphNode{
		ID:       missingGraphNodeID(dep),
		Key:      cacheKey(dep.Type, dep.Name),
		Type:     dep.Type,
		Name:     dep.Name,
		Optional: dep.Optional,
		Missing:  true,
	}
}

func graphNodeFrom(owner, current *Container, b *binding) GraphNode {
	return GraphNode{
		ID:           graphNodeID(owner, b.key()),
		Key:          b.key(),
		Type:         b.typ,
		Name:         b.name,
		Group:        b.group,
		InterfaceFor: b.interfaceFor,
		Lifetime:     b.lifetime,
		Owner:        owner,
		Inherited:    owner != nil && current != nil && owner != current,
		HasFactory:   b.factory.IsValid(),
	}
}

func overrideGraphNode(owner *Container, key string) GraphNode {
	return GraphNode{
		ID:         overrideGraphNodeID(owner, key),
		Key:        key,
		Owner:      owner,
		Inherited:  owner != nil,
		HasFactory: true,
		Override:   true,
	}
}

func autoWiredGraphNodeID(owner *Container, dep Dependency) string {
	return fmt.Sprintf("%p:auto:%s", owner, cacheKey(dep.Type, dep.Name))
}

func autoWiredGraphNode(owner, current *Container, dep Dependency) GraphNode {
	return GraphNode{
		ID:         autoWiredGraphNodeID(owner, dep),
		Key:        cacheKey(dep.Type, dep.Name),
		Type:       dep.Type,
		Name:       dep.Name,
		Owner:      owner,
		Inherited:  owner != nil && current != nil && owner != current,
		HasFactory: false,
		AutoWired:  true,
	}
}

// Graph returns the effective dependency graph visible from this container.
func (c *Container) Graph() (Graph, error) {
	return c.graphInScope(c.scope)
}

func (c *Container) graphInScope(sc *scope) (Graph, error) {
	graph, err := c.introspection().graphInScope(sc)
	if err != nil {
		return Graph{}, err
	}
	sort.Slice(graph.Nodes, func(i, j int) bool {
		if graph.Nodes[i].Missing != graph.Nodes[j].Missing {
			return !graph.Nodes[i].Missing
		}
		if graph.Nodes[i].Override != graph.Nodes[j].Override {
			return !graph.Nodes[i].Override
		}
		if graph.Nodes[i].Key != graph.Nodes[j].Key {
			return graph.Nodes[i].Key < graph.Nodes[j].Key
		}
		return graph.Nodes[i].ID < graph.Nodes[j].ID
	})
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From != graph.Edges[j].From {
			return graph.Edges[i].From < graph.Edges[j].From
		}
		return graph.Edges[i].To < graph.Edges[j].To
	})
	return graph, nil
}

// DumpGraph returns a readable text dump of this container's effective graph.
func (c *Container) DumpGraph() (string, error) {
	graph, err := c.Graph()
	if err != nil {
		return "", err
	}
	return graph.String(), nil
}

// DumpGraphDOT returns a Graphviz DOT dump of this container's effective graph.
func (c *Container) DumpGraphDOT() (string, error) {
	graph, err := c.Graph()
	if err != nil {
		return "", err
	}
	return graph.DOT(), nil
}

func graphLabel(node GraphNode) string {
	if node.Missing {
		if node.Optional {
			return "optional-missing:" + node.Key
		}
		return "missing:" + node.Key
	}

	parts := []string{node.Key}
	if node.AutoWired {
		parts = append(parts, "autowired")
	}
	if node.Override {
		parts = append(parts, "override")
	}
	if node.Lifetime != LifetimeUnknown {
		parts = append(parts, node.Lifetime.String())
	}
	if node.Inherited {
		parts = append(parts, "inherited")
	}
	return strings.Join(parts, " ")
}

// String returns a readable text dump of the graph.
func (g Graph) String() string {
	if len(g.Nodes) == 0 {
		return "(empty graph)"
	}

	adjacency := make(map[string][]GraphEdge)
	nodeByID := make(map[string]GraphNode, len(g.Nodes))
	for _, node := range g.Nodes {
		nodeByID[node.ID] = node
	}
	for _, edge := range g.Edges {
		adjacency[edge.From] = append(adjacency[edge.From], edge)
	}
	for id := range adjacency {
		sort.Slice(adjacency[id], func(i, j int) bool {
			return adjacency[id][i].To < adjacency[id][j].To
		})
	}

	var builder strings.Builder
	for i, node := range g.Nodes {
		if i > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(graphLabel(node))
		for _, edge := range adjacency[node.ID] {
			builder.WriteString("\n  -> ")
			if edge.Dependency.Optional {
				builder.WriteString("(optional) ")
			}
			builder.WriteString(graphLabel(nodeByID[edge.To]))
		}
	}

	return builder.String()
}

func dotEscape(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")
	return replacer.Replace(value)
}

// DOT returns a Graphviz DOT representation of the graph.
func (g Graph) DOT() string {
	var builder strings.Builder
	builder.WriteString("digraph di {\n")

	for _, node := range g.Nodes {
		label := graphLabel(node)
		shape := "box"
		if node.Missing {
			shape = "diamond"
		}
		builder.WriteString(fmt.Sprintf("  %q [label=\"%s\" shape=\"%s\"];\n", node.ID, dotEscape(label), shape))
	}

	for _, edge := range g.Edges {
		if edge.Dependency.Optional {
			builder.WriteString(fmt.Sprintf("  %q -> %q [style=\"dashed\" label=\"optional\"];\n", edge.From, edge.To))
			continue
		}
		builder.WriteString(fmt.Sprintf("  %q -> %q;\n", edge.From, edge.To))
	}

	builder.WriteString("}\n")
	return builder.String()
}
