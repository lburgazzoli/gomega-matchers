package k8s_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func TestCompletedDeploymentRollout(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*appsv1.Deployment)
		match  bool
	}{
		{name: "complete", mutate: func(*appsv1.Deployment) {}, match: true},
		{name: "updated behind", mutate: func(d *appsv1.Deployment) { d.Status.UpdatedReplicas = 1 }},
		{name: "old replicas", mutate: func(d *appsv1.Deployment) { d.Status.Replicas = 3 }},
		{name: "total behind", mutate: func(d *appsv1.Deployment) { d.Status.Replicas = 1 }},
		{name: "available behind", mutate: func(d *appsv1.Deployment) { d.Status.AvailableReplicas = 1 }},
		{name: "stale generation", mutate: func(d *appsv1.Deployment) { d.Status.ObservedGeneration = 1 }},
		{name: "no status", mutate: func(d *appsv1.Deployment) { d.Status = appsv1.DeploymentStatus{} }},
		{name: "default replicas", mutate: func(d *appsv1.Deployment) {
			d.Spec.Replicas = nil
			d.Status = appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: 1, Replicas: 1, AvailableReplicas: 1}
		}, match: true},
		{name: "zero replicas", mutate: func(d *appsv1.Deployment) {
			d.Spec.Replicas = new(int32(0))
			d.Status = appsv1.DeploymentStatus{ObservedGeneration: 2}
		}, match: true},
		{name: "unobserved zero replicas", mutate: func(d *appsv1.Deployment) {
			d.Generation = 0
			d.Spec.Replicas = new(int32(0))
			d.Status = appsv1.DeploymentStatus{}
		}},
	}

	for _, tc := range cases {
		for _, generic := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unstructured=%t", tc.name, generic), func(t *testing.T) {
				t.Parallel()

				g := NewWithT(t)

				obj := completeDeployment()
				tc.mutate(obj)

				var actual client.Object = obj
				if generic {
					actual = asUnstructured(t, obj)
				} else {
					obj.TypeMeta = metav1.TypeMeta{}
				}

				before := actual.DeepCopyObject()

				matcher := k8s.HasCompletedDeploymentRollout()

				matched, err := matcher.Match(actual)

				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(matched).To(Equal(tc.match))
				g.Expect(matcher.FailureMessage(actual)).To(ContainSubstring("desired="))
				g.Expect(matcher.NegatedFailureMessage(actual)).To(ContainSubstring("not to have completed"))
				g.Expect(actual).To(Equal(before))
			})
		}
	}
}

func TestRolloutErrorsAndDiagnostics(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	obj := completeDeployment()
	obj.Status.UpdatedReplicas = 1
	obj.Status.Conditions = []appsv1.DeploymentCondition{{
		Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse,
		Reason: "ProgressDeadlineExceeded", Message: "pods cannot start",
	}}
	matcher := k8s.HasCompletedDeploymentRollout()

	matched, err := matcher.Match(obj)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(matched).To(BeFalse())
	g.Expect(matcher.FailureMessage(obj)).To(SatisfyAll(
		ContainSubstring("desired=2"), ContainSubstring("updated=1"),
		ContainSubstring("ProgressDeadlineExceeded"), ContainSubstring("pods cannot start"),
	))

	obj.Status.UpdatedReplicas = 2
	g.Expect(obj).To(matcher)

	bad := asUnstructured(t, obj)
	g.Expect(unstructured.SetNestedField(bad.Object, "two", "status", "replicas")).To(Succeed())

	for _, actual := range []any{bad, &corev1.Pod{}, asUnstructured(t, &corev1.Pod{})} {
		_, matchErr := matcher.Match(actual)
		g.Expect(matchErr).To(HaveOccurred())
	}
}

func TestCompareQuantity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		operator string
		actual   string
		expected string
		match    bool
	}{
		{"==", "500m", "0.5", true}, {"==", "1Gi", "1024Mi", true}, {"==", "1G", "1Gi", false},
		{"!=", "500m", "1", true}, {"!=", "500m", "0.5", false},
		{"<", "499m", "500m", true}, {"<", "500m", "500m", false},
		{"<=", "500m", "500m", true}, {"<=", "501m", "500m", false},
		{">", "501m", "500m", true}, {">", "500m", "500m", false},
		{">=", "500m", "500m", true}, {">=", "499m", "500m", false},
		{"==", "0", "0m", true}, {"<", "-1m", "0", true},
		{">", "9007199254740993", "9007199254740992", true},
	}

	for _, tc := range cases {
		t.Run(tc.actual+tc.operator+tc.expected, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			quantity := resource.MustParse(tc.actual)
			before := quantity.DeepCopy()

			matcher := k8s.CompareQuantity(tc.operator, tc.expected)

			for _, actual := range []any{tc.actual, quantity, &quantity} {
				matched, err := matcher.Match(actual)

				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(matched).To(Equal(tc.match))
				g.Expect(matcher.FailureMessage(actual)).To(ContainSubstring(tc.expected))
				g.Expect(matcher.NegatedFailureMessage(actual)).To(ContainSubstring("not to compare"))
			}

			g.Expect(quantity).To(Equal(before))
		})
	}
}

func TestCompareQuantityErrorsAndComposition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	for _, actual := range []any{nil, (*resource.Quantity)(nil), 500, "invalid"} {
		_, err := k8s.CompareQuantity("==", "500m").Match(actual)
		g.Expect(err).To(HaveOccurred())
	}

	for _, matcher := range []types.GomegaMatcher{k8s.CompareQuantity("=", "1"), k8s.CompareQuantity("==", "bad")} {
		_, err := matcher.Match("1")
		g.Expect(err).To(HaveOccurred())
	}

	container := corev1.Container{Resources: corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("0.5")},
	}}
	g.Expect(container).To(WithTransform(k8s.ResourceRequests(), HaveKeyWithValue(
		corev1.ResourceCPU, k8s.CompareQuantity(">=", "500m"),
	)))
}

func TestDeploymentRolloutRejectsInvalidObjects(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	matcher := k8s.HasCompletedDeploymentRollout()

	for _, actual := range []any{nil, (*appsv1.Deployment)(nil), (*unstructured.Unstructured)(nil), "bad"} {
		_, err := matcher.Match(actual)
		g.Expect(err).To(HaveOccurred())
	}
}

func TestDeploymentRolloutPollsUntilStatusChanges(t *testing.T) {
	t.Parallel()

	for _, generic := range []bool{false, true} {
		t.Run(fmt.Sprintf("unstructured=%t", generic), func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			scheme := runtime.NewScheme()
			g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())

			deployment := completeDeployment()
			deployment.Status.UpdatedReplicas = 1

			var object client.Object = deployment
			if generic {
				object = asUnstructured(t, deployment)
			}

			cli := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(object).WithObjects(object).Build()

			get := k8s.Get(cli, object)
			polls := 0

			g.Eventually(t.Context(), func(ctx context.Context) (client.Object, error) {
				observed, err := get(ctx)
				if err != nil {
					return nil, err
				}

				polls++
				if polls == 1 {
					updated, ok := observed.DeepCopyObject().(client.Object)
					g.Expect(ok).To(BeTrue())

					switch obj := updated.(type) {
					case *appsv1.Deployment:
						obj.Status.UpdatedReplicas = 2
					case *unstructured.Unstructured:
						g.Expect(unstructured.SetNestedField(obj.Object, int64(2), "status", "updatedReplicas")).To(Succeed())
					}

					g.Expect(cli.Status().Update(ctx, updated)).To(Succeed())
				}

				return observed, nil
			}).WithTimeout(time.Second).Should(k8s.HasCompletedDeploymentRollout())

			g.Expect(polls).To(Equal(2))
		})
	}
}
