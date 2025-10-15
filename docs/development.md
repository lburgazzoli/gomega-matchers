# Development Guidelines

This document outlines the development standards and practices for the gomega-matchers project.

## Project Overview

This library provides additional matchers for [Gomega](https://onsi.github.io/gomega/), focusing on JQ-based matchers for validating JSON structures in tests.

## Development Commands

### Testing
```bash
# Run all tests
make test

# Run tests for a specific package
go test -v ./pkg/matchers/jq/...

# Run a specific test
go test -v ./pkg/matchers/jq -run TestMatcher
```

### Code Quality
```bash
# Run linter
make lint

# Auto-fix linting issues
make lint/fix

# Format code
make fmt

# Check for vulnerabilities
make vulncheck

# Run all quality checks (lint + vulncheck)
make check
```

### Dependency Management
```bash
# Tidy dependencies
make deps

# Clean build cache
make clean
```

## Testing

### Gomega Without Ginkgo

- Use vanilla Gomega assertions for all tests (not Ginkgo BDD style)
- Always use dot imports for Gomega:
  ```go
  import . "github.com/onsi/gomega"
  ```
- Use standard Go testing with `testing.T`
- Use `NewWithT(t)` pattern to create Gomega instance
- Mark tests with `t.Parallel()` where appropriate

**Example:**
```go
func TestMatcher(t *testing.T) {
    t.Parallel()

    g := NewWithT(t)

    g.Expect(`{"a":1}`).Should(
        jq.Match(`.a == 1`),
    )
}
```

## Architecture

### Matcher Structure

The library follows Gomega's matcher interface pattern. All matchers implement `types.GomegaMatcher` with three required methods:
- `Match(actual interface{}) (bool, error)` - performs the matching logic
- `FailureMessage(actual interface{}) string` - returns the failure message
- `NegatedFailureMessage(actual interface{}) string` - returns the negated failure message

### JQ Matchers (`pkg/matchers/jq/`)

The JQ matchers use [gojq](https://github.com/itchyny/gojq) to query and validate JSON structures:

- **`jq.Match(expression)`** - evaluates a JQ expression that returns a boolean
- **`jq.Extract(expression)`** - extracts data using JQ, designed for use with `WithTransform`

#### Type Conversion (`jq_support.go`)

The `toType()` function handles conversion of various input types to JQ-compatible data structures:
- String/`[]byte`/`json.RawMessage` → unmarshaled JSON
- `io.Reader` → read and unmarshaled JSON
- `*gbytes.Buffer` → contents unmarshaled as JSON
- `unstructured.Unstructured` → Kubernetes unstructured objects
- `map` and `slice` types → passed through directly

JSON input must be an object `{}` or array `[]` (validated by checking the first byte).

### Usage Examples

```go
// Direct JSON string matching
Expect(`{"a":1}`).Should(jq.Match(`.a == 1`))

// Combining matchers
Expect(`{"status":{"foo":"bar"}}`).Should(
    And(
        jq.Match(`.status.foo == "bar"`),
        jq.Match(`.status != null`),
    ),
)

// Using Extract with WithTransform
Expect(jsonString).Should(
    WithTransform(jq.Extract(`.status`),
        jq.Match(`.foo == "bar"`),
    ),
)

// Working with Go types
Expect(map[string]any{"a": 1}).Should(
    WithTransform(json.Marshal, jq.Match(`.a == 1`)),
)

// Complex array matching
Expect(`{"Values":[ "foo" ]}`).Should(
    jq.Match(`.Values | if . then any(. == "foo") else false end`),
)
```

## Code Quality

### Linting

Always run the linter before committing changes:

```bash
make lint
```

- Address all linter errors before submitting code
- Linter configuration is defined in `.golangci.yml`
- The project uses golangci-lint v2 with most linters enabled
- Import grouping enforced by `gci`: standard → default → blank → k8s.io → project-specific → dot
- Some complexity linters are disabled (cyclop, gocognit, funlen) for practical reasons
- If you need to disable a linter for a specific case, document why with a comment

## Git Workflow

### Conventional Commits

Use conventional commit messages to maintain a clear and structured git history:

**Format:**
```
<type>: <description>

[optional body]

[optional footer]
```

**Common types:**
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code changes that neither fix bugs nor add features
- `chore`: Maintenance tasks, dependency updates, etc.

**Examples:**
```
feat: add server-side apply support for resources

fix: handle nil pointer in reconciler status update

docs: update installation instructions

test: add field manager verification tests

refactor: simplify error handling in controller
```

For more details, see the [Conventional Commits specification](https://www.conventionalcommits.org/).

## Code Organization

### Error Handling

- Return errors as the last return value
- Use `fmt.Errorf` with `%w` for error wrapping to maintain error chains
- Provide context in error messages about what operation failed

**Example from the codebase:**
```go
if err := json.Unmarshal(in, &data); err != nil {
    return nil, fmt.Errorf("unable to unmarshal result, %w", err)
}
```

### Comments and Documentation

- Comments should clarify **why** something is done, not **what** is being done
- Focus on:
  - Non-obvious business logic or algorithm choices
  - Edge cases and their handling
  - Relationships between components
- Avoid redundant comments that merely restate the code
