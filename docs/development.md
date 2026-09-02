# Development Guide

This guide is for contributors changing the library. Start with the root
[README](../README.md) if you are using the library, and see
[architecture.md](architecture.md) for the implementation model.

## Repository map

| Path | Purpose |
| --- | --- |
| `pkg/matchers/jq/` | JQ matchers, transforms, input conversion, and tests |
| `pkg/matchers/k8s/` | Kubernetes operations, extractors, matchers, and tests |
| `pkg/matchers/k8s/condition/` | Matchers for Kubernetes condition fields |
| `docs/` | Contributor and architecture documentation |
| `Makefile` | Canonical interface for local development and CI checks |
| `.github/workflows/` | Push and pull-request build checks |
| `go.mod` | Module metadata and dependency versions |

Tests are colocated with the package they exercise. Most tests use external
test packages (`jq_test` or `k8s_test`) so they verify the public API; the
small number of internal tests are reserved for package implementation details.

## Prerequisites

- Go 1.26.7, as declared by `go.mod`.
- Network access the first time tools or modules need to be downloaded.
- No Kubernetes cluster is required for the test suite. Kubernetes tests use
  controller-runtime's fake client.

Check the local toolchain with:

```bash
go version
```

## Standard workflow

Use the Makefile targets for routine tasks. CI runs `make check` followed by
`make test` for both pushes to `main` and pull requests.

```bash
make deps        # Tidy go.mod and go.sum after dependency changes
make fmt         # Run the configured formatter and gofmt
make test        # Run all package tests
make lint        # Run golangci-lint
make vulncheck   # Run govulncheck
make check       # Run lint and vulnerability checks
make clean       # Remove Go build and test caches
```

The normal final verification is:

```bash
make test
make check
```

For a focused test, direct `go test` is appropriate because the Makefile only
provides the all-package test target:

```bash
go test -v ./pkg/matchers/jq/...
go test -v ./pkg/matchers/k8s/...
go test -v ./pkg/matchers/jq -run TestMatcher
```

## Testing conventions

Tests use the standard `testing` package with vanilla Gomega assertions, not
Ginkgo. Create the assertion object with `NewWithT(t)` and use `t.Parallel()`
for tests that do not share mutable state.

```go
func TestMatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	g.Expect(`{"a":1}`).Should(jq.Match(`.a == 1`))
}
```

When adding or changing behavior, cover both successful and error paths. In
particular:

- JQ tests should cover conversion, expression parsing, evaluation, result
  cardinality, and zero-result behavior where relevant. Test explicit `null`
  separately from no result, and use `ExtractAll`/`TransformAll` for intentional
  multi-result queries.
- Kubernetes tests should cover typed and unstructured inputs when the API
  supports both, and should use the fake client for client operations.
- Eventually-compatible operations should be tested for retryable errors and
  terminal errors (`StopTrying`) separately.
- Tests should assert the public result or error contract rather than private
  helper calls unless an internal edge case is the subject of the test.

## Formatting and linting

Run `make fmt` instead of formatting individual files manually. The formatter
configuration is in `.golangci.yml`; it enforces the project's import grouping
and runs gofmt-related formatters.

Before committing, run `make lint` and address all issues. If a linter must be
disabled for a non-obvious case, document the reason next to the exception.
Prefer a code change over a new suppression.

### JQ instances and converters

The package-level JQ API uses an unexported shared instance. `jq.New()` creates
an isolated `jq.Instance` with the built-in converters, so prefer it when a test
or feature needs custom conversion rules without global state:

```go
instance := jq.New()
if err := instance.RegisterConverter(myConverter); err != nil {
	t.Fatal(err)
}

matcher := instance.Match(`.value == "expected"`)
```

Converters should return `jq.ErrTypeNotSupported` for inputs they do not own.
Registration rejects nil converters. Use `jq.ResetConverters()` only when a
test deliberately exercises the package-level registry; isolated instances do
not need cleanup shared with other tests.

Strict JQ operations (`Match`, `Extract`, and `Transform`) require exactly one
query result. Add tests for no result, one result, explicit `null`, and multiple
results when changing query execution. Use the `All` variants when multiple
results are part of the intended contract.

## Dependency changes

Keep direct dependencies in the first `require` block of `go.mod` and let
`make deps` maintain the module graph and `go.sum` entries. After changing a
dependency:

1. Run `make deps`.
2. Run the affected focused tests.
3. Run `make test` and `make check`.

Kubernetes modules must stay on a version line supported by the selected
controller-runtime release. Keep related Kubernetes modules aligned unless
there is a documented reason not to.

## Adding or changing API

For a new exported matcher, operation, extractor, mutator, or converter:

1. Put it in the package that owns the behavior.
2. Add a Go doc comment and a focused test.
3. Update the relevant usage section in [README.md](../README.md).
4. Update [architecture.md](architecture.md) when the change affects data
   flow, conversion rules, retry behavior, or package responsibilities.
5. Run `make fmt`, `make test`, and `make check`.

Keep README examples concise and move implementation or behavioral detail into
the contextual documentation under `docs/`.

## Git and pull requests

Use conventional commit messages:

```text
<type>: <description>
```

Typical types are `feat`, `fix`, `docs`, `test`, `refactor`, and `chore`.
Keep commits small and logically scoped. Pull requests should explain the
behavioral change, mention any compatibility considerations, and include the
verification commands that were run.

## Documentation maintenance

When moving or renaming documentation, update links in the same change. Keep
contributor workflow in this guide, implementation context in
[architecture.md](architecture.md), user-facing examples in
[README.md](../README.md), and agent workflow guidance in
[AGENTS.md](../AGENTS.md).
