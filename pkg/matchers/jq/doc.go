// Package jq provides Gomega matchers and transforms backed by gojq.
//
// Match evaluates an expression that must produce exactly one boolean result.
// Extract and Transform evaluate expressions that must produce exactly one
// result; ExtractAll and TransformAll are available when multiple results are
// intentional. JSON strings, byte slices, raw messages, maps, and slices are
// supported by the built-in converters.
//
// Package-level functions use a shared, unexported instance. Use New to create
// an isolated Instance with its own converter registry.
package jq
