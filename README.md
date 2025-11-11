# gomega-matchers

Additional matchers for [Gomega](https://onsi.github.io/gomega/), the Go testing assertion framework. This library provides powerful JQ-based matchers for validating JSON structures in tests.

## Installation

```bash
go get github.com/lburgazzoli/gomega-matchers
```

## Features

- **JQ matchers** - Use [jq](https://jqlang.github.io/jq/) expressions to query and validate JSON structures
- **Flexible input types** - Works with JSON strings, byte slices, readers, and Go types (maps, structs)
- **Kubernetes support** - Native support for `unstructured.Unstructured` objects
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

## Kubernetes Matchers

The library provides helper functions for testing Kubernetes resources with Gomega. Two APIs are available: a **typed API** that works with Kubernetes object types, and an **unstructured API** that works with GVK (GroupVersionKind).

### Typed API (Recommended)

The typed API accepts Kubernetes typed objects (e.g., `*corev1.ConfigMap`) and returns unstructured results compatible with JQ matchers:

```go
import (
    . "github.com/onsi/gomega"
    "github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
    "github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create a typed matcher
k := k8s.New(client, scheme)

// Get a resource
obj, err := k.Get(ctx, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})
Expect(err).ToNot(HaveOccurred())
Expect(obj).Should(jq.Match(`.data.key == "value"`))

// List resources
list, err := k.List(ctx, &corev1.ConfigMapList{},
    client.InNamespace("default"),
    client.MatchingLabels{"app": "myapp"},
)
Expect(err).ToNot(HaveOccurred())
Expect(list).Should(jq.Match(`. | length > 0`))

// Update a resource (Komega-style)
obj, err := k.Update(ctx, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
}, func(cm client.Object) {
    configMap := cm.(*corev1.ConfigMap)
    configMap.Data["key"] = "new-value"
})
Expect(err).ToNot(HaveOccurred())
Expect(obj).Should(jq.Match(`.data.key == "new-value"`))

// Delete a resource
err := k.Delete(ctx, &corev1.ConfigMap{
    ObjectMeta: metav1.ObjectMeta{
        Name:      "my-config",
        Namespace: "default",
    },
})
Expect(err).ToNot(HaveOccurred())
```

### Unstructured API

The unstructured API works with GVK and returns functions compatible with `Eventually()`:

```go
import (
    "k8s.io/apimachinery/pkg/runtime/schema"
)

// Create an unstructured matcher
k := k8s.NewUnstructured(client)

podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}

// Get with Eventually
Eventually(k.Get(podGVK, k8s.Named("my-pod").InNamespace("default"))).
    WithContext(ctx).
    Should(jq.Match(`.status.phase == "Running"`))

// List resources
Eventually(k.List(podGVK, client.InNamespace("default"))).
    WithContext(ctx).
    Should(jq.Match(`. | length > 0`))

// Delete a resource
err := k.Delete(podGVK, k8s.Named("my-pod").InNamespace("default"))(ctx)
Expect(err).ToNot(HaveOccurred())
```

## Documentation

For development guidelines, architecture details, and contributing information, see [docs/development.md](docs/development.md).

## License

See [LICENSE](LICENSE) for details.
