package condition

import (
	"encoding/json/v2"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateTypeAndStatus checks that conditionType is nonempty and status is
// True, False, or Unknown. It does not validate other condition fields or change
// the behavior of the condition-field matchers.
func ValidateTypeAndStatus(conditionType string, status metav1.ConditionStatus) error {
	if err := validateType(conditionType); err != nil {
		return err
	}

	switch status {
	case metav1.ConditionTrue, metav1.ConditionFalse, metav1.ConditionUnknown:
		return nil
	default:
		return fmt.Errorf("invalid condition status %q", status)
	}
}

// Find parses and validates the raw .status.conditions entries and returns the
// unique condition with conditionType. Entries must be map[string]any values
// convertible to metav1.Condition with a nonempty type and standard status.
// Missing matches return nil without an error. Malformed entries, an empty
// requested type, or duplicate entries of the requested type return errors.
// Other condition types are validated but need not be unique. Input is not
// mutated; callers can change the returned condition independently.
func Find(entries []any, conditionType string) (*metav1.Condition, error) {
	if err := validateType(conditionType); err != nil {
		return nil, err
	}

	var found *metav1.Condition

	for _, entry := range entries {
		parsed, err := parse(entry)
		switch {
		case err != nil:
			return nil, err
		case parsed.Type != conditionType:
			continue
		case found != nil:
			return nil, fmt.Errorf("duplicate condition type %q", conditionType)
		default:
			found = parsed
		}
	}

	return found, nil
}

func validateType(conditionType string) error {
	if conditionType == "" {
		return errors.New("condition type must not be empty")
	}

	return nil
}

func parse(entry any) (*metav1.Condition, error) {
	value, ok := entry.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected condition object, got %T", entry)
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling condition: %w", err)
	}

	var parsed metav1.Condition
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshaling condition: %w", err)
	}

	if err := ValidateTypeAndStatus(parsed.Type, parsed.Status); err != nil {
		return nil, err
	}

	return &parsed, nil
}
