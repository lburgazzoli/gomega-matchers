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

func TestExtract(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(e1).Should(
		WithTransform(yq.Extract(`.foo`), yq.Match(`.a == 1`)),
	)
}
