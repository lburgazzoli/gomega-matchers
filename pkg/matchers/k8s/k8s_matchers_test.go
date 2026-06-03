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

	g.Expect(cm).To(SatisfyAll(
		k8s.HasName("test-config"),
		k8s.HasNamespace("default"),
		k8s.HasLabel("app", "frontend"),
		k8s.HasAnnotation("managed-by", "operator"),
		k8s.HasOwnerReference("Module", "example"),
	))
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
