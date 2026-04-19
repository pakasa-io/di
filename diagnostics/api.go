package diagnostics

import di "github.com/pakasa-io/di"

// BindingInfo describes a registered binding.
type BindingInfo = di.BindingInfo

// ContainerInfo describes a container and its locally registered bindings.
type ContainerInfo = di.ContainerInfo

// ResolutionExplanation describes the binding or override selected for a resolution.
type ResolutionExplanation = di.ResolutionExplanation

// Graph describes the effective dependency graph visible from a container or scope.
type Graph = di.Graph

// GraphNode describes a binding, override, auto-wired node, or missing dependency in a graph.
type GraphNode = di.GraphNode

// GraphEdge describes a dependency edge between graph nodes.
type GraphEdge = di.GraphEdge

// ValidationError aggregates one or more graph validation issues.
type ValidationError = di.ValidationError

// Has reports whether type T has an explicit binding or override in an open container.
func Has[T any](c *di.Container, opts ...di.Option) bool {
	return di.HasInContainer[T](c, opts...)
}

// HasNamed reports whether the named binding for type T is resolvable from the provided container.
func HasNamed[T any](c *di.Container, name string, opts ...di.Option) bool {
	return di.HasNamedInContainer[T](c, name, opts...)
}

// Describe returns binding metadata for type T from the provided container.
func Describe[T any](c *di.Container, opts ...di.Option) (*BindingInfo, error) {
	return di.DescribeInContainer[T](c, opts...)
}

// DescribeNamed returns binding metadata for the named binding of type T from the provided container.
func DescribeNamed[T any](c *di.Container, name string, opts ...di.Option) (*BindingInfo, error) {
	return di.DescribeNamedInContainer[T](c, name, opts...)
}

// ListBindings returns locally registered bindings for the provided container.
func ListBindings(c *di.Container) []BindingInfo {
	if c == nil {
		return nil
	}
	return c.ListBindings()
}

// DescribeContainer returns a summary of the provided container.
func DescribeContainer(c *di.Container) ContainerInfo {
	if c == nil {
		return ContainerInfo{}
	}
	return c.DescribeContainer()
}

// Explain returns the selected binding or override for type T from the provided container.
func Explain[T any](c *di.Container, opts ...di.Option) (*ResolutionExplanation, error) {
	return di.ExplainFrom[T](c, opts...)
}

// ExplainNamed returns the selected binding or override for the named binding of type T from the provided container.
func ExplainNamed[T any](c *di.Container, name string, opts ...di.Option) (*ResolutionExplanation, error) {
	return di.ExplainNamedFrom[T](c, name, opts...)
}

// ExplainInScope returns the selected binding or override for type T from the provided scope.
func ExplainInScope[T any](s *di.Scope, opts ...di.Option) (*ResolutionExplanation, error) {
	return di.ExplainInScope[T](s, opts...)
}

// ExplainNamedInScope returns the selected binding or override for the named binding of type T from the provided scope.
func ExplainNamedInScope[T any](s *di.Scope, name string, opts ...di.Option) (*ResolutionExplanation, error) {
	return di.ExplainNamedInScope[T](s, name, opts...)
}

// Validate validates the provided container configuration without constructing instances.
func Validate(c *di.Container) error {
	if c == nil {
		return di.ErrInvalidOption
	}
	return c.Validate()
}

// ValidateScope validates bindings visible from the provided scope.
func ValidateScope(s *di.Scope) error {
	if s == nil {
		return di.ErrInvalidOption
	}
	return s.ValidateBindings()
}

// GraphOf returns the effective dependency graph for the provided container.
func GraphOf(c *di.Container) (Graph, error) {
	if c == nil {
		return Graph{}, di.ErrInvalidOption
	}
	return c.Graph()
}

// DumpGraph returns a readable text dump of the provided container's effective graph.
func DumpGraph(c *di.Container) (string, error) {
	if c == nil {
		return "", di.ErrInvalidOption
	}
	return c.DumpGraph()
}

// DumpGraphDOT returns a Graphviz DOT dump of the provided container's effective graph.
func DumpGraphDOT(c *di.Container) (string, error) {
	if c == nil {
		return "", di.ErrInvalidOption
	}
	return c.DumpGraphDOT()
}

// GraphOfScope returns the effective dependency graph for the provided scope.
func GraphOfScope(s *di.Scope) (Graph, error) {
	if s == nil {
		return Graph{}, di.ErrInvalidOption
	}
	return s.Graph()
}

// DumpGraphScope returns a readable text dump for the provided scope.
func DumpGraphScope(s *di.Scope) (string, error) {
	if s == nil {
		return "", di.ErrInvalidOption
	}
	return s.DumpGraph()
}

// DumpGraphDOTScope returns a Graphviz DOT dump for the provided scope.
func DumpGraphDOTScope(s *di.Scope) (string, error) {
	if s == nil {
		return "", di.ErrInvalidOption
	}
	return s.DumpGraphDOT()
}

// FormatValidation formats validation errors for display.
func FormatValidation(err error) string {
	return di.FormatValidation(err)
}
