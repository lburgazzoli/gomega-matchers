package yq_test

import (
	"testing"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/yq"

	. "github.com/onsi/gomega"
)

const e1 = `
foo:
  a: 1
`

const e2 = `
status:
  foo:
    bar: fr
    baz: fz
`

func TestExtract(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(e1).Should(
		WithTransform(yq.Extract(`.foo`), yq.Match(`.a == 1`)),
	)

	g.Expect(e2).Should(
		WithTransform(yq.Extract(`.status`),
			And(
				yq.Match(`.foo.bar == "fr"`),
				yq.Match(`.foo.baz == "fz"`),
			),
		),
	)
}
