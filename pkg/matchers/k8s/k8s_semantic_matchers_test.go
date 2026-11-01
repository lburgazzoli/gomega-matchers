package k8s_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

type conditionResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Status conditionResourceStatus `json:"status,omitzero"`
}

type conditionResourceStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func (r *conditionResource) DeepCopyObject() runtime.Object {
	result := *r
	result.ObjectMeta = *r.DeepCopy()
	result.Status.Conditions = slices.Clone(r.Status.Conditions)

	return &result
}

func currentResource() *conditionResource {
	return &conditionResource{
		TypeMeta:   metav1.TypeMeta{APIVersion: "testing.example.com/v1", Kind: "TestResource"},
		ObjectMeta: metav1.ObjectMeta{Name: "example", Namespace: "default", Generation: 2},
		Status: conditionResourceStatus{Conditions: []metav1.Condition{{
			Type: "Ready", Status: metav1.ConditionTrue, ObservedGeneration: 2,
		}}},
	}
}

func asUnstructured(t *testing.T, object client.Object) *unstructured.Unstructured {
	t.Helper()
	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	return &unstructured.Unstructured{Object: data}
}

func TestHasCurrentCondition(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*conditionResource)
		match  bool
	}{
		{name: "current", mutate: func(*conditionResource) {}, match: true},
		{name: "stale", mutate: func(r *conditionResource) { r.Status.Conditions[0].ObservedGeneration = 1 }},
		{name: "future", mutate: func(r *conditionResource) { r.Status.Conditions[0].ObservedGeneration = 3 }},
		{name: "unobserved", mutate: func(r *conditionResource) { r.Status.Conditions[0].ObservedGeneration = 0 }},
		{name: "no generation", mutate: func(r *conditionResource) { r.Generation = 0 }},
		{name: "wrong status", mutate: func(r *conditionResource) { r.Status.Conditions[0].Status = metav1.ConditionFalse }},
		{name: "wrong type", mutate: func(r *conditionResource) { r.Status.Conditions[0].Type = "Available" }},
		{name: "no conditions", mutate: func(r *conditionResource) { r.Status.Conditions = nil }},
	}
	for _, tc := range cases {
		for _, generic := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unstructured=%t", tc.name, generic), func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)
				obj := currentResource()
				tc.mutate(obj)
				var actual client.Object = obj
				if generic {
					actual = asUnstructured(t, obj)
				}
				before := actual.DeepCopyObject()
				matcher := k8s.HasCurrentCondition("Ready", metav1.ConditionTrue)
				matched, err := matcher.Match(actual)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(matched).To(Equal(tc.match))
				g.Expect(matcher.FailureMessage(actual)).To(ContainSubstring("generation="))
				g.Expect(matcher.NegatedFailureMessage(actual)).To(ContainSubstring("not to have current condition"))
				g.Expect(actual).To(Equal(before))
			})
		}
	}
}

func TestCurrentConditionDiagnosticsAndReuse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	obj := currentResource()
	obj.Status.Conditions[0].ObservedGeneration = 1
	matcher := k8s.HasCurrentCondition("Ready", metav1.ConditionTrue)
	g.Expect(obj).NotTo(matcher)
	g.Expect(matcher.FailureMessage(obj)).To(SatisfyAll(
		ContainSubstring("Ready=True"), ContainSubstring("generation=2"), ContainSubstring("observedGeneration=1"),
	))
	obj.Status.Conditions[0].ObservedGeneration = 2
	g.Expect(obj).To(matcher)
	for _, status := range []metav1.ConditionStatus{metav1.ConditionFalse, metav1.ConditionUnknown} {
		obj.Status.Conditions[0].Status = status
		g.Expect(obj).To(k8s.HasCurrentCondition("Ready", status))
	}
	obj.Status.Conditions = append(obj.Status.Conditions, obj.Status.Conditions[0])
	_, err := matcher.Match(obj)
	g.Expect(err).To(MatchError(ContainSubstring("duplicate condition")))
}

func TestCurrentConditionDoesNotUseTopLevelGeneration(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	obj := asUnstructured(t, currentResource())
	g.Expect(unstructured.SetNestedSlice(obj.Object, []any{
		map[string]any{"type": "Ready", "status": "True"},
	}, "status", "conditions")).To(Succeed())
	g.Expect(unstructured.SetNestedField(obj.Object, int64(2), "status", "observedGeneration")).To(Succeed())
	g.Expect(obj).NotTo(k8s.HasCurrentCondition("Ready", metav1.ConditionTrue))
}

func TestCurrentConditionErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*unstructured.Unstructured)
	}{
		{name: "duplicate", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": []any{
				map[string]any{"type": "Ready", "status": "True"},
				map[string]any{"type": "Ready", "status": "True"},
			}}
		}},
		{name: "invalid generation", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["metadata"] = map[string]any{"generation": "two"}
		}},
		{name: "invalid collection", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": "bad"}
		}},
		{name: "invalid entry", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": []any{"bad"}}
		}},
		{name: "invalid status", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": []any{map[string]any{"type": "Ready", "status": "bad"}}}
		}},
		{name: "missing type", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": []any{map[string]any{"status": "True"}}}
		}},
		{name: "invalid observed generation", mutate: func(obj *unstructured.Unstructured) {
			obj.Object["status"] = map[string]any{"conditions": []any{
				map[string]any{"type": "Ready", "status": "True", "observedGeneration": "two"},
			}}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			obj := asUnstructured(t, currentResource())
			tc.mutate(obj)
			_, err := k8s.HasCurrentCondition("Ready", metav1.ConditionTrue).Match(obj)
			NewWithT(t).Expect(err).To(HaveOccurred())
		})
	}
	for _, matcher := range []types.GomegaMatcher{
		k8s.HasCurrentCondition("", metav1.ConditionTrue),
		k8s.HasCurrentCondition("Ready", "invalid"),
	} {
		_, err := matcher.Match(currentResource())
		NewWithT(t).Expect(err).To(HaveOccurred())
	}
}

func completeDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", Generation: 2},
		Spec:       appsv1.DeploymentSpec{Replicas: new(int32(2))},
		Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: 2, Replicas: 2, AvailableReplicas: 2},
	}
}

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

func TestSemanticMatchersRejectInvalidObjects(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	for _, matcher := range []types.GomegaMatcher{
		k8s.HasCurrentCondition("Ready", metav1.ConditionTrue), k8s.HasCompletedDeploymentRollout(),
	} {
		for _, actual := range []any{nil, (*appsv1.Deployment)(nil), (*unstructured.Unstructured)(nil), "bad"} {
			_, err := matcher.Match(actual)
			g.Expect(err).To(HaveOccurred())
		}
	}
}

func TestSemanticMatchersPollUntilStatusChanges(t *testing.T) {
	t.Parallel()
	for _, generic := range []bool{false, true} {
		for _, kind := range []string{"condition", "deployment"} {
			t.Run(fmt.Sprintf("%s/unstructured=%t", kind, generic), func(t *testing.T) {
				t.Parallel()
				g := NewWithT(t)
				scheme := runtime.NewScheme()
				g.Expect(appsv1.AddToScheme(scheme)).To(Succeed())
				scheme.AddKnownTypeWithName(schema.GroupVersionKind{
					Group: "testing.example.com", Version: "v1", Kind: "TestResource",
				}, &conditionResource{})
				var object client.Object
				var matcher types.GomegaMatcher
				if kind == "condition" {
					obj := currentResource()
					obj.Status.Conditions[0].ObservedGeneration = 1
					object = obj
					matcher = k8s.HasCurrentCondition("Ready", metav1.ConditionTrue)
				} else {
					obj := completeDeployment()
					obj.Status.UpdatedReplicas = 1
					object = obj
					matcher = k8s.HasCompletedDeploymentRollout()
				}
				if generic {
					object = asUnstructured(t, object)
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
						case *conditionResource:
							obj.Status.Conditions[0].ObservedGeneration = 2
						case *appsv1.Deployment:
							obj.Status.UpdatedReplicas = 2
						case *unstructured.Unstructured:
							if kind == "condition" {
								g.Expect(unstructured.SetNestedSlice(obj.Object, []any{map[string]any{
									"type": "Ready", "status": "True", "observedGeneration": int64(2),
								}}, "status", "conditions")).To(Succeed())
							} else {
								g.Expect(unstructured.SetNestedField(obj.Object, int64(2), "status", "updatedReplicas")).To(Succeed())
							}
						}
						g.Expect(cli.Status().Update(ctx, updated)).To(Succeed())
					}

					return observed, nil
				}).Should(matcher)
				g.Expect(polls).To(Equal(2))
			})
		}
	}
}
