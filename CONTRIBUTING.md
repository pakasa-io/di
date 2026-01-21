
---

# `CONTRIBUTING.md` (MIT-aligned)

```md
# Contributing to pakasa-io/di

Thank you for considering contributing to this project.

This library is intended to be a small, dependable piece of Go infrastructure.
Contributions should favor clarity, correctness, and long-term maintainability
over cleverness.

---

## License

By contributing to this repository, you agree that your contributions will be
licensed under the **MIT License**, the same license as the project.

You retain copyright to your contributions.

No Contributor License Agreement (CLA) is required.

---

## What Makes a Good Contribution

We especially welcome contributions that:

- improve developer experience or error clarity
- simplify APIs without reducing correctness
- add focused tests for edge cases
- improve documentation or examples
- fix real bugs with clear reproduction steps

Please open an issue before making large or architectural changes.

---

## Coding Guidelines

- Follow standard Go formatting (`gofmt`)
- Prefer explicitness over abstraction
- Avoid unnecessary reflection or global state
- Keep exported APIs small and intentional
- Public types and functions must have doc comments

---

## Tests

- All new functionality should include tests
- Bug fixes should include a failing test first
- Run all tests before submitting:

```bash
go test ./...
