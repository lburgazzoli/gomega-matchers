package k8s

import (
	"context"
	"testing"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	. "github.com/onsi/gomega"
)

func TestGoneTreatsNoMatchAsGone(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isGone, err := gone(func(context.Context) (struct{}, error) {
		return struct{}{}, &apimeta.NoKindMatchError{
			GroupKind: schema.GroupKind{
				Group: "apps",
				Kind:  "Deployment",
			},
			SearchedVersions: []string{"v1"},
		}
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isGone).To(BeTrue())
}
