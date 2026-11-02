package k8s

import (
	"errors"
	"fmt"
	"slices"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// HasCompletedDeploymentRollout matches an apps/v1 Deployment whose latest
// generation has been observed and whose updated, total and available replica
// counts equal spec.replicas (defaulting to one when omitted). Progress deadline
// failures remain mismatches so Eventually may continue polling. A missing or
// zero observedGeneration does not match, including when scaling to zero.
func HasCompletedDeploymentRollout() types.GomegaMatcher {
	return &deploymentRolloutMatcher{}
}

type deploymentRolloutMatcher struct{}

func (m *deploymentRolloutMatcher) Match(actual any) (bool, error) {
	deployment, err := m.deployment(actual)
	if err != nil {
		return false, err
	}

	want := m.desiredReplicas(deployment)
	status := deployment.Status

	return status.ObservedGeneration > 0 && status.ObservedGeneration >= deployment.Generation &&
		status.UpdatedReplicas == want && status.Replicas == want && status.AvailableReplicas == want, nil
}

func (m *deploymentRolloutMatcher) FailureMessage(actual any) string {
	return m.message(actual, "to have completed Deployment rollout")
}

func (m *deploymentRolloutMatcher) NegatedFailureMessage(actual any) string {
	return m.message(actual, "not to have completed Deployment rollout")
}

func (*deploymentRolloutMatcher) deployment(actual any) (*appsv1.Deployment, error) {
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

func (*deploymentRolloutMatcher) desiredReplicas(deployment *appsv1.Deployment) int32 {
	if deployment.Spec.Replicas == nil {
		return 1
	}

	return *deployment.Spec.Replicas
}

func (m *deploymentRolloutMatcher) message(actual any, verb string) string {
	deployment, err := m.deployment(actual)
	if err != nil {
		return err.Error()
	}

	status := deployment.Status
	detail := fmt.Sprintf("desired=%d, updated=%d, total=%d, available=%d, generation=%d, observedGeneration=%d",
		m.desiredReplicas(deployment),
		status.UpdatedReplicas,
		status.Replicas,
		status.AvailableReplicas,
		deployment.Generation,
		status.ObservedGeneration,
	)

	deadline := slices.IndexFunc(status.Conditions, func(item appsv1.DeploymentCondition) bool {
		return item.Type == appsv1.DeploymentProgressing && item.Reason == "ProgressDeadlineExceeded"
	})

	if deadline >= 0 {
		progress := status.Conditions[deadline]
		detail += fmt.Sprintf(", %s: %s", progress.Reason, progress.Message)
	}

	return format.Message(detail, verb)
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

func (m *quantityMatcher) Match(actual any) (bool, error) {
	if m.parseErr != nil {
		return false, fmt.Errorf("invalid expected quantity: %w", m.parseErr)
	}

	quantity, err := m.value(actual)
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

func (*quantityMatcher) value(actual any) (resource.Quantity, error) {
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

func (m *quantityMatcher) message(actual any, verb string) string {
	display := actual

	if quantity, err := m.value(actual); err == nil {
		copyOfQuantity := quantity.DeepCopy()
		display = copyOfQuantity.String()
	}

	return format.Message(display, verb+m.comparator, m.expectedText)
}
