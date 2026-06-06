package k8s_test

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func TestObjectMetadataMatchers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels: map[string]string{
				"app": "frontend",
			},
			Annotations: map[string]string{
				"managed-by": "operator",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Module",
					Name: "example",
				},
			},
		},
	}

	owner := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "Module"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "example",
		},
	}

	g.Expect(cm).To(SatisfyAll(
		k8s.HasName("test-config"),
		k8s.HasNamespace("default"),
		k8s.HasLabel("app", "frontend"),
		k8s.HasAnnotation("managed-by", "operator"),
		k8s.HasOwnerReference(owner),
	))
}

func TestIsControlledBy(t *testing.T) {
	t.Parallel()

	boolPtr := func(v bool) *bool { return &v }

	owner := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-deploy",
		},
	}

	t.Run("matches controller owner reference", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "my-deploy", Controller: boolPtr(true)},
				},
			},
		}

		g.Expect(child).To(k8s.IsControlledBy(owner))
	})

	t.Run("rejects when controller is nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "my-deploy"},
				},
			},
		}

		g.Expect(child).ToNot(k8s.IsControlledBy(owner))
	})

	t.Run("rejects when controller is false", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "my-deploy", Controller: boolPtr(false)},
				},
			},
		}

		g.Expect(child).ToNot(k8s.IsControlledBy(owner))
	})

	t.Run("rejects when kind does not match", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "StatefulSet", Name: "my-deploy", Controller: boolPtr(true)},
				},
			},
		}

		g.Expect(child).ToNot(k8s.IsControlledBy(owner))
	})

	t.Run("matches UID when set on owner", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ownerWithUID := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-deploy",
				UID:  "abc-123",
			},
		}

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "my-deploy", UID: "abc-123", Controller: boolPtr(true)},
				},
			},
		}

		g.Expect(child).To(k8s.IsControlledBy(ownerWithUID))
	})

	t.Run("rejects when UID does not match", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		ownerWithUID := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{Kind: "Deployment"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-deploy",
				UID:  "abc-123",
			},
		}

		child := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "Deployment", Name: "my-deploy", UID: "different", Controller: boolPtr(true)},
				},
			},
		}

		g.Expect(child).ToNot(k8s.IsControlledBy(ownerWithUID))
	})
}

func TestOwnerReferenceWithEmptyKind(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ownerNoKind := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	child := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "ConfigMap", Name: "test"},
			},
		},
	}

	_, err := k8s.HasOwnerReference(ownerNoKind).Match(child)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty Kind"))

	_, err = k8s.IsControlledBy(ownerNoKind).Match(child)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty Kind"))
}

func TestGroupVersionMatchers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",
		},
	}

	g.Expect(obj).To(k8s.MatchesGroupVersion(schema.GroupVersion{Version: "v1"}))
	g.Expect(obj).To(k8s.MatchesGroupVersionKind(schema.GroupVersionKind{
		Version: "v1",
		Kind:    "ConfigMap",
	}))
}

func TestObjectMatchersWorkWithEventuallyGet(t *testing.T) {
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
			Annotations: map[string]string{
				"team": "platform",
			},
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
	})).WithContext(t.Context()).Should(SatisfyAll(
		k8s.HasName("test-config"),
		k8s.HasNamespace("default"),
		k8s.HasLabel("env", "prod"),
		k8s.HasAnnotation("team", "platform"),
		k8s.MatchesGroupVersion(schema.GroupVersion{Version: "v1"}),
		k8s.MatchesGroupVersionKind(schema.GroupVersionKind{
			Version: "v1",
			Kind:    "ConfigMap",
		}),
	))
}
