package di

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

func TestGraphDumpAndDOT(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	if len(graph.Nodes) != 2 {
		t.Fatalf("expected 2 graph nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 1 {
		t.Fatalf("expected 1 graph edge, got %d", len(graph.Edges))
	}

	dump, err := container.DumpGraph()
	if err != nil {
		t.Fatalf("DumpGraph failed: %v", err)
	}
	if !strings.Contains(dump, getType[*testDependent]().String()) || !strings.Contains(dump, getType[*testService]().String()) {
		t.Fatalf("expected graph dump to mention both bindings, got %q", dump)
	}

	dot, err := container.DumpGraphDOT()
	if err != nil {
		t.Fatalf("DumpGraphDOT failed: %v", err)
	}
	if !strings.Contains(dot, "digraph di") || !strings.Contains(dot, getType[*testDependent]().String()) || !strings.Contains(dot, getType[*testService]().String()) {
		t.Fatalf("expected DOT dump to include graph metadata, got %q", dot)
	}
}

func TestGraphIncludesMissingDependencies(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testDependent](container, func(s *testService) *testDependent {
		return &testDependent{Service: s}
	})

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	foundMissing := false
	for _, node := range graph.Nodes {
		if node.Missing && node.Key == cacheKey(getType[*testService](), "") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected graph to include missing dependency node, got %#v", graph.Nodes)
	}
}

func TestMetricsAndInstrumentation(t *testing.T) {
	prepareTest(t)

	container := newTestContainer()
	var resolveEvents atomic.Int32
	var instanceEvents atomic.Int32

	container.SetInstrumentation(Instrumentation{
		OnResolve: func(event ResolveEvent) {
			if event.Key == cacheKey(getType[*testService](), "") && event.Err == nil {
				resolveEvents.Add(1)
			}
		},
		OnInstanceCreated: func(event InstanceEvent) {
			if event.Key == cacheKey(getType[*testService](), "") && event.Err == nil {
				instanceEvents.Add(1)
			}
		},
	})
	defer container.SetInstrumentation(Instrumentation{})

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	}, WithLifetime(LifetimeTransient))

	container.ResetMetrics()

	value, err := ResolveFrom[*testService](container)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if value.ID != "svc" {
		t.Fatalf("unexpected resolved value: %#v", value)
	}

	metrics := container.Metrics()
	if metrics.Resolutions != 1 {
		t.Fatalf("expected 1 resolution, got %#v", metrics)
	}
	if metrics.InstancesCreated != 1 {
		t.Fatalf("expected 1 created instance, got %#v", metrics)
	}
	if metrics.ResolutionErrors != 0 || metrics.InstanceCreationErrors != 0 {
		t.Fatalf("expected zero metric errors, got %#v", metrics)
	}
	if resolveEvents.Load() != 1 || instanceEvents.Load() != 1 {
		t.Fatalf("expected instrumentation callbacks once, got resolve=%d instance=%d", resolveEvents.Load(), instanceEvents.Load())
	}

	restore, err := OverrideInContainer[*testService](container, func() (*testService, error) {
		return &testService{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("OverrideE failed: %v", err)
	}
	defer restore()

	if _, err := ResolveFrom[*testService](container); err != nil {
		t.Fatalf("Resolve with override failed: %v", err)
	}
	if container.Metrics().OverrideCalls != 1 {
		t.Fatalf("expected override usage to be tracked, got %#v", container.Metrics())
	}

	container.ResetMetrics()
	if got := container.Metrics(); got.Resolutions != 0 || got.InstancesCreated != 0 || got.OverrideCalls != 0 {
		t.Fatalf("expected ResetMetrics to clear counters, got %#v", got)
	}
}

func TestGraphIncludesOverrides(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	restore, err := OverrideInContainer[*testService](container, func() (*testService, error) {
		return &testService{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("OverrideE failed: %v", err)
	}
	defer restore()

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	foundOverride := false
	for _, node := range graph.Nodes {
		if node.Override && node.Key == cacheKey(getType[*testService](), "") {
			foundOverride = true
			break
		}
	}
	if !foundOverride {
		t.Fatalf("expected graph to include override node, got %#v", graph.Nodes)
	}

	dump, err := container.DumpGraph()
	if err != nil {
		t.Fatalf("DumpGraph failed: %v", err)
	}
	if !strings.Contains(dump, "override") {
		t.Fatalf("expected graph dump to mention overrides, got %q", dump)
	}
}

func TestGraphDoesNotDuplicateOverriddenBindings(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})

	restore, err := OverrideInContainer[*testService](container, func() (*testService, error) {
		return &testService{ID: "override"}, nil
	})
	if err != nil {
		t.Fatalf("OverrideE failed: %v", err)
	}
	defer restore()

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	count := 0
	for _, node := range graph.Nodes {
		if node.Key == cacheKey(getType[*testService](), "") {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one node for overridden binding, got %#v", graph.Nodes)
	}
}

func TestGraphReturnsClosedErrorForContainerAndScope(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})

	scope := container.MustNewScope()
	if err := scope.Close(); err != nil {
		t.Fatalf("scope close failed: %v", err)
	}
	if _, err := scope.Graph(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected scope.Graph after close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := scope.DumpGraph(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected scope.DumpGraph after close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := scope.DumpGraphDOT(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected scope.DumpGraphDOT after close to fail with ErrScopeClosed, got %v", err)
	}

	closedContainer := newTestContainer()
	MustProvideTo[*testService](closedContainer, func() *testService {
		return &testService{ID: "svc"}
	})
	if err := closedContainer.Close(); err != nil {
		t.Fatalf("container close failed: %v", err)
	}
	if _, err := closedContainer.Graph(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected container.Graph after close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := closedContainer.DumpGraph(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected container.DumpGraph after close to fail with ErrScopeClosed, got %v", err)
	}
	if _, err := closedContainer.DumpGraphDOT(); !errors.Is(err, ErrScopeClosed) {
		t.Fatalf("expected container.DumpGraphDOT after close to fail with ErrScopeClosed, got %v", err)
	}
}
