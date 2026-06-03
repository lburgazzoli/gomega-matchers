package jq_test

import (
	"encoding/json"
	"testing"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	. "github.com/onsi/gomega"
)

func TestExtract(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{ "foo": { "a": 1 }}`).Should(
		WithTransform(jq.Extract(`.foo`), WithTransform(json.Marshal,
			jq.Match(`.a == 1`),
		)),
	)

	g.Expect(`{ "status": { "foo": { "bar": "fr", "baz": "fz" } } }`).Should(
		WithTransform(jq.Extract(`.status`),
			And(
				jq.Match(`.foo.bar == "fr"`),
				jq.Match(`.foo.baz == "fz"`),
			),
		),
	)
}

func TestExtractFormattedExpression(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{"status":{"phase":"Running"}}`).Should(
		WithTransform(jq.Extractf(`.status.%s`, "phase"), Equal("Running")),
	)
}

func TestExtractNoResultReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	value, err := jq.Extract(`.missing`)(`{"present":true}`)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(value).Should(BeNil())
}

func TestExtractParseErrorStopsRetrying(t *testing.T) {
	t.Parallel()

	_, err := jq.Extract(`[`)(`{"present":true}`)

	assertStopTrying(t, err)
}
