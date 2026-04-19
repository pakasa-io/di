# Contributing

Thanks for contributing to `di`.

## Development Setup

Requirements:

- Go version compatible with [go.mod](go.mod)
- `git`

Clone the repository and run the standard checks from the module root:

```bash
go test ./...
go test -race ./...
go vet ./...
go install honnef.co/go/tools/cmd/staticcheck@latest
"$(go env GOPATH)/bin/staticcheck" ./...
```

## Project Expectations

When changing public behavior:

- add or update tests
- update `README.md` when the change affects the main usage path
- update `docs/guides/` when the change adds or changes a user-facing feature
- preserve the non-panicking default API shape where possible; `Must*` helpers are the panic-oriented opt-in path

When changing exported APIs:

- prefer additive changes over breaking changes
- keep naming and generic API shape consistent with the existing surface
- include examples or documentation for new entry points

## Pull Requests

Before opening a pull request:

1. Ensure the worktree is clean except for the intended changes.
2. Run the checks listed above.
3. Summarize the behavior change clearly.
4. Call out any breaking changes explicitly.

## Documentation

Useful repo entry points:

- [README.md](README.md)
- [docs/guides/README.md](docs/guides/README.md)
- [RELEASING.md](RELEASING.md)
