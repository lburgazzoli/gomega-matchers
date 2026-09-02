package k8s_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func TestUsingDelegatesToOperations(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cli := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-config", Namespace: "default"},
		Data:       map[string]string{"key": "value"},
	})
	ops := k8s.Using(cli)

	result, err := ops.Get(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-config", Namespace: "default"},
	})(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveField("Data", HaveKeyWithValue("key", "value")))

	list, err := ops.List(&corev1.ConfigMapList{})(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(list).To(HaveField("Items", HaveLen(1)))
}
