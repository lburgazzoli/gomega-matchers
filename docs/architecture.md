# Architecture

This document describes the behavior and boundaries of the library's main
packages. It is intended for contributors who need to extend an existing
package or decide where new behavior belongs.

## Runtime model

The library adapts two kinds of behavior to Gomega's assertion model:

```text
caller
  ├─ jq.Match / jq.Extract / jq.Transform
  │    └─ Convert → gojq query execution → Gomega matcher or transform result
  └─ k8s.Get / k8s.Update / k8s.List / ...
       └─ client.Client operation → result or error for Expect/Eventually
```

The packages do not own a Gomega test lifecycle. They return matchers or
functions that callers pass to `Expect`, `Eventually`, or
`WithTransform`.

## Package boundaries

```text
pkg/matchers/jq/
  ├─ conversion and numeric normalization
  ├─ JQ matchers and transforms
  └─ gojq integration

pkg/matchers/k8s/
  ├─ client operations and retry semantics
  ├─ typed/unstructured conversion
  ├─ object matchers and field extractors
  └─ event options and metadata mutators

pkg/matchers/k8s/condition/
  └─ matchers for condition type, status, reason, and message
```

The Kubernetes package may use the JQ package in caller-facing examples, but
the package responsibilities remain separate: `k8s` handles Kubernetes object
semantics and `jq` handles JSON conversion and query evaluation.

## JQ package

### Conversion pipeline

`jq.Match`, `jq.Extract`, and `jq.Transform` compile their expression when the
function is created. On invocation they:

1. Convert the input through the global converter registry.
2. Normalize numeric values into types supported by gojq.
3. Execute the compiled query.
4. Return a boolean matcher result or the first transform result.

Built-in converters accept JSON strings, byte slices, `json.RawMessage`,
Gomega byte buffers, maps, and slices. JSON-encoded inputs must contain an
object or array; JSON primitives and `null` are rejected. Maps and slices are
passed through and then recursively normalized when they contain
`map[string]any` or `[]any` values.

### Converter registry

The registry is global and protected by a read/write mutex. A registered
converter is prepended, so user converters run before built-ins. A converter
that does not handle an input must return `jq.ErrTypeNotSupported`; any other
error stops conversion immediately.

Because registration is global, tests that register converters must restore the
registry with `jq.ResetConverters()` in cleanup. Avoid relying on registration
order across tests.

### Query result semantics

- `jq.Match` requires the first query result to be a boolean.
- `jq.Extract` returns the first result and returns `nil` when the query
  produces no result.
- `jq.Transform` returns the first result and reports an error when the query
  produces no result.
- Additional results from a multi-result query are intentionally discarded.
- Parse and evaluation errors are terminal when used with Gomega's
  `Eventually`, so callers do not waste retries on invalid expressions.

## Kubernetes package

### Client operations

Operations return context-aware functions so they can be used directly with
Gomega's `Eventually`:

- `Get` clones the prototype and returns the fetched object without mutating
  the caller's input.
- `Lookup` fetches into the caller-provided object and returns only an error.
- `Create`, `Update`, `StatusUpdate`, and `Upsert` return the object after a
  final fetch, so server-side defaults and mutations are visible in the result.
- `Update` accepts either a typed callback (`func(T)`) or a reusable
  `func(client.Object)` mutator.
- `Upsert` updates an existing object and creates only when the initial fetch
  returns a Kubernetes NotFound error.
- `List` returns a copied list object, while `Events` returns a plain slice of
  typed events for use with standard Gomega collection matchers.

`Singleton` and `LookupSingleton` list objects and require exactly one match.
No matches produce a NotFound error, which is retryable by `Eventually`; more
than one match produces `StopTrying` because retrying cannot make the current
result unambiguous.

### Absence and not-found semantics

`Absent` returns true for a missing object or a resource type without a REST
mapping. `NotFound` returns true only for an HTTP 404 for the requested object.
Unexpected errors and unsupported situations are terminal.

### Typed and unstructured objects

Operations and matchers accept controller-runtime `client.Object` values. The
implementation supports both typed Kubernetes objects and
`*unstructured.Unstructured` values. When a typed result must be reconstructed
from an unstructured list item, conversion uses the Kubernetes runtime
converter.

Group-version-kind matchers read `TypeMeta`. This metadata is normally
available on unstructured objects and real API-server responses, but may be
empty on typed objects returned by the controller-runtime fake client. Tests
that assert GVK should set `TypeMeta` explicitly or use unstructured objects.

### Extractors, matchers, and mutators

Extractors return functions for `WithTransform`:

- `Data`, `Finalizers`, and `ListItems` expose common object fields.
- `Conditions` returns generic condition maps; `ConditionsOf[T]` converts them
  to a concrete condition type.
- `PodTemplate`, `Containers`, `Container`, `EnvVars`, and `EnvVar` traverse
  common pod, workload, and CronJob shapes.

Metadata matchers such as `HasName`, `HasNamespace`, `HasLabel`, and
`HasAnnotation` are implemented as Gomega transforms. `HasOwnerReference`,
`IsControlledBy`, deletion, and GVK matchers follow the same composition model.

`SetLabel` and `SetAnnotation` copy existing maps before changing them, and
`Apply` composes mutators in order. This keeps reusable mutators from
unexpectedly modifying a map shared by another object.

## Extension points

Choose an extension point based on the desired caller experience:

| Need | Extension point |
| --- | --- |
| Assert a JQ expression returns true | `jq.Match` / `jq.Matchf` |
| Feed a selected JQ value into another matcher | `jq.Extract` / `jq.Extractf` |
| Mutate data through a JQ expression | `jq.Transform` / `jq.Transformf` |
| Support a new input type for JQ | `jq.RegisterConverter` |
| Assert a Kubernetes field | A `k8s` matcher or extractor composed with Gomega |
| Add a Kubernetes client operation | A context-aware function compatible with `Expect`/`Eventually` |
| Match condition fields | A matcher in `k8s/condition` |

New public behavior should include a package-level Go doc comment, focused
tests, and a concise usage example in the [README](../README.md).
