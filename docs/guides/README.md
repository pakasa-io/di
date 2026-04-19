# Guides

These guides cover the main `di` workflows from first registration to production-oriented graph design, diagnostics, testing, and creative composition patterns.

The module has three public packages:

- `github.com/pakasa-io/di` for explicit container and scope usage
- `github.com/pakasa-io/di/global` for process-wide default and named containers
- `github.com/pakasa-io/di/diagnostics` for validation, graph, explanation, and metadata helpers

## Which Package Should I Use?

- Start with `di` when you want explicit wiring, testable container ownership, and request/job scopes.
- Use `global` when the app naturally has one default graph or a few process-wide named graphs.
- Use `diagnostics` around startup, debugging, admin endpoints, and tooling.

## Recommended Reading Order

If you are new to the library, read the guides in order through guide 8. After that, jump based on your use case.

## Basic

- [01-getting-started.md](01-getting-started.md) for the smallest working container plus startup-time validation patterns
- [02-named-bindings-and-explicit-deps.md](02-named-bindings-and-explicit-deps.md) for multiple bindings of the same type
- [03-groups-and-interface-aliases.md](03-groups-and-interface-aliases.md) for collections and interface-based lookups
- [04-invoke-and-fill-struct.md](04-invoke-and-fill-struct.md) for function calls and struct field injection

## Medium

- [05-lifetimes-and-scopes.md](05-lifetimes-and-scopes.md) for singleton, transient, and scoped behavior
- [06-optional-dependencies.md](06-optional-dependencies.md) for soft dependencies and feature toggles
- [07-global-containers.md](07-global-containers.md) for app-style default and named containers
- [08-diagnostics-and-graphs.md](08-diagnostics-and-graphs.md) for validation, explanation, and graph dumps

## Advanced

- [09-struct-autowiring.md](09-struct-autowiring.md) for opt-in implicit struct dependency resolution
- [10-builder-hooks-and-cleanup.md](10-builder-hooks-and-cleanup.md) for the fluent binding API and lifecycle hooks
- [11-overlays-and-runtime-overrides.md](11-overlays-and-runtime-overrides.md) for inherited registrations and temporary replacements
- [12-introspection-metrics-and-instrumentation.md](12-introspection-metrics-and-instrumentation.md) for metadata and observability
- [13-testing-with-testutil.md](13-testing-with-testutil.md) for isolated test containers and scoped fixtures

## Complex

- [14-complex-patterns.md](14-complex-patterns.md) for multi-feature composition patterns
- [15-best-practices-and-gotchas.md](15-best-practices-and-gotchas.md) for production guidance, design heuristics, and common mistakes
- [16-creative-uses.md](16-creative-uses.md) for less obvious but effective ways to use the library

## Choose By Need

- “I want a working graph quickly”: guides 1, 2, 5
- “I need many implementations or plug-ins”: guides 3, 14, 16
- “I need request/job scoping”: guides 5, 14, 15
- “I need observability and debugging”: guides 8, 12, 15
- “I need test isolation and overrides”: guides 11, 13, 15
- “I want less boilerplate for dependency bundles”: guides 4, 9, 16

## Coverage

This set covers the main public use cases and feature areas:

- container creation and typed resolution
- named bindings and explicit dependency declarations
- groups and interface aliases
- function invocation and struct injection
- lifetimes, scopes, and context-aware resolution
- optional dependencies
- global default and named containers
- validation, formatted errors, explanations, and graph dumps
- struct auto-wiring
- fluent binding builder, pre/post hooks, and close hooks
- overlays and runtime overrides
- binding/container introspection
- instrumentation and metrics
- test fixtures and temporary overrides
- production best practices and sharp edges
- creative graph composition patterns
