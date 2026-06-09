package k8s_test

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func newTypedResourcesWithObjects(objects ...client.Object) *k8s.Resources {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()

	return k8s.NewResources(c, scheme)
}

func TestTypedGet(t *testing.T) {
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
			"key1": "value1",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Get(&corev1.ConfigMap{
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

func TestTypedGetWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels: map[string]string{
				"env": "prod",
			},
		},
		Data: map[string]string{
			"database": "postgres",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Get(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(jq.Match(`.data.database == "postgres"`))

	g.Eventually(k.Get(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(jq.Match(`.metadata.labels.env == "prod"`))
}

func TestTypedAbsent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Absent(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "missing-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(BeTrue())
}

func TestTypedAbsentExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Absent(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(BeFalse())
}

func TestTypedList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-1",
			Namespace: "default",
		},
	}

	cm2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-2",
			Namespace: "default",
		},
	}

	cm3 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-3",
			Namespace: "other",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm1, cm2, cm3).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.List(&corev1.ConfigMapList{}, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(jq.Match(`. | length == 2`))
}

func TestTypedListWithJQMatcher(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "frontend",
			},
		},
	}

	cm2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "backend",
			},
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm1, cm2).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.List(&corev1.ConfigMapList{}, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(jq.Match(`. | length == 2`))
}

func TestTypedListIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	k := newTypedResourcesWithObjects()

	g.Eventually(k.List(&corev1.ConfigMapList{}, client.InNamespace("default"))).
		WithContext(t.Context()).
		Should(k8s.IsEmptyList())
}

func TestTypedDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Delete(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(Succeed())

	result := &corev1.ConfigMap{}
	err := c.Get(t.Context(), types.NamespacedName{Name: "test-config", Namespace: "default"}, result)
	g.Expect(err).To(HaveOccurred())
}

func TestGenericCreate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k8s.Create(k, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "created",
		},
	})).WithContext(t.Context()).Should(jq.Match(`
		.metadata.name == "test-config" and
		.metadata.namespace == "default" and
		.data.key1 == "created"
	`))
}

func TestGenericDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k8s.Delete(k, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	})).WithContext(t.Context()).Should(Succeed())

	result := &corev1.ConfigMap{}
	err := c.Get(t.Context(), types.NamespacedName{Name: "test-config", Namespace: "default"}, result)
	g.Expect(err).To(HaveOccurred())
}

func TestTypedUpdate(t *testing.T) {
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
			"key1": "original",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Update(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(o client.Object) {
		configMap, ok := o.(*corev1.ConfigMap)
		g.Expect(ok).To(BeTrue())
		configMap.Data["key1"] = "updated"
		configMap.Data["key2"] = "new"
	})).WithContext(t.Context()).Should(jq.Match(`
		.data.key1 == "updated" and
		.data.key2 == "new"
	`))
}

func TestGenericUpsertCreatesMissingObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k8s.Upsert(k, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(cm *corev1.ConfigMap) {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		cm.Data["key1"] = "created"
	})).WithContext(t.Context()).Should(jq.Match(`
		.data.key1 == "created"
	`))
}

func TestGenericUpdateWithTypeInference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "original",
		},
	}

	k := newTypedResourcesWithObjects(cm)

	g.Eventually(k8s.Update(k, &corev1.ConfigMap{
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

func TestTypedStatusUpdate(t *testing.T) {
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
			Phase: corev1.PodPending,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&corev1.Pod{}).
		WithObjects(pod).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.StatusUpdate(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}, func(o client.Object) {
		currentPod, ok := o.(*corev1.Pod)
		g.Expect(ok).To(BeTrue())
		currentPod.Status.Phase = corev1.PodSucceeded
	})).WithContext(t.Context()).Should(jq.Match(`.status.phase == "Succeeded"`))
}

func TestGenericStatusUpdateWithTypeInference(t *testing.T) {
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
			Phase: corev1.PodPending,
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&corev1.Pod{}).
		WithObjects(pod).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k8s.StatusUpdate(k, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}, func(pod *corev1.Pod) {
		pod.Status.Phase = corev1.PodSucceeded
	})).WithContext(t.Context()).Should(jq.Match(`.status.phase == "Succeeded"`))
}

func TestGenericUpsertUpdatesExistingObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "original",
		},
	}

	k := newTypedResourcesWithObjects(cm)

	g.Eventually(k8s.Upsert(k, &corev1.ConfigMap{
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

func TestTypedUpdateWithJQMatcher(t *testing.T) {
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
			"status": "pending",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cm).
		Build()

	k := k8s.NewResources(c, scheme)

	g.Eventually(k.Update(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
	}, func(o client.Object) {
		configMap, ok := o.(*corev1.ConfigMap)
		g.Expect(ok).To(BeTrue())
		configMap.Data["status"] = "completed"
	})).WithContext(t.Context()).Should(jq.Match(`.data.status == "completed"`))
}
