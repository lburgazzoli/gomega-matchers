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
		WithTransform(jq.Extract(`.foo`), WithTransform(json.Marshal, jq.Match(`.a == 1`))),
	)
}
