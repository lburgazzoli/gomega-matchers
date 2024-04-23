package matchers_test

import (
	"testing"

	. "github.com/lburgazzoli/gomega-matchers/pkg/matchers"
	. "github.com/onsi/gomega"
)

func TestJQMatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`{"a":1}`).Should(MatchJQ(`.a == 1`))
	g.Expect(`{"a":1}`).Should(Not(MatchJQ(`.a == 2`)))
	g.Expect(`{"Values":[ "foo" ]}`).Should(MatchJQ(`.Values | if . then any(. == "foo") else false end`))
	g.Expect(`{"Values":[ "foo" ]}`).Should(Not(MatchJQ(`.Values | if . then any(. == "bar") else false end`)))
	g.Expect(`{"Values": null}`).Should(Not(MatchJQ(`.Values | if . then any(. == "foo") else false end`)))
}
