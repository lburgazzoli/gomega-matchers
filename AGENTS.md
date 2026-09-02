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

## Change guidelines

- Follow the conventions in [docs/development.md](docs/development.md).
- Keep documentation links relative to the repository root.
- Use conventional commit messages for commits.
