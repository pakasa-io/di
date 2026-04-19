# Releasing

This repository is a Go module published directly from Git tags. Use this checklist for public releases and `pkg.go.dev` indexing.

Module path:

- `github.com/pakasa-io/di`

## Pre-Release Checks

From the repository root:

```bash
go mod tidy
go test ./...
go test -race ./...
go vet ./...
go install honnef.co/go/tools/cmd/staticcheck@latest
"$(go env GOPATH)/bin/staticcheck" ./...
```

Confirm before tagging:

- `README.md` reflects the current public API
- public package docs and examples are still accurate
- `LICENSE` is present
- the worktree is clean

## Versioning

Use semantic version tags.

Examples:

- `v0.1.0` for the first public release
- `v0.2.0` for new features before `v1`
- `v1.0.0` once the public API is considered stable

Important:

- do not modify a published tag
- if a release is bad, publish a new version instead of force-moving the tag

## Release Steps

1. Commit the release-ready changes.
2. Create a tag.

```bash
git tag v0.1.0
```

3. Push the branch and tag.

```bash
git push origin HEAD
git push origin v0.1.0
```

4. Trigger Go proxy and `pkg.go.dev` indexing.

```bash
GOPROXY=proxy.golang.org go list -m github.com/pakasa-io/di@v0.1.0
```

5. Verify the public package page:

- `https://pkg.go.dev/github.com/pakasa-io/di`

Check for:

- valid module details
- detected MIT license
- tagged version
- rendered README
- rendered examples and package docs

## After Release

- update any release notes or changelog if you maintain one
- verify the repo is still green in CI
- if this is the first stable release, ensure docs refer to `v1` stability expectations clearly
