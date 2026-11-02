package condition_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s/condition"

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

				matcher := condition.HasCurrentCondition("Ready", metav1.ConditionTrue)

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

	matcher := condition.HasCurrentCondition("Ready", metav1.ConditionTrue)

	g.Expect(obj).NotTo(matcher)
	g.Expect(matcher.FailureMessage(obj)).To(SatisfyAll(
		ContainSubstring("Ready=True"), ContainSubstring("generation=2"), ContainSubstring("observedGeneration=1"),
	))

	obj.Status.Conditions[0].ObservedGeneration = 2
	g.Expect(obj).To(matcher)

	for _, status := range []metav1.ConditionStatus{metav1.ConditionFalse, metav1.ConditionUnknown} {
		obj.Status.Conditions[0].Status = status
		g.Expect(obj).To(condition.HasCurrentCondition("Ready", status))
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
	g.Expect(obj).NotTo(condition.HasCurrentCondition("Ready", metav1.ConditionTrue))
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

			_, err := condition.HasCurrentCondition("Ready", metav1.ConditionTrue).Match(obj)
			NewWithT(t).Expect(err).To(HaveOccurred())
		})
	}

	for _, matcher := range []types.GomegaMatcher{
		condition.HasCurrentCondition("", metav1.ConditionTrue),
		condition.HasCurrentCondition("Ready", "invalid"),
	} {
		_, err := matcher.Match(currentResource())
		NewWithT(t).Expect(err).To(HaveOccurred())
	}
}

func TestCurrentConditionRejectsInvalidObjects(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	matcher := condition.HasCurrentCondition("Ready", metav1.ConditionTrue)

	for _, actual := range []any{nil, (*conditionResource)(nil), (*unstructured.Unstructured)(nil), "bad"} {
		_, err := matcher.Match(actual)
		g.Expect(err).To(HaveOccurred())
	}
}

func TestCurrentConditionPollsUntilStatusChanges(t *testing.T) {
	t.Parallel()

	for _, generic := range []bool{false, true} {
		t.Run(fmt.Sprintf("unstructured=%t", generic), func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			scheme := runtime.NewScheme()
			scheme.AddKnownTypeWithName(schema.GroupVersionKind{
				Group: "testing.example.com", Version: "v1", Kind: "TestResource",
			}, &conditionResource{})

			resource := currentResource()
			resource.Status.Conditions[0].ObservedGeneration = 1

			var object client.Object = resource
			if generic {
				object = asUnstructured(t, resource)
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
					case *unstructured.Unstructured:
						g.Expect(unstructured.SetNestedSlice(obj.Object, []any{map[string]any{
							"type": "Ready", "status": "True", "observedGeneration": int64(2),
						}}, "status", "conditions")).To(Succeed())
					}

					g.Expect(cli.Status().Update(ctx, updated)).To(Succeed())
				}

				return observed, nil
			}).WithTimeout(time.Second).Should(condition.HasCurrentCondition("Ready", metav1.ConditionTrue))

			g.Expect(polls).To(Equal(2))
		})
	}
}
