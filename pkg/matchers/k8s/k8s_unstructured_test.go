package k8s_test

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func newUnstructuredConfigMap(data map[string]any) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-config",
				"namespace": "default",
			},
			"data": data,
		},
	}
	obj.SetGroupVersionKind(configMapGVK)

	return obj
}

func TestGet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	k := k8s.NewUnstructuredResources(c)

	obj, err := k.Get(podGVK, k8s.Named("test-pod").InNamespace("default"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj).ToNot(BeNil())
	g.Expect(obj.GetName()).To(Equal("test-pod"))
	g.Expect(obj.GetNamespace()).To(Equal("default"))
}

func TestGetWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.Get(podGVK, k8s.Named(pod.Name).InNamespace(pod.Namespace))).
		WithContext(t.Context()).
		Should(jq.Match(`.status.phase == "Running"`))

	g.Eventually(k.Get(podGVK, k8s.Named(pod.Name).InNamespace(pod.Namespace))).
		WithContext(t.Context()).
		Should(jq.Match(`.metadata.labels.app == "test"`))
}

func TestGetNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewUnstructuredResources(c)

	_, err := k.Get(podGVK, k8s.Named("nonexistent").InNamespace("default"))(t.Context())
	g.Expect(err).To(HaveOccurred())
}

func TestAbsent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.Absent(podGVK, k8s.Named("nonexistent").InNamespace("default"))).
		WithContext(t.Context()).
		Should(BeTrue())
}

func TestAbsentExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.Absent(podGVK, k8s.Named("test-pod").InNamespace("default"))).
		WithContext(t.Context()).
		Should(BeFalse())
}

func TestGetClusterScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	k := k8s.NewUnstructuredResources(c)

	obj, err := k.Get(namespaceGVK, k8s.Named("test-namespace"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj).ToNot(BeNil())
	g.Expect(obj.GetName()).To(Equal("test-namespace"))
}

func TestGetWithNamespacedNamed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	k := k8s.NewUnstructuredResources(c)

	obj, err := k.Get(podGVK, k8s.NamespacedNamed("default", "test-pod"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj).ToNot(BeNil())
	g.Expect(obj.GetName()).To(Equal("test-pod"))
}

func TestList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-3",
			Namespace: "other",
			Labels: map[string]string{
				"app": "other",
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod1, pod2, pod3).
		Build()

	k := k8s.NewUnstructuredResources(c)

	list, err := k.List(podGVK, client.InNamespace("default"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(list).ToNot(BeNil())
	g.Expect(list.Items).To(HaveLen(2))
}

func TestListWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "default",
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod1, pod2).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.List(podGVK, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(jq.Match(`. | length == 2`))

	g.Eventually(k.List(podGVK, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(jq.Match(`.[0].metadata.name == "pod-1"`))
}

func TestListIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.List(podGVK, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(k8s.IsEmptyList())
}

func TestListWithLabelSelector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "frontend",
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "backend",
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod1, pod2).
		Build()

	k := k8s.NewUnstructuredResources(c)

	list, err := k.List(
		podGVK,
		client.InNamespace("default"),
		client.MatchingLabels{"app": "frontend"},
	)(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(list.Items).To(HaveLen(1))
	g.Expect(list.Items[0].GetName()).To(Equal("pod-1"))
}

func TestDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		Build()

	k := k8s.NewUnstructuredResources(c)

	err := k.Delete(podGVK, k8s.Named("test-pod").InNamespace("default"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(podGVK)
	err = c.Get(t.Context(), types.NamespacedName{Name: "test-pod", Namespace: "default"}, obj)
	g.Expect(err).To(HaveOccurred())
}

func TestCreateUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k8s.CreateUnstructured(k, newUnstructuredConfigMap(map[string]any{
		"key": "created",
	}))).
		WithContext(t.Context()).
		Should(jq.Match(`.data.key == "created"`))
}

func TestDeleteUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(newUnstructuredConfigMap(map[string]any{
			"key": "value",
		})).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k8s.DeleteUnstructured(k, newUnstructuredConfigMap(nil))).
		WithContext(t.Context()).
		Should(Succeed())

	obj := newUnstructuredConfigMap(nil)
	err := c.Get(t.Context(), types.NamespacedName{Name: "test-config", Namespace: "default"}, obj)
	g.Expect(err).To(HaveOccurred())
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key": "old-value",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.Update(configMapGVK, k8s.Named("test-config").InNamespace("default"),
		func(obj *unstructured.Unstructured) {
			data := obj.Object["data"].(map[string]any) //nolint:forcetypeassert
			data["key"] = "new-value"
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "new-value"`))
}

func TestUpdateUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(newUnstructuredConfigMap(map[string]any{
			"key": "old-value",
		})).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k8s.UpdateUnstructured(k, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			data := obj.Object["data"].(map[string]any) //nolint:forcetypeassert
			data["key"] = "new-value"
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "new-value"`))
}

func TestUpdateWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k.Update(configMapGVK, k8s.Named("test-config").InNamespace("default"),
		func(obj *unstructured.Unstructured) {
			labels := obj.GetLabels()
			labels["updated"] = "true"
			obj.SetLabels(labels)
		},
	)).WithContext(t.Context()).Should(jq.Match(`.metadata.labels.updated == "true"`))
}

func TestUpdateWithJQTransform(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Data: map[string]string{
			"key": "old-value",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewUnstructuredResources(c)

	transform := jq.Transform(`.data.key = "new-value" | .metadata.labels += {"updated": "true"}`)

	g.Eventually(k.Update(configMapGVK, k8s.Named("test-config").InNamespace("default"),
		func(obj *unstructured.Unstructured) {
			result, err := transform(obj.Object)
			g.Expect(err).ShouldNot(HaveOccurred())

			obj.Object = result.(map[string]any) //nolint:forcetypeassert
		},
	)).WithContext(t.Context()).Should(And(
		jq.Match(`.data.key == "new-value"`),
		jq.Match(`.metadata.labels.updated == "true"`),
		jq.Match(`.metadata.labels.app == "test"`),
	))
}

func TestUpsertUnstructuredCreatesMissingObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k8s.UpsertUnstructured(k, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			obj.Object["data"] = map[string]any{"key": "created"}
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "created"`))
}

func TestUpsertUnstructuredUpdatesExistingObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(newUnstructuredConfigMap(map[string]any{
			"key": "old-value",
		})).
		Build()

	k := k8s.NewUnstructuredResources(c)

	g.Eventually(k8s.UpsertUnstructured(k, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			data := obj.Object["data"].(map[string]any) //nolint:forcetypeassert
			data["key"] = "new-value"
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "new-value"`))
}

func TestDeleteClusterScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ns).
		Build()

	k := k8s.NewUnstructuredResources(c)

	err := k.Delete(namespaceGVK, k8s.Named("test-namespace"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(namespaceGVK)
	err = c.Get(t.Context(), types.NamespacedName{Name: "test-namespace"}, obj)
	g.Expect(err).To(HaveOccurred())
}
