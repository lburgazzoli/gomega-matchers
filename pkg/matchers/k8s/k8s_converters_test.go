package k8s_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"

	. "github.com/onsi/gomega"
)

func TestUnstructuredConverter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "test-pod",
			},
		},
	}

	result, err := jq.Convert(obj)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(obj.Object))
}

func TestUnstructuredPtrConverter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "test-pod",
			},
		},
	}

	result, err := jq.Convert(obj)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(obj.Object))
}

func TestUnstructuredPtrConverterNilDoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var obj *unstructured.Unstructured
	var (
		result any
		err    error
	)

	g.Expect(func() {
		result, err = jq.Convert(obj)
	}).ShouldNot(Panic())
	g.Expect(result).To(BeNil())
	g.Expect(err).To(HaveOccurred())
}

func TestUnstructuredListPtrConverter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	list := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{Object: map[string]any{"apiVersion": "v1", "kind": "Pod"}},
			{Object: map[string]any{"apiVersion": "v1", "kind": "Service"}},
		},
	}

	result, err := jq.Convert(list)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal([]any{
		map[string]any{"apiVersion": "v1", "kind": "Pod"},
		map[string]any{"apiVersion": "v1", "kind": "Service"},
	}))
}

func TestUnstructuredListPtrConverterNilDoesNotPanic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var list *unstructured.UnstructuredList
	var (
		result any
		err    error
	)

	g.Expect(func() {
		result, err = jq.Convert(list)
	}).ShouldNot(Panic())
	g.Expect(result).To(BeNil())
	g.Expect(err).To(HaveOccurred())
}

func TestClientObjectConverter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	result, err := jq.Convert(cm)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(BeAssignableToTypeOf(map[string]any{}))
}

func TestClientObjectConverterWithJQMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cm",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	g.Expect(cm).Should(jq.Match(`.metadata.name == "test-cm"`))
	g.Expect(cm).Should(jq.Match(`.metadata.namespace == "default"`))
	g.Expect(cm).Should(jq.Match(`.data.key == "value"`))
}

func TestClientObjectConverterDoesNotAffectUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": "test",
			},
		},
	}

	result, err := jq.Convert(obj)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(obj.Object))
}

func TestConvertNormalizesUnstructuredInt64(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":       "test",
				"generation": int64(7),
			},
			"data": map[string]any{
				"count": int64(42),
			},
		},
	}

	g.Expect(obj).Should(jq.Match(`.metadata.generation == 7`))
	g.Expect(obj).Should(jq.Match(`.data.count == 42`))
	g.Expect(obj).Should(jq.Match(`.data.count > 10`))
}
