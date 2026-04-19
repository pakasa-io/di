package di

import (
	"errors"
	"strings"
	"testing"
)

type autoWireCycleDeps struct {
	Consumer *autoWireCycleConsumer
}

type autoWireCycleConsumer struct {
	Deps autoWireCycleDeps
}

func TestStructAutoWiringDisabledByDefault(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testAutoWireConsumer](container, func(deps testAutoWireDeps) *testAutoWireConsumer {
		return &testAutoWireConsumer{Deps: deps}
	})

	if container.StructAutoWiringEnabled() {
		t.Fatal("expected struct auto-wiring to be disabled by default")
	}

	if _, err := ResolveFrom[*testAutoWireConsumer](container); err == nil {
		t.Fatal("expected Resolve to fail when struct auto-wiring is disabled")
	}

	err := container.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail when struct auto-wiring is disabled")
	}
	if !strings.Contains(err.Error(), cacheKey(getType[testAutoWireDeps](), "")) {
		t.Fatalf("expected validation error to mention the missing struct dependency, got %v", err)
	}

	explanation, explainErr := ExplainFrom[*testAutoWireConsumer](container)
	if explainErr != nil {
		t.Fatalf("Explain failed: %v", explainErr)
	}
	if len(explanation.Dependencies) != 1 || !explanation.Dependencies[0].Missing {
		t.Fatalf("expected Explain to report the struct dependency as missing, got %#v", explanation.Dependencies)
	}

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	foundMissing := false
	for _, node := range graph.Nodes {
		if node.Missing && node.Key == cacheKey(getType[testAutoWireDeps](), "") {
			foundMissing = true
			break
		}
	}
	if !foundMissing {
		t.Fatalf("expected graph to include the missing struct dependency, got %#v", graph.Nodes)
	}
}

func TestStructAutoWiringCanBeEnabledExplicitly(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()

	container.SetStructAutoWiring(true)
	if !container.StructAutoWiringEnabled() {
		t.Fatal("expected explicit struct auto-wiring opt-in to be enabled")
	}

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testAutoWireConsumer](container, func(deps testAutoWireDeps) *testAutoWireConsumer {
		return &testAutoWireConsumer{Deps: deps}
	})

	value, err := ResolveFrom[*testAutoWireConsumer](container)
	if err != nil {
		t.Fatalf("Resolve failed with explicit struct auto-wiring: %v", err)
	}
	if value.Deps.Service == nil || value.Deps.Service.ID != "svc" {
		t.Fatalf("unexpected auto-wired value: %#v", value)
	}

	if err := container.Validate(); err != nil {
		t.Fatalf("Validate failed with explicit struct auto-wiring: %v", err)
	}

	explanation, err := ExplainFrom[*testAutoWireConsumer](container)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}
	if len(explanation.Dependencies) != 1 || !explanation.Dependencies[0].AutoWired {
		t.Fatalf("expected Explain to mark the struct dependency as auto-wired, got %#v", explanation.Dependencies)
	}
	if len(explanation.Dependencies[0].Dependencies) != 1 || explanation.Dependencies[0].Dependencies[0].Missing {
		t.Fatalf("expected auto-wired struct explanation to include resolved fields, got %#v", explanation.Dependencies[0].Dependencies)
	}

	graph, err := container.Graph()
	if err != nil {
		t.Fatalf("GraphOf failed: %v", err)
	}
	foundAutoWired := false
	foundMissing := false
	for _, node := range graph.Nodes {
		switch {
		case node.AutoWired && node.Key == cacheKey(getType[testAutoWireDeps](), ""):
			foundAutoWired = true
		case node.Missing && node.Key == cacheKey(getType[testAutoWireDeps](), ""):
			foundMissing = true
		}
	}
	if !foundAutoWired || foundMissing {
		t.Fatalf("expected graph to include an auto-wired struct node without a matching missing node, got %#v", graph.Nodes)
	}
}

func TestStructAutoWiringEnvEnablesNewContainers(t *testing.T) {
	prepareTest(t)

	t.Setenv(EnvEnableStructAutoWiring, "true")
	resetContainersForTest()
	container := newTestContainer()

	if !container.StructAutoWiringEnabled() {
		t.Fatal("expected env opt-in to enable struct auto-wiring for new containers")
	}

	MustProvideTo[*testService](container, func() *testService {
		return &testService{ID: "svc"}
	})
	MustProvideTo[*testAutoWireConsumer](container, func(deps testAutoWireDeps) *testAutoWireConsumer {
		return &testAutoWireConsumer{Deps: deps}
	})

	value, err := ResolveFrom[*testAutoWireConsumer](container)
	if err != nil {
		t.Fatalf("Resolve failed with env-enabled struct auto-wiring: %v", err)
	}
	if value.Deps.Service == nil || value.Deps.Service.ID != "svc" {
		t.Fatalf("unexpected env auto-wired value: %#v", value)
	}
}

func TestStructAutoWiringDetectsCyclesDuringResolutionAndValidation(t *testing.T) {
	prepareTest(t)
	container := newTestContainer()
	container.SetStructAutoWiring(true)

	MustProvideTo[*autoWireCycleConsumer](container, func(deps autoWireCycleDeps) *autoWireCycleConsumer {
		return &autoWireCycleConsumer{Deps: deps}
	})

	_, err := ResolveFrom[*autoWireCycleConsumer](container)
	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected Resolve to fail with ErrCircularDependency, got %v", err)
	}
	if !strings.Contains(err.Error(), getType[*autoWireCycleConsumer]().String()) || !strings.Contains(err.Error(), getType[autoWireCycleDeps]().String()) {
		t.Fatalf("expected resolution cycle trace to include both the binding and auto-wired struct, got %v", err)
	}

	err = container.Validate()
	if !errors.Is(err, ErrCircularDependency) {
		t.Fatalf("expected Validate to fail with ErrCircularDependency, got %v", err)
	}
	if !strings.Contains(err.Error(), getType[*autoWireCycleConsumer]().String()) || !strings.Contains(err.Error(), getType[autoWireCycleDeps]().String()) {
		t.Fatalf("expected validation cycle trace to include both the binding and auto-wired struct, got %v", err)
	}
}
