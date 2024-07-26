# gomega-matchers

Additional matchers for [gomega](https://onsi.github.io/gomega/)

# JQ support
```go

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
    )),
)

```


# YQ support
```go

in := `
status:
  foo:
    bar: fr
    baz: fz
`

Expect(in).Should(
    WithTransform(yq.Extract(`.status`),
        And(
            yq.Match(`.foo.bar == "fr"`),
            yq.Match(`.foo.baz == "fz"`),
        ),
    )),
)

```