# AGENTS.md

This file provides guidance for agents working in this repository.

## Project overview

`gomega-matchers` is a Go library that provides additional matchers and helpers
for [Gomega](https://onsi.github.io/gomega/). It includes:

- JQ-based matchers for validating JSON and Go values in `pkg/matchers/jq/`.
- Kubernetes helpers and matchers for typed and unstructured resources in
  `pkg/matchers/k8s/`.

## Repository guide

- [README.md](README.md) contains installation instructions and usage examples
  for the JQ matchers and Kubernetes helpers.
- [docs/development.md](docs/development.md) contains development guidelines,
  testing conventions, dependency workflow, and contribution guidance.
- [docs/architecture.md](docs/architecture.md) documents package boundaries,
  conversion behavior, and Kubernetes retry semantics.
- [Makefile](Makefile) is the canonical interface for formatting, testing,
  dependency management, linting, and vulnerability checks.
- [go.mod](go.mod) defines the module, Go version, and dependencies.

## Agent quick start

Before changing code:

1. Check `git status` and preserve unrelated worktree changes.
2. Read the relevant package implementation and tests.
3. Use the package-level APIs and existing test patterns unless the change is
   specifically about their design.
4. Keep the change focused and update documentation when public behavior or
   architecture changes.

Use [docs/development.md](docs/development.md) for the complete workflow and
[docs/architecture.md](docs/architecture.md) for behavior that is easy to
misinterpret from the API alone.

## Development commands

Use the Makefile targets for routine development tasks:

```bash
make fmt         # Format the code
make test        # Run all tests
make deps        # Tidy dependencies
make lint        # Run the linter
make vulncheck   # Check for known vulnerabilities
make check       # Run lint and vulnerability checks
```

The module currently requires Go 1.26.7. Run `make test` and `make check`
before submitting changes.

## API and design rules

- Use package-level generic Kubernetes functions when a concrete typed result
  is important.
- Use `k8s.Using(client)` when sharing a client is the main goal. Its methods
  return `client.Object` because Go does not support type parameters on
  methods, so it is not a replacement for typed package-level calls.
- Use `client.Reader` for read-only operations, `client.Writer` for
  delete-only operations, and `client.Client` when an operation mixes reads
  and writes. Do not introduce a custom composite reader/writer interface.
- Use `jq` for arbitrary field traversal. Add a Kubernetes extractor only when
  it provides Kubernetes-specific semantics, such as resource quantities or
  named volume/container lookup.
- `Events` means `events.k8s.io/v1`; use `CoreEvents` for legacy `core/v1`
  events.
- Use `jq.New()` for isolated converter state, especially in tests. Do not
  reintroduce global converter reset APIs.
- Strict JQ operations require exactly one result; use the `All` variants when
  multiple results are intentional.
- Gomega's `WithT(t)` does not expose the test context. Pass `t.Context()` to
  `Eventually` or another context-aware operation explicitly.

Kubernetes tests should use the fake client unless a change specifically
requires an integration environment. Typed and unstructured behavior should
both be covered when an API supports both.

## Change guidelines

- Follow the conventions in [docs/development.md](docs/development.md).
- Keep documentation links relative to the repository root.
- Use conventional commit messages for commits.
- Keep one logical change per commit and report the validation commands run.
