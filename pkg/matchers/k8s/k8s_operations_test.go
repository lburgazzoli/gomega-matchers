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

func newFakeClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&corev1.Pod{}).
		WithObjects(objects...).
		Build()
}

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

func newUnstructuredPod() *unstructured.Unstructured {
	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name":      "test-pod",
				"namespace": "default",
			},
		},
	}
	obj.SetGroupVersionKind(podGVK)

	return obj
}

// --- Get ---

func TestLookupTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels:    map[string]string{"env": "prod"},
		},
		Data: map[string]string{
			"key1": "value1",
		},
	})

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	g.Eventually(k8s.Lookup(c, cm)).
		WithContext(t.Context()).
		Should(Succeed())
	g.Expect(cm.Data).To(HaveKeyWithValue("key1", "value1"))
	g.Expect(cm.Labels).To(HaveKeyWithValue("env", "prod"))
}

func TestLookupUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	})

	obj := newUnstructuredPod()

	g.Eventually(k8s.Lookup(c, obj)).
		WithContext(t.Context()).
		Should(Succeed())
	g.Expect(obj.GetName()).To(Equal("test-pod"))
	g.Expect(obj.GetNamespace()).To(Equal("default"))
	g.Expect(obj.GetLabels()).To(HaveKeyWithValue("app", "test"))
}

func TestGetTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	})

	g.Eventually(k8s.Get(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(jq.Match(`
		.metadata.name == "test-config" and
		.metadata.namespace == "default" and
		.data.key1 == "value1"
	`))
}

func TestGetTypedWithLabels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels:    map[string]string{"env": "prod"},
		},
		Data: map[string]string{"database": "postgres"},
	})

	g.Eventually(k8s.Get(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(And(
		jq.Match(`.data.database == "postgres"`),
		jq.Match(`.metadata.labels.env == "prod"`),
	))
}

func TestGetUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	})

	obj, err := k8s.Get(c, newUnstructuredPod())(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj.GetName()).To(Equal("test-pod"))
	g.Expect(obj.GetNamespace()).To(Equal("default"))
}

func TestGetUnstructuredWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	})

	g.Eventually(k8s.Get(c, newUnstructuredPod())).
		WithContext(t.Context()).
		Should(And(
			jq.Match(`.status.phase == "Running"`),
			jq.Match(`.metadata.labels.app == "test"`),
		))
}

func TestGetNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	_, err := k8s.Get(c, newUnstructuredPod())(t.Context())
	g.Expect(err).To(HaveOccurred())
}

func TestGetClusterScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	})

	ns := &unstructured.Unstructured{}
	ns.SetGroupVersionKind(namespaceGVK)
	ns.SetName("test-namespace")

	obj, err := k8s.Get(c, ns)(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(obj.GetName()).To(Equal("test-namespace"))
}

// --- Create ---

func TestCreateTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Create(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key1": "created"},
	})).WithContext(t.Context()).Should(jq.Match(`
		.metadata.name == "test-config" and
		.data.key1 == "created"
	`))
}

func TestCreateUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Create(c, newUnstructuredConfigMap(map[string]any{
		"key": "created",
	}))).
		WithContext(t.Context()).
		Should(jq.Match(`.data.key == "created"`))
}

// --- Delete ---

func TestDeleteTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})

	g.Eventually(k8s.Delete(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(Succeed())

	result := &corev1.ConfigMap{}
	err := c.Get(t.Context(), types.NamespacedName{Name: "test-config", Namespace: "default"}, result)
	g.Expect(err).To(HaveOccurred())
}

func TestDeleteUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(newUnstructuredConfigMap(map[string]any{"key": "value"}))

	g.Eventually(k8s.Delete(c, newUnstructuredConfigMap(nil))).
		WithContext(t.Context()).
		Should(Succeed())

	obj := newUnstructuredConfigMap(nil)
	err := c.Get(t.Context(), types.NamespacedName{Name: "test-config", Namespace: "default"}, obj)
	g.Expect(err).To(HaveOccurred())
}

func TestDeleteClusterScoped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	})

	ns := &unstructured.Unstructured{}
	ns.SetGroupVersionKind(namespaceGVK)
	ns.SetName("test-namespace")

	err := k8s.Delete(c, ns)(t.Context())
	g.Expect(err).ToNot(HaveOccurred())

	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(namespaceGVK)
	err = c.Get(t.Context(), types.NamespacedName{Name: "test-namespace"}, obj)
	g.Expect(err).To(HaveOccurred())
}

// --- Update ---

func TestUpdateTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key1": "original"},
	})

	g.Eventually(k8s.Update(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(cm *corev1.ConfigMap) {
		cm.Data["key1"] = "updated"
		cm.Data["key2"] = "new"
	})).WithContext(t.Context()).Should(jq.Match(`
		.data.key1 == "updated" and
		.data.key2 == "new"
	`))
}

func TestUpdateUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(newUnstructuredConfigMap(map[string]any{"key": "old-value"}))

	g.Eventually(k8s.Update(c, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			data := obj.Object["data"].(map[string]any) //nolint:forcetypeassert
			data["key"] = "new-value"
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "new-value"`))
}

func TestUpdateUnstructuredWithJQTransform(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels:    map[string]string{"app": "test"},
		},
		Data: map[string]string{"key": "old-value"},
	})

	transform := jq.Transform(`.data.key = "new-value" | .metadata.labels += {"updated": "true"}`)

	g.Eventually(k8s.Update(c, newUnstructuredConfigMap(nil),
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

// --- StatusUpdate ---

func TestStatusUpdateTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	})

	g.Eventually(k8s.StatusUpdate(c, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}, func(pod *corev1.Pod) {
		pod.Status.Phase = corev1.PodSucceeded
	})).WithContext(t.Context()).Should(jq.Match(`.status.phase == "Succeeded"`))
}

func TestStatusUpdateUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{Phase: corev1.PodPending},
	})

	g.Eventually(k8s.StatusUpdate(c, newUnstructuredPod(),
		func(obj *unstructured.Unstructured) {
			err := unstructured.SetNestedField(obj.Object, "Succeeded", "status", "phase")
			g.Expect(err).ToNot(HaveOccurred())
		},
	)).WithContext(t.Context()).Should(jq.Match(`.status.phase == "Succeeded"`))
}

// --- Upsert ---

func TestUpsertCreatesMissingTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Upsert(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(cm *corev1.ConfigMap) {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		cm.Data["key1"] = "created"
	})).WithContext(t.Context()).Should(jq.Match(`.data.key1 == "created"`))
}

func TestUpsertUpdatesExistingTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{"key1": "original"},
	})

	g.Eventually(k8s.Upsert(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(cm *corev1.ConfigMap) {
		cm.Data["key1"] = "updated"
		cm.Data["key2"] = "new"
	})).WithContext(t.Context()).Should(jq.Match(`
		.data.key1 == "updated" and
		.data.key2 == "new"
	`))
}

func TestUpsertCreatesMissingUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Upsert(c, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			obj.Object["data"] = map[string]any{"key": "created"}
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "created"`))
}

func TestUpsertUpdatesExistingUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(newUnstructuredConfigMap(map[string]any{"key": "old-value"}))

	g.Eventually(k8s.Upsert(c, newUnstructuredConfigMap(nil),
		func(obj *unstructured.Unstructured) {
			data := obj.Object["data"].(map[string]any) //nolint:forcetypeassert
			data["key"] = "new-value"
		},
	)).WithContext(t.Context()).Should(jq.Match(`.data.key == "new-value"`))
}

// --- Absent / NotFound ---

func TestAbsentTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Absent(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "missing-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(BeTrue())
}

func TestAbsentExistingTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})

	g.Eventually(k8s.Absent(c, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(BeFalse())
}

func TestAbsentUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.Absent(c, newUnstructuredPod())).
		WithContext(t.Context()).
		Should(BeTrue())
}

func TestAbsentExistingUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	})

	g.Eventually(k8s.Absent(c, newUnstructuredPod())).
		WithContext(t.Context()).
		Should(BeFalse())
}

// --- List ---

func TestListTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config-1", Namespace: "default"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config-2", Namespace: "default"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "config-3", Namespace: "other"}},
	)

	g.Eventually(k8s.List(c, &corev1.ConfigMapList{}, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(WithTransform(k8s.ListItems(), HaveLen(2)))
}

func TestListTypedIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	g.Eventually(k8s.List(c, &corev1.ConfigMapList{}, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(k8s.IsEmptyList())
}

func TestListUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-3", Namespace: "other"}},
	)

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(podGVK)

	result, err := k8s.List(c, list, client.InNamespace("default"))(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(2))
}

func TestListUnstructuredWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"}},
	)

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(podGVK)

	g.Eventually(k8s.List(c, list, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(jq.Match(`. | length == 2`))
}

func TestListUnstructuredIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient()

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(podGVK)

	g.Eventually(k8s.List(c, list, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(k8s.IsEmptyList())
}

func TestListUnstructuredWithLabelSelector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "frontend"},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "backend"},
			},
		},
	)

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(podGVK)

	result, err := k8s.List(c, list,
		client.InNamespace("default"),
		client.MatchingLabels{"app": "frontend"},
	)(t.Context())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Items).To(HaveLen(1))
	g.Expect(result.Items[0].GetName()).To(Equal("pod-1"))
}
