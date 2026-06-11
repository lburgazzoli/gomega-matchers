# gomega-matchers

Additional matchers for [Gomega](https://onsi.github.io/gomega/), the Go testing assertion framework. This library provides powerful JQ-based matchers for validating JSON structures in tests.

## Installation

```bash
go get github.com/lburgazzoli/gomega-matchers
```

## Features

- **JQ matchers** - Use [jq](https://jqlang.github.io/jq/) expressions to query and validate JSON structures
- **Flexible input types** - Works with JSON strings, byte slices, readers, and Go types (maps, structs)
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
jq.RegisterConverter(func(v any) (any, error) {
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
})

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
All operations are package-level functions that take a `client.Client` directly.
Both typed objects (e.g., `*corev1.ConfigMap`) and unstructured objects work with the same functions.

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

// Query events — use standard Gomega matchers on the result
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
```

## Documentation

For development guidelines, architecture details, and contributing information, see [docs/development.md](docs/development.md).

## License

See [LICENSE](LICENSE) for details.
