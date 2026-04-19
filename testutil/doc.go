// Package testutil provides helpers for isolated DI setup in tests.
//
// The package is designed for tests that want short-lived containers, request-
// style scopes, and temporary overrides without leaking state across test
// cases.
//
// The most commonly used helpers are:
//
//   - [Container] for an isolated named container
//   - [Scope] for an isolated scoped context
//   - [ScopedFixture] for a container plus scope bundle
//   - [Override] for temporary runtime overrides with automatic cleanup
package testutil
