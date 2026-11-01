package k8s_test

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func partialPod() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", Labels: map[string]string{"app": "backend"}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "example/backend:v2"}}},
	}
}

func TestMatchPartiallyTyped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := partialPod()
	pattern.Namespace = ""
	actual := partialPod()
	actual.Labels["injected"] = "true"
	actual.Spec.Containers[0].Args = []string{"serve"}
	actual.Spec.Containers = append(actual.Spec.Containers, corev1.Container{Name: "sidecar", Image: "proxy:v1"})
	before := actual.DeepCopy()
	for _, expected := range []any{pattern, *pattern} {
		for _, input := range []any{actual, *actual} {
			g.Expect(input).To(k8s.MatchPartially(expected))
		}
	}
	g.Expect(actual).To(Equal(before))
	actual.Spec.Containers[0].Image = "example/backend:v1"
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
	g.Expect(&corev1.ConfigMap{}).NotTo(k8s.MatchPartially(corev1.Pod{}))
}

func TestMatchPartiallyUsesNativeTypedRules(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*corev1.Pod)
		match  bool
	}{
		{name: "empty pattern", mutate: func(*corev1.Pod) {}, match: true},
		{name: "false remains an assertion", mutate: func(p *corev1.Pod) { p.Spec.HostNetwork = true }},
		{name: "zero remains an assertion", mutate: func(p *corev1.Pod) { p.Generation = 1 }},
		{name: "creation timestamp remains an assertion", mutate: func(p *corev1.Pod) {
			p.CreationTimestamp = metav1.Now()
		}},
		{name: "nil pointer is ignored", mutate: func(p *corev1.Pod) {
			p.Spec.TerminationGracePeriodSeconds = new(int64(30))
		}, match: true},
		{name: "empty string is ignored", mutate: func(p *corev1.Pod) { p.Spec.NodeName = "node" }, match: true},
		{name: "empty slice is ignored", mutate: func(p *corev1.Pod) {
			p.Spec.Containers = []corev1.Container{{Name: "app", Image: "image"}}
		}, match: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			pattern := &corev1.Pod{}
			actual := &corev1.Pod{}
			tc.mutate(actual)
			matched, err := k8s.MatchPartially(pattern).Match(actual)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(matched).To(Equal(tc.match))
		})
	}
}

func TestMatchPartiallySlicePositionsAndPointers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := partialPod()
	pattern.Spec.Containers[0].Args = []string{"serve"}
	pattern.Spec.SecurityContext = &corev1.PodSecurityContext{RunAsNonRoot: new(false)}
	actual := pattern.DeepCopy()
	actual.Spec.Containers[0].Args = []string{"serve", "--verbose"}
	g.Expect(actual).To(k8s.MatchPartially(pattern))
	actual.Spec.Containers[0].Args = []string{"--verbose", "serve"}
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
	actual = pattern.DeepCopy()
	actual.Spec.Containers = append([]corev1.Container{{Name: "sidecar"}}, actual.Spec.Containers...)
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
	actual = pattern.DeepCopy()
	actual.Spec.SecurityContext.RunAsNonRoot = new(true)
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
	actual.Spec.SecurityContext = nil
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
}

func TestMatchPartiallySemanticValues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := partialPod()
	pattern.CreationTimestamp = metav1.NewTime(time.Date(2026, time.November, 1, 12, 0, 0, 0, time.UTC))
	pattern.Spec.Containers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m")}
	actual := pattern.DeepCopy()
	actual.CreationTimestamp.Time = actual.CreationTimestamp.In(time.FixedZone("offset", 3600))
	actual.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("0.5")
	g.Expect(actual).To(k8s.MatchPartially(pattern))
	actual.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU] = resource.MustParse("501m")
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
}

func TestMatchPartiallyUnstructuredRules(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{"enabled": false, "count": int64(0)},
	}}
	actual := &unstructured.Unstructured{Object: map[string]any{
		"spec": map[string]any{"enabled": false, "count": int64(0), "extra": true},
	}}
	g.Expect(actual).To(k8s.MatchPartially(*pattern))
	g.Expect(unstructured.SetNestedField(actual.Object, true, "spec", "enabled")).To(Succeed())
	g.Expect(actual).NotTo(k8s.MatchPartially(pattern))
	for _, empty := range []any{"", nil, []any{}, map[string]any{}} {
		p := &unstructured.Unstructured{Object: map[string]any{"value": empty}}
		a := &unstructured.Unstructured{Object: map[string]any{"value": empty}}
		g.Expect(a).To(k8s.MatchPartially(p))
		delete(a.Object, "value")
		g.Expect(a).NotTo(k8s.MatchPartially(p))
	}
	g.Expect(&unstructured.Unstructured{Object: map[string]any{"value": "anything"}}).To(
		k8s.MatchPartially(unstructured.Unstructured{Object: map[string]any{"value": ""}}),
	)
	g.Expect(&unstructured.Unstructured{Object: map[string]any{"value": []any{int64(1)}}}).To(
		k8s.MatchPartially(unstructured.Unstructured{Object: map[string]any{"value": []any{}}}),
	)
	g.Expect(&unstructured.Unstructured{Object: map[string]any{"value": map[string]any{"extra": true}}}).To(
		k8s.MatchPartially(unstructured.Unstructured{Object: map[string]any{"value": map[string]any{}}}),
	)
	g.Expect(&unstructured.Unstructured{Object: map[string]any{"value": "0.5"}}).NotTo(
		k8s.MatchPartially(unstructured.Unstructured{Object: map[string]any{"value": "500m"}}),
	)
}

func TestMatchPartiallyMixedInputs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := partialPod()
	actual := partialPod()
	actual.Labels["extra"] = "allowed"
	g.Expect(asUnstructured(t, actual)).To(k8s.MatchPartially(pattern))
	g.Expect(actual).To(k8s.MatchPartially(asUnstructured(t, pattern)))
	actual.Kind = "ConfigMap"
	g.Expect(asUnstructured(t, actual)).NotTo(k8s.MatchPartially(pattern))
	actual = partialPod()
	actual.APIVersion = "other.example.com/v1"
	g.Expect(asUnstructured(t, actual)).NotTo(k8s.MatchPartially(pattern))
	actual = partialPod()
	bad := asUnstructured(t, actual)
	g.Expect(unstructured.SetNestedField(bad.Object, "invalid", "spec", "hostNetwork")).To(Succeed())
	_, err := k8s.MatchPartially(pattern).Match(bad)
	g.Expect(err).To(HaveOccurred())
	pattern.TypeMeta = metav1.TypeMeta{}
	_, err = k8s.MatchPartially(pattern).Match(asUnstructured(t, actual))
	g.Expect(err).To(MatchError(ContainSubstring("APIVersion and Kind")))
	_, err = k8s.MatchPartially(asUnstructured(t, actual)).Match(pattern)
	g.Expect(err).To(MatchError(ContainSubstring("APIVersion and Kind")))
}

func TestMatchPartiallySnapshotAndErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	pattern := partialPod()
	matcher := k8s.MatchPartially(pattern)
	pattern.Labels["app"] = "changed"
	pattern.Spec.Containers[0].Image = "changed"
	actual := partialPod()
	before := actual.DeepCopy()
	g.Expect(actual).To(matcher)
	g.Expect(actual).To(Equal(before))
	g.Expect(matcher.FailureMessage(actual)).To(ContainSubstring("to partially match"))
	g.Expect(matcher.NegatedFailureMessage(actual)).To(ContainSubstring("not to partially match"))
	for _, invalid := range []any{
		nil, (*corev1.Pod)(nil), (*unstructured.Unstructured)(nil), "invalid", 1, corev1.Container{},
	} {
		_, err := k8s.MatchPartially(invalid).Match(actual)
		g.Expect(err).To(HaveOccurred())
		_, err = matcher.Match(invalid)
		g.Expect(err).To(HaveOccurred())
	}
}

func TestMatchPartiallyPolling(t *testing.T) {
	t.Parallel()
	for _, generic := range []bool{false, true} {
		t.Run(map[bool]string{false: "typed", true: "unstructured"}[generic], func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			scheme := runtime.NewScheme()
			g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())
			object := completeDeployment()
			object.Status.AvailableReplicas = 1
			cli := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(object).WithObjects(object).Build()
			pattern := object.DeepCopy()
			pattern.ResourceVersion = ""
			pattern.Status.AvailableReplicas = 2
			var expected any = pattern
			if generic {
				expected = asUnstructured(t, pattern)
			}
			get := k8s.Get(cli, object)
			polls := 0
			g.Eventually(t.Context(), func(ctx context.Context) (*appsv1.Deployment, error) {
				observed, err := get(ctx)
				if err != nil {
					return nil, err
				}
				polls++
				observed.TypeMeta = object.TypeMeta
				if polls == 1 {
					updated := observed.DeepCopy()
					updated.Status.AvailableReplicas = 2
					g.Expect(cli.Status().Update(ctx, updated)).To(Succeed())
				}

				return observed, nil
			}).WithTimeout(time.Second).Should(k8s.MatchPartially(expected))
			g.Expect(polls).To(Equal(2))
		})
	}
}
