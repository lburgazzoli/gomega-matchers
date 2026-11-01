package k8s_test

import (
	"testing"

	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func TestOwnerReferenceIdentity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		version    string
		kind       string
		ownerName  string
		uid        k8stypes.UID
		controller *bool
		match      bool
		controlled bool
	}{
		{name: "same owner", version: "apps/v1", kind: "Deployment", ownerName: "app",
			uid: "uid", controller: new(true), match: true, controlled: true},
		{name: "other version", version: "apps/v1beta1", kind: "Deployment", ownerName: "app",
			uid: "uid", controller: new(true), match: true, controlled: true},
		{name: "other group", version: "other.example.com/v1", kind: "Deployment", ownerName: "app",
			uid: "uid", controller: new(true)},
		{name: "other kind", version: "apps/v1", kind: "StatefulSet", ownerName: "app", uid: "uid", controller: new(true)},
		{name: "other name", version: "apps/v1", kind: "Deployment", ownerName: "other", uid: "uid", controller: new(true)},
		{name: "other UID", version: "apps/v1", kind: "Deployment", ownerName: "app",
			uid: "recreated", controller: new(true)},
		{name: "missing version", kind: "Deployment", ownerName: "app", uid: "uid", controller: new(true)},
		{name: "malformed version", version: "apps/v1/extra", kind: "Deployment", ownerName: "app",
			uid: "uid", controller: new(true)},
		{name: "no controller", version: "apps/v1", kind: "Deployment", ownerName: "app", uid: "uid", match: true},
		{name: "false controller", version: "apps/v1", kind: "Deployment", ownerName: "app",
			uid: "uid", controller: new(false), match: true},
	}
	for _, tc := range cases {
		for _, generic := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/typed", true: "/unstructured"}[generic], func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)
				owner := completeDeployment()
				owner.UID = "uid"
				child := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{
					APIVersion: tc.version, Kind: tc.kind, Name: tc.ownerName, UID: tc.uid, Controller: tc.controller,
				}}}}
				var actual client.Object = child
				var expected client.Object = owner
				if generic {
					actual = asUnstructured(t, child)
					expected = asUnstructured(t, owner)
				}
				before := actual.DeepCopyObject()
				matched, err := k8s.HasOwnerReference(expected).Match(actual)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(matched).To(Equal(tc.match))
				matched, err = k8s.IsControlledBy(expected).Match(actual)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(matched).To(Equal(tc.controlled))
				g.Expect(actual).To(Equal(before))
			})
		}
	}
}

func TestCoreOwnerReferenceAndOptionalUID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	owner := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"}, ObjectMeta: metav1.ObjectMeta{Name: "config"},
	}
	child := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{
		APIVersion: "v1", Kind: "ConfigMap", Name: "config", UID: "any-uid",
	}}}}
	g.Expect(child).To(k8s.HasOwnerReference(owner))
	child.OwnerReferences[0].APIVersion = "other.example.com/v1"
	g.Expect(child).NotTo(k8s.HasOwnerReference(owner))
}

func TestInvalidOwnersReturnErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	for _, owner := range []client.Object{
		nil, (*appsv1.Deployment)(nil), (*unstructured.Unstructured)(nil),
		&appsv1.Deployment{TypeMeta: metav1.TypeMeta{Kind: "Deployment"}, ObjectMeta: metav1.ObjectMeta{Name: "app"}},
		&appsv1.Deployment{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1"}, ObjectMeta: metav1.ObjectMeta{Name: "app"}},
		&appsv1.Deployment{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"}},
		&unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1/extra", "kind": "Deployment", "metadata": map[string]any{"name": "app"},
		}},
	} {
		for _, matcher := range []types.GomegaMatcher{k8s.HasOwnerReference(owner), k8s.IsControlledBy(owner)} {
			_, err := matcher.Match(&corev1.Pod{})
			g.Expect(err).To(HaveOccurred())
		}
	}
}

func TestMetadataMatchersRejectNilObjects(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	for _, matcher := range []types.GomegaMatcher{
		k8s.HasName("app"), k8s.HasNamespace("default"), k8s.HasLabel("app", "app"),
		k8s.HasAnnotation("app", "app"), k8s.HasFinalizer("example.com/finalizer"), k8s.IsDeleting(),
		k8s.MatchesGroupVersion(schema.GroupVersion{Group: "apps", Version: "v1"}),
		k8s.MatchesGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment")),
		k8s.HasOwnerReference(completeDeployment()), k8s.IsControlledBy(completeDeployment()),
	} {
		for _, actual := range []any{nil, (*corev1.Pod)(nil), (*unstructured.Unstructured)(nil), "invalid"} {
			_, err := matcher.Match(actual)
			g.Expect(err).To(HaveOccurred())
		}
	}
}
