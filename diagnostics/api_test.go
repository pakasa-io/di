package diagnostics_test

import (
	"strings"
	"testing"

	di "github.com/pakasa-io/di"
	didiag "github.com/pakasa-io/di/diagnostics"
)

type service struct {
	ID string
}

type dependent struct {
	Service *service
}

type groupValue struct {
	ID int
}

type alias interface {
	AliasID() string
}

type aliasImpl struct{}

func (*aliasImpl) AliasID() string { return "alias" }

func TestDiagnosticsContainerAndScopeHelpers(t *testing.T) {
	container := di.NewContainer()
	di.MustProvideTo[*service](container, func() *service { return &service{ID: "svc"} })
	di.MustProvideNamedTo[*service](container, "named", func() *service { return &service{ID: "named"} })
	di.MustProvideGroupTo[*groupValue](container, "helpers", func() *groupValue {
		return &groupValue{ID: 1}
	}, di.WithName("one"), di.WithLifetime(di.LifetimeTransient))
	di.MustProvideAsTo[*aliasImpl, alias](container, func() *aliasImpl { return &aliasImpl{} })
	di.MustProvideTo[*dependent](container, func(s *service) *dependent { return &dependent{Service: s} })

	if !didiag.Has[*service](container) {
		t.Fatal("expected Has to report true")
	}
	if !didiag.HasNamed[*service](container, "named") {
		t.Fatal("expected HasNamed to report true")
	}

	info, err := didiag.Describe[*service](container)
	if err != nil || info.Type != di.Dep[*service]().Type {
		t.Fatalf("Describe failed: %#v %v", info, err)
	}
	namedInfo, err := didiag.DescribeNamed[*service](container, "named")
	if err != nil || namedInfo.Name != "named" {
		t.Fatalf("DescribeNamed failed: %#v %v", namedInfo, err)
	}

	if bindings := didiag.ListBindings(container); len(bindings) < 4 {
		t.Fatalf("expected local bindings, got %#v", bindings)
	}
	containerInfo := didiag.DescribeContainer(container)
	if containerInfo.BindingCount < 4 || containerInfo.Closed {
		t.Fatalf("unexpected container info: %#v", containerInfo)
	}

	explanation, err := didiag.Explain[*dependent](container)
	if err != nil || len(explanation.Dependencies) != 1 {
		t.Fatalf("Explain failed: %#v %v", explanation, err)
	}
	namedExplanation, err := didiag.ExplainNamed[*service](container, "named")
	if err != nil || namedExplanation.Key == "" {
		t.Fatalf("ExplainNamed failed: %#v %v", namedExplanation, err)
	}

	scope := container.MustNewScope()
	defer scope.Close()

	scopeExplanation, err := didiag.ExplainInScope[*dependent](scope)
	if err != nil || len(scopeExplanation.Dependencies) != 1 {
		t.Fatalf("ExplainInScope failed: %#v %v", scopeExplanation, err)
	}
	namedScopeExplanation, err := didiag.ExplainNamedInScope[*service](scope, "named")
	if err != nil || namedScopeExplanation.Key == "" {
		t.Fatalf("ExplainNamedInScope failed: %#v %v", namedScopeExplanation, err)
	}

	if err := didiag.Validate(container); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if err := didiag.ValidateScope(scope); err != nil {
		t.Fatalf("ValidateScope failed: %v", err)
	}

	graph, err := didiag.GraphOf(container)
	if err != nil || len(graph.Nodes) < 2 {
		t.Fatalf("GraphOf failed: %#v %v", graph, err)
	}
	if dump, err := didiag.DumpGraph(container); err != nil || !strings.Contains(dump, "service") {
		t.Fatalf("DumpGraph failed: %q %v", dump, err)
	}
	if dot, err := didiag.DumpGraphDOT(container); err != nil || !strings.Contains(dot, "digraph di") {
		t.Fatalf("DumpGraphDOT failed: %q %v", dot, err)
	}

	scopeGraph, err := didiag.GraphOfScope(scope)
	if err != nil || len(scopeGraph.Nodes) < 2 {
		t.Fatalf("GraphOfScope failed: %#v %v", scopeGraph, err)
	}
	if dump, err := didiag.DumpGraphScope(scope); err != nil || !strings.Contains(dump, "service") {
		t.Fatalf("DumpGraphScope failed: %q %v", dump, err)
	}
	if dot, err := didiag.DumpGraphDOTScope(scope); err != nil || !strings.Contains(dot, "digraph di") {
		t.Fatalf("DumpGraphDOTScope failed: %q %v", dot, err)
	}
}

func TestDiagnosticsNilAndValidationFormatting(t *testing.T) {
	if didiag.Has[*service](nil) {
		t.Fatal("expected nil Has to return false")
	}
	if _, err := didiag.Describe[*service](nil); err == nil {
		t.Fatal("expected Describe(nil) to fail")
	}
	if err := didiag.Validate(nil); err == nil {
		t.Fatal("expected Validate(nil) to fail")
	}
	if _, err := didiag.GraphOf(nil); err == nil {
		t.Fatal("expected GraphOf(nil) to fail")
	}
	if _, err := didiag.GraphOfScope(nil); err == nil {
		t.Fatal("expected GraphOfScope(nil) to fail")
	}

	broken := di.NewContainer()
	di.MustProvideTo[*dependent](broken, func(s *service) *dependent { return &dependent{Service: s} })

	err := didiag.Validate(broken)
	if err == nil {
		t.Fatal("expected broken container validation to fail")
	}
	formatted := didiag.FormatValidation(err)
	if !strings.Contains(formatted, "validation failed") || !strings.Contains(formatted, "graph:") {
		t.Fatalf("unexpected formatted validation output: %q", formatted)
	}
}
