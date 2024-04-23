package jq_test

import (
	"testing"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	. "github.com/onsi/gomega"
)

func TestJQMatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{"a":1}`).Should(jq.Match(`.a == 1`))
	g.Expect(`{"a":1}`).Should(Not(jq.Match(`.a == 2`)))
	g.Expect(`{"Values":[ "foo" ]}`).Should(jq.Match(`.Values | if . then any(. == "foo") else false end`))
	g.Expect(`{"Values":[ "foo" ]}`).Should(Not(jq.Match(`.Values | if . then any(. == "bar") else false end`)))
	g.Expect(`{"Values": null}`).Should(Not(jq.Match(`.Values | if . then any(. == "foo") else false end`)))
}
