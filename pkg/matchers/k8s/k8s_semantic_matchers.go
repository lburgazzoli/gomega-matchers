package k8s

import (
	"errors"
	"fmt"
	"slices"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	generation, condition, err := m.observe(actual)
	if err != nil {
		return false, err
	}

	return generation > 0 && condition != nil && condition.Status == m.status &&
		condition.ObservedGeneration == generation, nil
}

func (m *currentConditionMatcher) FailureMessage(actual any) string {
	return m.message(actual, "to have current condition")
}

func (m *currentConditionMatcher) NegatedFailureMessage(actual any) string {
	return m.message(actual, "not to have current condition")
}

func (m *currentConditionMatcher) observe(actual any) (int64, *metav1.Condition, error) {
	switch {
	case m.conditionType == "":
		return 0, nil, errors.New("condition type must not be empty")
	case !validConditionStatus(m.status):
		return 0, nil, fmt.Errorf("invalid condition status %q", m.status)
	}

	obj, err := asObject(actual)
	if err != nil {
		return 0, nil, err
	}
	data, err := toUnstructuredMap(obj)
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

	found, err := findCondition(items, m.conditionType)

	return generation, found, err
}

func validConditionStatus(status metav1.ConditionStatus) bool {
	switch status {
	case metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionUnknown:
		return true
	default:
		return false
	}
}

func findCondition(items []any, conditionType string) (*metav1.Condition, error) {
	var found *metav1.Condition
	for _, item := range items {
		condition, err := parseCondition(item)
		if err != nil {
			return nil, err
		}
		if condition.Type != conditionType {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("duplicate condition type %q", conditionType)
		}
		found = condition
	}

	return found, nil
}

func parseCondition(item any) (*metav1.Condition, error) {
	value, ok := item.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected condition object, got %T", item)
	}
	condition, err := convertValue[metav1.Condition](value, "condition")
	switch {
	case err != nil:
		return nil, err
	case condition.Type == "":
		return nil, errors.New("condition type must not be empty")
	case !validConditionStatus(condition.Status):
		return nil, fmt.Errorf("invalid status %q for condition %q", condition.Status, condition.Type)
	}

	return &condition, nil
}

func (m *currentConditionMatcher) message(actual any, verb string) string {
	generation, condition, err := m.observe(actual)
	detail := fmt.Sprintf("generation=%d, condition missing", generation)
	if condition != nil {
		detail = fmt.Sprintf("generation=%d, status=%s, observedGeneration=%d",
			generation, condition.Status, condition.ObservedGeneration)
	}
	if err != nil {
		detail = err.Error()
	}

	return format.Message(detail, verb, fmt.Sprintf("%s=%s at the object's generation", m.conditionType, m.status))
}

// HasCompletedDeploymentRollout matches an apps/v1 Deployment whose latest
// generation has been observed and whose updated, total and available replica
// counts equal spec.replicas (defaulting to one when omitted). Progress deadline
// failures remain mismatches so Eventually may continue polling. A missing or
// zero observedGeneration does not match, including when scaling to zero.
func HasCompletedDeploymentRollout() types.GomegaMatcher {
	return &deploymentRolloutMatcher{}
}

type deploymentRolloutMatcher struct{}

func deploymentForRollout(actual any) (*appsv1.Deployment, error) {
	if _, err := asObject(actual); err != nil {
		return nil, err
	}
	switch obj := actual.(type) {
	case *appsv1.Deployment:
		return obj, nil
	case *unstructured.Unstructured:
		if obj.GroupVersionKind() != appsv1.SchemeGroupVersion.WithKind("Deployment") {
			return nil, fmt.Errorf("expected apps/v1 Deployment, got %s", obj.GroupVersionKind())
		}
		deployment, err := convertValue[appsv1.Deployment](obj.Object, "Deployment")

		return &deployment, err
	default:
		return nil, fmt.Errorf("expected *appsv1.Deployment or unstructured apps/v1 Deployment, got %T", actual)
	}
}

func desiredReplicas(deployment *appsv1.Deployment) int32 {
	if deployment.Spec.Replicas == nil {
		return 1
	}

	return *deployment.Spec.Replicas
}

func (*deploymentRolloutMatcher) Match(actual any) (bool, error) {
	deployment, err := deploymentForRollout(actual)
	if err != nil {
		return false, err
	}
	want := desiredReplicas(deployment)
	status := deployment.Status

	return status.ObservedGeneration > 0 && status.ObservedGeneration >= deployment.Generation &&
		status.UpdatedReplicas == want && status.Replicas == want && status.AvailableReplicas == want, nil
}

func deploymentRolloutMessage(actual any, verb string) string {
	deployment, err := deploymentForRollout(actual)
	if err != nil {
		return err.Error()
	}
	status := deployment.Status
	detail := fmt.Sprintf("desired=%d, updated=%d, total=%d, available=%d, generation=%d, observedGeneration=%d",
		desiredReplicas(deployment), status.UpdatedReplicas, status.Replicas, status.AvailableReplicas,
		deployment.Generation, status.ObservedGeneration)
	deadline := slices.IndexFunc(status.Conditions, func(condition appsv1.DeploymentCondition) bool {
		return condition.Type == appsv1.DeploymentProgressing && condition.Reason == "ProgressDeadlineExceeded"
	})
	if deadline >= 0 {
		condition := status.Conditions[deadline]
		detail += fmt.Sprintf(", %s: %s", condition.Reason, condition.Message)
	}

	return format.Message(detail, verb)
}

func (*deploymentRolloutMatcher) FailureMessage(actual any) string {
	return deploymentRolloutMessage(actual, "to have completed Deployment rollout")
}

func (*deploymentRolloutMatcher) NegatedFailureMessage(actual any) string {
	return deploymentRolloutMessage(actual, "not to have completed Deployment rollout")
}

// CompareQuantity compares resource.Quantity values, non-nil quantity pointers,
// or quantity strings using exact Kubernetes quantity arithmetic. Supported
// comparators are ==, !=, <, <=, > and >=. Invalid arguments are matcher errors.
func CompareQuantity(comparator string, expected string) types.GomegaMatcher {
	quantity, err := resource.ParseQuantity(expected)

	return &quantityMatcher{comparator: comparator, expected: quantity, expectedText: expected, parseErr: err}
}

type quantityMatcher struct {
	comparator   string
	expected     resource.Quantity
	expectedText string
	parseErr     error
}

func quantityValue(actual any) (resource.Quantity, error) {
	switch value := actual.(type) {
	case resource.Quantity:
		return value, nil
	case *resource.Quantity:
		if value == nil {
			return resource.Quantity{}, errors.New("expected non-nil quantity pointer")
		}

		return *value, nil
	case string:
		return resource.ParseQuantity(value)
	default:
		return resource.Quantity{}, fmt.Errorf(
			"expected quantity, non-nil quantity pointer or quantity string, got %T", actual)
	}
}

func (m *quantityMatcher) Match(actual any) (bool, error) {
	if m.parseErr != nil {
		return false, fmt.Errorf("invalid expected quantity: %w", m.parseErr)
	}
	quantity, err := quantityValue(actual)
	if err != nil {
		return false, err
	}
	comparison := quantity.Cmp(m.expected)
	switch m.comparator {
	case "==":
		return comparison == 0, nil
	case "!=":
		return comparison != 0, nil
	case "<":
		return comparison < 0, nil
	case "<=":
		return comparison <= 0, nil
	case ">":
		return comparison > 0, nil
	case ">=":
		return comparison >= 0, nil
	default:
		return false, fmt.Errorf("unsupported quantity comparator %q", m.comparator)
	}
}

func (m *quantityMatcher) FailureMessage(actual any) string {
	return m.message(actual, "to compare as a quantity ")
}

func (m *quantityMatcher) NegatedFailureMessage(actual any) string {
	return m.message(actual, "not to compare as a quantity ")
}

func (m *quantityMatcher) message(actual any, verb string) string {
	display := actual
	if quantity, err := quantityValue(actual); err == nil {
		copyOfQuantity := quantity.DeepCopy()
		display = copyOfQuantity.String()
	}

	return format.Message(display, verb+m.comparator, m.expectedText)
}
