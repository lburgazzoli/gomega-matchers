# gomega-matchers

Additional matchers for [Gomega](https://onsi.github.io/gomega/), the Go testing assertion framework. This library provides powerful JQ-based matchers for validating JSON structures in tests.

## Installation

```bash
go get github.com/lburgazzoli/gomega-matchers
```

## Features

- **JQ matchers** - Use [jq](https://jqlang.github.io/jq/) expressions to query and validate JSON structures
- **Flexible input types** - Works with JSON strings, byte slices, raw messages, and Go types (maps, slices, and custom converters)
- **Kubernetes support** - Generic helpers for typed and unstructured Kubernetes objects
- **Composable** - Combine with Gomega's built-in matchers like `And()`, `Or()`, and `WithTransform()`

## Usage

### Basic Examples

```go
import (
    . "github.com/onsi/gomega"
    "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
)

// Simple JSON string matching
Expect(`{"a":1}`).Should(jq.Match(`.a == 1`))

// Match nested fields
Expect(`{"status":{"ready":true}}`).Should(
    jq.Match(`.status.ready == true`),
)

// Negative assertions
Expect(`{"a":1}`).Should(Not(
    jq.Match(`.a == 2`),
))
```

### Working with Go Types

```go
// Match maps directly
Expect(map[string]any{"a": 1}).Should(
    jq.Match(`.a == 1`),
)

// Match structs (marshal to JSON first)
type Config struct {
    Port int `json:"port"`
}

Expect(Config{Port: 8080}).Should(
    WithTransform(json.Marshal, jq.Match(`.port == 8080`)),
)
```

### Custom Type Converters

Register custom converters for your own types to use them directly with JQ matchers:

```go
import "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

type Person struct {
    Name string `json:"name"`
    Age  int    `json:"age"`
}

// Register a converter for Person type
if err := jq.RegisterConverter(func(v any) (any, error) {
    p, ok := v.(Person)
    if !ok {
        return nil, jq.ErrTypeNotSupported
    }

    data, err := json.Marshal(p)
    if err != nil {
        return nil, err
    }

    var result map[string]any
    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }

    return result, nil
}); err != nil {
    panic(err)
}

// Now use Person directly without WithTransform
Expect(Person{Name: "Alice", Age: 30}).Should(
    jq.Match(`.name == "Alice" and .age == 30`),
)
```

### Array Matching

```go
// Check if array contains a value
Expect(`{"Values":["foo","bar"]}`).Should(
    jq.Match(`.Values | if . then any(. == "foo") else false end`),
)

// Check array is empty or null
Expect(`{"Values":null}`).Should(
    Not(jq.Match(`.Values | if . then any(. == "foo") else false end`)),
)
```

### Extracting and Transforming

```go
// Extract nested data for focused assertions
in := `
{
  "status":{
    "foo": {
      "bar": "fr",
      "baz": "fz"
    }
  }
}
`

Expect(in).Should(
    WithTransform(jq.Extract(`.status`),
        And(
            jq.Match(`.foo.bar == "fr"`),
            jq.Match(`.foo.baz == "fz"`),
        ),
    ),
)
```

### Transforming

```go
// Transform data with a JQ expression and get the full result back
result, err := jq.Transform(`. + {"new_field": "value"}`)(inputMap)

// Use with WithTransform to transform then assert
Expect(map[string]any{"status": "pending"}).Should(
    WithTransform(jq.Transform(`.status = "done"`),
        jq.Match(`.status == "done"`),
    ),
)

// Formatted expressions
transform := jq.Transformf(`.data.%s = "%s"`, "key", "new-value")
result, err = transform(inputMap)
```

Use `ExtractAll` or `TransformAll` when a query is expected to produce multiple
results. The strict `Extract` and `Transform` functions return an error for
multiple results, while the `All` variants return every result as a slice.

```go
values, err := jq.ExtractAll(`.items[]`)(inputMap)
allValues, err := jq.TransformAll(`.items[] | .name`)(inputMap)
```

### Combining Matchers

```go
// Use And() to combine multiple assertions
Expect(`{"status":{"foo":"bar","count":42}}`).Should(
    And(
        jq.Match(`.status.foo == "bar"`),
        jq.Match(`.status.count == 42`),
    ),
)
```

## Kubernetes Helpers

The library provides generic helpers for testing Kubernetes resources with Gomega.
Operations are package-level functions. Read-only operations accept a
`client.Reader`, `Delete` accepts a `client.Writer`, and `Create`, `Update`,
`StatusUpdate`, and `Upsert` accept a `client.Client`. `Singleton` and
`LookupSingleton` also require `client.Client` to resolve typed list objects
through its scheme.
Both typed objects (e.g., `*corev1.ConfigMap`) and unstructured objects work with the same functions.

When several operations share a client, bind it once with `Using`. The façade
uses `client.Object` results because Go methods cannot declare type parameters;
use the package-level functions when concrete generic result types are needed.

```go
ops := k8s.Using(cli)
Eventually(ctx, ops.Get(&corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{Name: "my-config", Namespace: "default"},
})).Should(jq.Match(`.data.key == "value"`))
```

Choose the API based on the result you need:

- Use package-level generic functions such as `k8s.Get` or `k8s.Singleton`
  when the returned concrete Kubernetes type matters.
- Use `k8s.Using(cli)` when several operations share a client and a
  `client.Object` result is sufficient.
- Use `jq.Extract` for arbitrary nested fields and `k8s` extractors for
  Kubernetes-specific traversal or semantics, such as named containers,
  volumes, and `resource.Quantity` values.
- Use `Events` for `events.k8s.io/v1` and `CoreEvents` for legacy `core/v1`
  events.

Operations are context-aware, so tests should pass their test context to
Gomega polling:

```go
Eventually(t.Context(), k8s.Get(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{Name: "my-config", Namespace: "default"},
})).Should(...)
```

```go
import (
    . "github.com/onsi/gomega"
    "github.com/onsi/gomega/gstruct"
    "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
    "github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

cli := /* client.Client */

// Get a resource — works with typed and unstructured objects
Eventually(ctx, k8s.Get(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})).Should(jq.Match(`.data.key == "value"`))

// Create a resource
Eventually(ctx, k8s.Create(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
    Data: map[string]string{"key": "value"},
})).Should(jq.Match(`.data.key == "value"`))

// Update with a typed callback — no casting needed
Eventually(ctx, k8s.Update(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}, func(cm *corev1.ConfigMap) {
    cm.Data["key"] = "new-value"
})).Should(jq.Match(`.data.key == "new-value"`))

// Update with a reusable metadata mutator
Eventually(ctx, k8s.Update(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}, k8s.SetLabel("env", "prod"))).Should(
    k8s.HasLabel("env", "prod"),
)

// Compose multiple metadata mutations
Eventually(ctx, k8s.Update(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}, k8s.Apply(
    k8s.SetLabel("env", "prod"),
    k8s.SetAnnotation("team", "platform"),
))).Should(SatisfyAll(
    k8s.HasLabel("env", "prod"),
    k8s.HasAnnotation("team", "platform"),
))

// Update a status subresource
Eventually(ctx, k8s.StatusUpdate(cli, &corev1.Pod{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-pod",
        Namespace: "default",
    },
}, func(pod *corev1.Pod) {
    pod.Status.Phase = corev1.PodSucceeded
})).Should(jq.Match(`.status.phase == "Succeeded"`))

// Create or update idempotently
Eventually(ctx, k8s.Upsert(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}, func(cm *corev1.ConfigMap) {
    if cm.Data == nil {
        cm.Data = map[string]string{}
    }
    cm.Data["key"] = "value"
})).Should(jq.Match(`.data.key == "value"`))

// Delete a resource
Eventually(ctx, k8s.Delete(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})).Should(Succeed())

// Check if a resource is absent (tolerates missing CRDs)
Eventually(ctx, k8s.Absent(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})).Should(BeTrue())

// Check if a specific resource is not found (HTTP 404 only)
Eventually(ctx, k8s.NotFound(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})).Should(BeTrue())

// List resources
Eventually(ctx, k8s.List(cli, &corev1.ConfigMapList{},
    client.InNamespace("default"),
)).Should(WithTransform(k8s.ListItems(), HaveLen(2)))

// Select exactly one typed object via list options
Eventually(ctx, k8s.Singleton[*corev1.ConfigMap](cli,
    client.InNamespace("default"),
    client.MatchingLabels{"app": "frontend"},
)).Should(HaveField("Data", HaveKeyWithValue("key", "value")))

// Lookup exactly one object and write it into an output value
selected := &corev1.ConfigMap{}
Eventually(ctx, k8s.LookupSingleton(cli, selected,
    client.InNamespace("default"),
    client.MatchingLabels{"app": "frontend"},
)).Should(Succeed())
Expect(selected.Name).To(Equal("frontend-config"))

// Query events — use standard Gomega matchers on the result.
// Events uses events.k8s.io/v1. Use CoreEvents for legacy core/v1 events.
Eventually(ctx, k8s.Events(cli,
    k8s.InNamespace("default"),
    k8s.ForObject(corev1.ObjectReference{
        Kind: "Pod",
        Name: "my-pod",
    }),
)).Should(ContainElement(
    gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
        "Reason": Equal("Ready"),
    }),
))

// Extract .data from ConfigMap, Secret, or unstructured objects
Expect(configMap).Should(
    WithTransform(k8s.Data(), HaveKeyWithValue("key", "value")),
)

// Extract finalizers
Expect(obj).Should(
    WithTransform(k8s.Finalizers(), ContainElement("example.com/finalizer")),
)

// Extract typed pod volumes from a workload
Expect(deployment).Should(
    WithTransform(k8s.Volume("config"), HaveField("ConfigMap.Name", Equal("settings"))),
)

// Compare resource quantities semantically, including minimums and limits
Expect(deployment).Should(
    WithTransform(
        k8s.Container("app"),
        WithTransform(k8s.ResourceRequests(), HaveKeyWithValue(
            corev1.ResourceCPU, k8s.CompareQuantity(">=", "500m"),
        )),
    ),
)

// Assert a list is empty
Eventually(ctx, k8s.List(cli, &corev1.ConfigMapList{},
    client.InNamespace("default"),
)).Should(k8s.IsEmptyList())

// Filter events by label selector
Eventually(ctx, k8s.Events(cli,
    k8s.InNamespace("default"),
    k8s.MatchingLabels(client.MatchingLabels{"app": "frontend"}),
)).Should(HaveLen(1))

// Metadata matchers compose with SatisfyAll
Eventually(ctx, k8s.Get(cli, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})).Should(SatisfyAll(
    k8s.HasName("my-config"),
    k8s.HasNamespace("default"),
    k8s.HasLabel("app", "myapp"),
    k8s.HasAnnotation("managed-by", "operator"),
))

// Deletion and finalizer matchers
Expect(pod).Should(SatisfyAll(
    k8s.IsDeleting(),
    k8s.HasFinalizer("example.com/finalizer"),
))

// Owner reference matchers
// Set APIVersion, Kind, and Name on owner, including in fake-client tests.
Expect(child).Should(k8s.HasOwnerReference(owner))
Expect(child).Should(k8s.IsControlledBy(owner))

// GVK matchers — work with unstructured objects and real apiserver responses.
// Note: typed objects from the fake client typically have empty TypeMeta,
// so these matchers are most useful with unstructured objects in unit tests.
Expect(obj).Should(k8s.MatchesGroupVersionKind(schema.GroupVersionKind{
    Group:   "apps",
    Version: "v1",
    Kind:    "Deployment",
}))

// Condition matchers compose with k8s.Conditions() and k8s.ConditionsOf(...)
Expect(obj).Should(WithTransform(k8s.Conditions(), ContainElement(
    condition.Is("Ready", "True"),
)))

Expect(obj).Should(WithTransform(k8s.ConditionsOf[metav1.Condition](), ContainElement(
    SatisfyAll(
        condition.Is("Available", metav1.ConditionTrue),
        condition.HasReason("MinimumReplicasAvailable"),
    ),
)))
```

### Partial resource matching

`MatchPartially` delegates to Kubernetes'
[`apiequality.Semantic.DeepDerivative`](https://pkg.go.dev/k8s.io/apimachinery/pkg/api/equality#Semantic).
It accepts typed resource values or pointers, and unstructured resources:

```go
Expect(pod).To(k8s.MatchPartially(corev1.Pod{
    Spec: corev1.PodSpec{
        Containers: []corev1.Container{
            {Name: "app", Image: "example/backend:v2"},
        },
    },
}))
```

This uses **Kubernetes' native comparison rules**. It does not infer which
fields you explicitly wrote in a Go struct literal:

| Expected value | Native comparison |
| --- | --- |
| Empty string, nil pointer, empty map or slice | Ignored |
| Numeric zero or `false` | Compared, including fields omitted from a typed literal |
| Map entries | Require the keys and partially compare their values; extra keys are allowed |
| Nonempty slices | Compare by position; extra trailing actual elements are allowed |
| Typed resource quantities and timestamps | Compare using Kubernetes semantic equality |

**Typed pattern caveat:** the example also compares zero-valued fields such as
`metadata.generation`, `spec.hostNetwork`, and `metadata.creationTimestamp`.
Consequently, a sparse typed pattern can fail against a real API object even
when the image matches. `corev1.Pod{}` is allowed and follows these same rules;
it is not a reliable existence check. Use `k8s.Get` or `k8s.NotFound` to check
resource existence. A copied live object also constrains its resourceVersion
and other populated metadata.

To select only particular fields, use an unstructured pattern. For a typed
actual object, set its TypeMeta to the matching APIVersion and Kind:

```go
Expect(pod).To(k8s.MatchPartially(unstructured.Unstructured{
    Object: map[string]any{
        "apiVersion": "v1",
        "kind": "Pod",
        "metadata": map[string]any{
            "labels": map[string]any{"app": "backend"},
        },
        "spec": map[string]any{
            "containers": []any{
                map[string]any{"name": "app", "image": "example/backend:v2"},
            },
        },
    },
}))
```

Only keys present in an unstructured pattern participate, but native wildcard
rules still apply: an explicitly present empty string, null, or empty collection
does not assert that the actual value is empty. Use `jq.Match`, `BeEmpty`, or
`BeNil` for those assertions. Unstructured patterns compare serialized maps;
zero-valued typed fields omitted by the Kubernetes converter are absent, so use
an explicit field matcher when asserting those Go values.

**List caveat:** the first container in the pattern matches the first actual
container, even if another container has the requested name. Command arguments
also allow extra trailing elements. For named lookup or exact collections,
compose the existing extractors with Gomega matchers:

```go
Expect(pod).To(WithTransform(k8s.Container("app"),
    HaveField("Image", Equal("example/backend:v2")),
))
Expect(pod).To(jq.Match(`.spec.hostNetwork == false`))
```

Typed comparisons require the same concrete resource type. Mixed typed and
unstructured inputs require matching APIVersion and Kind on both sides; fake
clients may require you to populate TypeMeta explicitly. An unstructured actual
is converted to the typed pattern's type, preserving typed quantity/time
semantics. With an unstructured pattern, quantity strings are literal (`500m`
and `0.5` differ). The matcher performs no defaulting or API version conversion,
copies the pattern at construction, and shows actual and expected objects in
failure messages.

### Conditions for the latest generation

A condition can remain `Ready=True` while a controller is processing a newer
spec. For resources with condition-level `observedGeneration`, assert freshness
against the fetched object's generation:

```go
Eventually(t.Context(), k8s.Get(cli, resource)).Should(
    k8s.HasCurrentCondition("Ready", metav1.ConditionTrue),
)
```

This requires a positive `metadata.generation` and a condition whose
`observedGeneration` equals that generation. Missing or stale conditions remain
retryable mismatches. For resources without condition-level generations, use
the existing `Conditions` and `condition` matchers.

### Deployment rollout completion

```go
Eventually(t.Context(), k8s.Get(cli, deployment)).Should(
    k8s.HasCompletedDeploymentRollout(),
)
```

The matcher requires the latest generation to have been observed, with updated,
total, and available replicas all equal to the desired count. This catches old
replicas remaining during an upgrade. Omitted desired replicas default to one;
an explicit zero is supported. Failure messages show replica counts, generation
lag, and any progress-deadline failure, while allowing polling to continue.

### Quantity comparisons and owner identity

`CompareQuantity` accepts quantity values, non-nil quantity pointers, and quantity
strings. It supports `==`, `!=`, `<`, `<=`, `>`, and `>=` using exact Kubernetes
arithmetic: `500m` equals `0.5`, and `1Gi` equals `1024Mi`.

`HasOwnerReference` and `IsControlledBy` require the expected owner to have
APIVersion, Kind, and Name set. They compare API group, Kind, and Name, plus UID
when supplied; different versions within the same group may match. **Migration:**
Kind-only owner fixtures must now set APIVersion, for example `apps/v1` for a
Deployment or `v1` for a ConfigMap. Nil and typed-nil objects produce matcher
errors rather than panics.

## Documentation

See the contextual documentation for more detail:

- [Agent Guide](AGENTS.md) — repository navigation, design rules, and the
  agent workflow.
- [Development Guide](docs/development.md) — contribution workflow, testing,
  dependencies, and code quality.
- [Architecture](docs/architecture.md) — package boundaries and runtime
  behavior.

## License

See [LICENSE](LICENSE) for details.
