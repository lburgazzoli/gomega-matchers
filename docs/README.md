# Documentation

Use the document that matches your context:

- [README](../README.md) — installation, user-facing API, and usage examples.
- [Development Guide](development.md) — local workflow, testing, dependency
  changes, formatting, and contribution conventions.
- [Architecture](architecture.md) — package boundaries, data flow, conversion
  behavior, and Kubernetes retry semantics.
- [AGENTS.md](../AGENTS.md) — concise instructions for coding agents.

The package-level Go documentation is also useful when working on a specific
API:

- [`pkg/matchers/k8s` package docs](../pkg/matchers/k8s/doc.go)
- [`pkg/matchers/jq` source](../pkg/matchers/jq/)
- [`pkg/matchers/k8s/condition` source](../pkg/matchers/k8s/condition/)

Keep user-facing examples in the root README. Put contributor workflow and
implementation context in the documents in this directory.
