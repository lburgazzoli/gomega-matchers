package condition

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// HasCurrentCondition matches a condition's type and status only when its
// observedGeneration equals the object's positive metadata.generation.
// It accepts typed and unstructured client.Object values. Missing or stale
// conditions are mismatches; malformed or duplicate conditions are errors.
// Resources without condition-level observedGeneration cannot match.
func HasCurrentCondition(conditionType string, status metav1.ConditionStatus) types.GomegaMatcher {
	return &currentConditionMatcher{conditionType: conditionType, status: status}
}

type currentConditionMatcher struct {
	conditionType string
	status        metav1.ConditionStatus
}

func (m *currentConditionMatcher) Match(actual any) (bool, error) {
	generation, observed, err := m.observe(actual)
	if err != nil {
		return false, err
	}

	return generation > 0 && observed != nil && observed.Status == m.status &&
		observed.ObservedGeneration == generation, nil
}

func (m *currentConditionMatcher) FailureMessage(actual any) string {
	return m.message(actual, "to have current condition")
}

func (m *currentConditionMatcher) NegatedFailureMessage(actual any) string {
	return m.message(actual, "not to have current condition")
}

func (m *currentConditionMatcher) observe(actual any) (int64, *metav1.Condition, error) {
	if err := ValidateTypeAndStatus(m.conditionType, m.status); err != nil {
		return 0, nil, err
	}

	data, err := m.objectData(actual)
	if err != nil {
		return 0, nil, err
	}

	generation, _, err := unstructured.NestedInt64(data, "metadata", "generation")
	if err != nil {
		return 0, nil, fmt.Errorf("reading generation: %w", err)
	}

	items, _, err := unstructured.NestedSlice(data, "status", "conditions")
	if err != nil {
		return 0, nil, fmt.Errorf("reading conditions: %w", err)
	}

	found, err := Find(items, m.conditionType)

	return generation, found, err
}

func (m *currentConditionMatcher) message(actual any, verb string) string {
	generation, observed, err := m.observe(actual)

	detail := fmt.Sprintf("generation=%d, condition missing", generation)
	if observed != nil {
		detail = fmt.Sprintf("generation=%d, status=%s, observedGeneration=%d",
			generation, observed.Status, observed.ObservedGeneration)
	}

	if err != nil {
		detail = err.Error()
	}

	return format.Message(detail, verb, fmt.Sprintf("%s=%s at the object's generation", m.conditionType, m.status))
}

func (*currentConditionMatcher) objectData(actual any) (map[string]any, error) {
	obj, ok := actual.(client.Object)
	if !ok {
		return nil, fmt.Errorf("expected client.Object, got %T", actual)
	}

	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, fmt.Errorf("expected non-nil client.Object, got %T", actual)
	}

	if object, isUnstructured := obj.(*unstructured.Unstructured); isUnstructured {
		return object.Object, nil
	}

	return runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
}
