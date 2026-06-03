package jq_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	. "github.com/onsi/gomega"
)

type stopTryingError interface {
	error
	IsStopTrying() bool
}

func assertStopTrying(t *testing.T, err error) {
	t.Helper()

	var stopErr stopTryingError
	if !errors.As(err, &stopErr) {
		t.Fatalf("expected StopTrying error, got %T: %v", err, err)
	}

	if !stopErr.IsStopTrying() {
		t.Fatalf("expected StopTrying error, got %T: %v", err, err)
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{"a":1}`).Should(
		jq.Match(`.a == 1`),
	)

	g.Expect(`{"a":1}`).Should(
		Not(
			jq.Match(`.a == 2`),
		),
	)

	g.Expect(`{"Values":[ "foo" ]}`).Should(
		jq.Match(`.Values | if . then any(. == "foo") else false end`),
	)

	g.Expect(`{"Values":[ "foo" ]}`).Should(
		Not(
			jq.Match(`.Values | if . then any(. == "bar") else false end`),
		),
	)

	g.Expect(`{"Values": null}`).Should(
		Not(
			jq.Match(`.Values | if . then any(. == "foo") else false end`),
		),
	)

	g.Expect(`{ "status": { "foo": { "bar": "fr", "baz": "fb" } } }`).Should(
		And(
			jq.Match(`.status.foo.bar == "fr"`),
			jq.Match(`.status.foo.baz == "fb"`),
		),
	)

	g.Expect(`{"value":"100%"}`).Should(
		jq.Match(`.value == "100%"`),
	)
}

func TestMatcherFormattedExpression(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{"status":{"phase":"Running"}}`).Should(
		jq.Matchf(`.status.phase == "%s"`, "Running"),
	)
}

func TestMatcherParseErrorStopsRetrying(t *testing.T) {
	t.Parallel()

	_, err := jq.Match(`[`).
		Match(`{"status":{"phase":"Running"}}`)

	assertStopTrying(t, err)
}

func TestMatcherWithType(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(map[string]any{"a": 1}).
		Should(
			WithTransform(json.Marshal, jq.Match(`.a == 1`)),
		)

	g.Expect(
		map[string]any{
			"status": map[string]any{
				"foo": map[string]any{
					"bar": "fr",
					"baz": "fb",
				},
			},
		}).
		Should(
			WithTransform(json.Marshal, And(
				jq.Match(`.status.foo.bar == "fr"`),
				jq.Match(`.status.foo.baz == "fb"`),
			)),
		)

	g.Expect(map[string]any{"a": 1}).
		Should(jq.Match(`.a == 1`))

	g.Expect(
		struct {
			A int `json:"a"`
		}{
			A: 1,
		}).
		Should(
			WithTransform(json.Marshal, jq.Match(`.a == 1`)),
		)
}
