package yq_test

import (
	"testing"

	"github.com/goccy/go-yaml"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/yq"

	. "github.com/onsi/gomega"
)

func TestMatcher(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(`a: 1`).Should(yq.Match(`.a == 1`))
}

func TestMatcherWithType(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(map[string]any{"a": 1}).Should(
		WithTransform(yaml.Marshal, yq.Match(`.a == 1`)),
	)

	g.Expect(
		struct {
			A int `yaml:"a"`
		}{
			A: 1,
		}).
		Should(
			WithTransform(yaml.Marshal, yq.Match(`.a == 1`)),
		)
}
