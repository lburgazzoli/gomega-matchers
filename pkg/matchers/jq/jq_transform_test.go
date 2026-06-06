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

func TestTransform(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.Transform(`. + {"b": 2}`)(map[string]any{"a": 1})

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"a": 1, "b": 2}))
}

func TestTransformModifiesField(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.Transform(`.status = "done"`)(map[string]any{"status": "pending"})

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"status": "done"}))
}

func TestTransformFormattedExpression(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.Transformf(`.data.%s = "%s"`, "key", "new")(
		map[string]any{"data": map[string]any{"key": "old"}},
	)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(
		WithTransform(json.Marshal, jq.Match(`.data.key == "new"`)),
	)
}

func TestTransformNoResultReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := jq.Transform(`empty`)(`{"a":1}`)

	g.Expect(err).Should(HaveOccurred())
	g.Expect(err.Error()).Should(ContainSubstring("produced no result"))
}

func TestTransformParseErrorStopsRetrying(t *testing.T) {
	t.Parallel()

	_, err := jq.Transform(`[`)(`{"a":1}`)

	assertStopTrying(t, err)
}

func TestTransformWithStringInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.Transform(`. + {"baz": "qux"}`)(`{"foo": "bar"}`)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar", "baz": "qux"}))
}
