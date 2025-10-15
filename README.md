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

## Documentation

For development guidelines, architecture details, and contributing information, see [docs/development.md](docs/development.md).

## License

See [LICENSE](LICENSE) for details.
