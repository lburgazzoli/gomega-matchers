package condition_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s/condition"

	. "github.com/onsi/gomega"
)

func TestHasTypeMatchesMapBackedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := map[string]any{
		"type":    "Ready",
		"status":  "True",
		"reason":  "AllGood",
		"message": "resource is ready",
	}

	g.Expect(cond).To(condition.HasType("Ready"))
}

func TestHasStatusMatchesMapBackedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := map[string]any{
		"type":   "Ready",
		"status": "True",
	}

	g.Expect(cond).To(condition.HasStatus("True"))
}

func TestHasStatusMatchesTypedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
	}

	g.Expect(cond).To(condition.HasStatus(metav1.ConditionTrue))
}

func TestHasReasonMatchesMapBackedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := map[string]any{
		"type":   "Available",
		"status": "True",
		"reason": "MinimumReplicasAvailable",
	}

	g.Expect(cond).To(condition.HasReason("MinimumReplicasAvailable"))
}

func TestHasReasonMatchesTypedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := appsv1.DeploymentCondition{
		Type:   appsv1.DeploymentAvailable,
		Status: corev1.ConditionTrue,
		Reason: "MinimumReplicasAvailable",
	}

	g.Expect(cond).To(condition.HasReason("MinimumReplicasAvailable"))
}

func TestHasMessageMatchesMapBackedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := map[string]any{
		"type":    "Ready",
		"status":  "True",
		"message": "resource is ready",
	}

	g.Expect(cond).To(condition.HasMessage("resource is ready"))
}

func TestIsMatchesTypeAndStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := map[string]any{
		"type":   "Ready",
		"status": "True",
		"reason": "AllGood",
	}

	g.Expect(cond).To(condition.Is("Ready", "True"))
}

func TestIsMatchesTypedCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cond := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "AllGood",
	}

	g.Expect(cond).To(condition.Is("Ready", metav1.ConditionTrue))
}

func TestHasTypeReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := condition.HasType("Ready").Match(42)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not a struct"))
}
