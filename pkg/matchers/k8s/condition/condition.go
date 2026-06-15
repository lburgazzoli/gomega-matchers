package condition

import (
	"strings"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// HasType matches a condition by its Type field or "type" key.
func HasType(expected any) types.GomegaMatcher {
	return &delegatingFieldMatcher{
		field:    "Type",
		expected: expected,
	}
}

// HasStatus matches a condition by its Status field or "status" key.
func HasStatus(expected any) types.GomegaMatcher {
	return &delegatingFieldMatcher{
		field:    "Status",
		expected: expected,
	}
}

// HasReason matches a condition by its Reason field or "reason" key.
func HasReason(expected any) types.GomegaMatcher {
	return &delegatingFieldMatcher{
		field:    "Reason",
		expected: expected,
	}
}

// HasMessage matches a condition by its Message field or "message" key.
func HasMessage(expected any) types.GomegaMatcher {
	return &delegatingFieldMatcher{
		field:    "Message",
		expected: expected,
	}
}

// Is matches the common case of a condition type plus status pair.
func Is(typeValue any, statusValue any) types.GomegaMatcher {
	return gomega.SatisfyAll(
		HasType(typeValue),
		HasStatus(statusValue),
	)
}

type delegatingFieldMatcher struct {
	field    string
	expected any
}

func (m *delegatingFieldMatcher) Match(actual any) (bool, error) {
	return m.delegateFor(actual).Match(actual)
}

func (m *delegatingFieldMatcher) FailureMessage(actual any) string {
	return m.delegateFor(actual).FailureMessage(actual)
}

func (m *delegatingFieldMatcher) NegatedFailureMessage(actual any) string {
	return m.delegateFor(actual).NegatedFailureMessage(actual)
}

func (m *delegatingFieldMatcher) delegateFor(actual any) types.GomegaMatcher {
	if _, ok := actual.(map[string]any); ok {
		return gomega.HaveKeyWithValue(
			strings.ToLower(m.field[:1])+m.field[1:],
			m.expected,
		)
	}

	return gomega.HaveField(m.field, gomega.Equal(m.expected))
}
