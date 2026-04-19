// Package di provides typed dependency injection for Go using explicit
// [Container] and [Scope] values.
//
// The package centers on a small set of primitives:
//
//   - [NewContainer] to create a root container
//   - [ProvideTo] and [BindTo] to register bindings
//   - [ResolveFrom] to resolve values from a container
//   - [ResolveInScope] to resolve scoped values from a [Scope]
//   - [Named] and [Dep] with [WithDeps] for explicit dependency declarations
//   - [Optional] for dependencies that may be absent without failing resolution
//   - [ExplainFrom], [FormatValidation], and graph helpers for debugging
//
// The module's public surface is intentionally split:
//
//   - package di for explicit instance-oriented APIs
//   - package [github.com/pakasa-io/di/global] for process-wide default and named container helpers
//   - package [github.com/pakasa-io/di/diagnostics] for validation, explanation, graph, and metadata helpers
//   - package [github.com/pakasa-io/di/testutil] for test fixtures and isolated container setup
//
// Implicit struct auto-wiring is disabled by default. Opt in per container with
// [Container.SetStructAutoWiring] or for newly created containers with
// [EnvEnableStructAutoWiring].
package di
