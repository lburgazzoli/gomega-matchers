package k8s_test

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func TestDataExtractsConfigMapData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"foo": "bar",
		},
	}

	g.Expect(cm).To(WithTransform(k8s.Data(), Equal(map[string]string{
		"foo": "bar",
	})))
}

func TestDataExtractsSecretData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"token": []byte("secret"),
		},
	}

	g.Expect(secret).To(WithTransform(k8s.Data(), Equal(map[string][]byte{
		"token": []byte("secret"),
	})))
}

func TestDataExtractsUnstructuredData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-config",
				"namespace": "default",
			},
			"data": map[string]any{
				"foo": "bar",
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Data(), Equal(map[string]any{
		"foo": "bar",
	})))
}

func TestDataReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Data()(42)

	g.Expect(err).To(MatchError("expected *corev1.ConfigMap, *corev1.Secret, or *unstructured.Unstructured, got int"))
}

func TestFinalizersExtractsTypedObjectFinalizers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"example.com/finalizer"},
		},
	}

	g.Expect(cm).To(WithTransform(k8s.Finalizers(), Equal([]string{"example.com/finalizer"})))
}

func TestFinalizersExtractsUnstructuredObjectFinalizers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": "test-config",
				"finalizers": []any{
					"example.com/finalizer",
				},
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Finalizers(), Equal([]string{"example.com/finalizer"})))
}

func TestFinalizersReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Finalizers()(42)

	g.Expect(err).To(MatchError("expected client.Object, got int"))
}

func TestListItemsExtractsTypedListItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	list := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				Data: map[string]string{
					"key": "value",
				},
			},
		},
	}

	g.Expect(list).To(WithTransform(k8s.ListItems(), HaveLen(1)))
}

func TestListItemsExtractsUnstructuredListItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	list := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]any{
					"metadata": map[string]any{
						"name": "test",
					},
				},
			},
		},
	}

	g.Expect(list).To(WithTransform(k8s.ListItems(), HaveLen(1)))
}

func TestListItemsReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.ListItems()(42)

	g.Expect(err).To(MatchError("expected runtime.Object list, got int"))
}

func TestListItemsReturnsErrorForNilUnstructuredList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var list *unstructured.UnstructuredList

	_, err := k8s.ListItems()(list)

	g.Expect(err).To(MatchError("expected runtime.Object list, got *unstructured.UnstructuredList"))
}
