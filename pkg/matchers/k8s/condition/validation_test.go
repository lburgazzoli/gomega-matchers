package condition_test

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s/condition"

	. "github.com/onsi/gomega"
)

func TestValidateTypeAndStatus(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	for _, status := range []metav1.ConditionStatus{
		metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionUnknown,
	} {
		g.Expect(condition.ValidateTypeAndStatus("Ready", status)).To(Succeed())
	}

	g.Expect(condition.ValidateTypeAndStatus("", metav1.ConditionTrue)).To(
		MatchError(ContainSubstring("condition type must not be empty")),
	)

	for _, status := range []metav1.ConditionStatus{"", "true", "invalid"} {
		g.Expect(condition.ValidateTypeAndStatus("Ready", status)).To(
			MatchError(ContainSubstring("invalid condition status")),
		)
	}
}

func TestFindCondition(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	entry := map[string]any{
		"type": "Ready", "status": "True", "observedGeneration": int64(2),
		"reason": "Reconciled", "message": "ready", "lastTransitionTime": "2026-11-01T12:00:00Z",
	}

	entries := []any{
		map[string]any{"type": "Available", "status": "False"},
		entry,
	}

	found, err := condition.Find(entries, "Ready")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).NotTo(BeNil())
	g.Expect(found.Type).To(Equal("Ready"))
	g.Expect(found.Status).To(Equal(metav1.ConditionTrue))
	g.Expect(found.ObservedGeneration).To(Equal(int64(2)))
	g.Expect(found.Reason).To(Equal("Reconciled"))
	g.Expect(found.LastTransitionTime.IsZero()).To(BeFalse())

	found.Message = "changed"

	g.Expect(entry["message"]).To(Equal("ready"))

	for _, values := range [][]any{nil, {}, entries} {
		missing, findErr := condition.Find(values, "Missing")
		g.Expect(findErr).NotTo(HaveOccurred())
		g.Expect(missing).To(BeNil())
	}

	_, err = condition.Find(nil, "")
	g.Expect(err).To(MatchError(ContainSubstring("condition type must not be empty")))
}

func TestFindRejectsMalformedAndDuplicateConditions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		entries []any
	}{
		{name: "duplicate requested type", entries: []any{
			map[string]any{"type": "Ready", "status": "True"},
			map[string]any{"type": "Ready", "status": "False"},
		}},
		{name: "non-object entry", entries: []any{"bad"}},
		{name: "nil entry", entries: []any{nil}},
		{name: "missing type", entries: []any{map[string]any{"status": "True"}}},
		{name: "invalid status", entries: []any{map[string]any{"type": "Ready", "status": "bad"}}},
		{name: "missing status", entries: []any{map[string]any{"type": "Ready"}}},
		{name: "invalid generation", entries: []any{
			map[string]any{"type": "Ready", "status": "True", "observedGeneration": "two"},
		}},
		{name: "invalid timestamp", entries: []any{
			map[string]any{"type": "Ready", "status": "True", "lastTransitionTime": "bad"},
		}},
		{name: "malformed unrelated condition", entries: []any{
			map[string]any{"type": "Ready", "status": "True"},
			map[string]any{"type": "Available", "status": "bad"},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := condition.Find(tc.entries, "Ready")
			NewWithT(t).Expect(err).To(HaveOccurred())
		})
	}
}

func TestFindAllowsDuplicateUnrelatedTypes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	found, err := condition.Find([]any{
		map[string]any{"type": "Available", "status": "True"},
		map[string]any{"type": "Ready", "status": "Unknown"},
		map[string]any{"type": "Available", "status": "False"},
	}, "Ready")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).NotTo(BeNil())
	g.Expect(found.Status).To(Equal(metav1.ConditionUnknown))
}

func TestFieldMatchersDoNotEnforceValidation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(map[string]any{"type": "Custom", "status": "custom"}).To(condition.Is("Custom", "custom"))
	g.Expect(metav1.Condition{Type: "Custom", Status: "custom"}).To(
		condition.Is("Custom", metav1.ConditionStatus("custom")),
	)
}
