package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

// Match creates a Gomega matcher that evaluates a JQ expression against the actual value.
// The expression should return a boolean result.
//
// Example:
//
//	Expect(`{"a":1}`).Should(jq.Match(`.a == 1`))
func Match(expression string) types.GomegaMatcher {
	query, parseErr := parseQuery(expression)

	return &jqMatcher{
		expression: expression,
		query:      query,
		parseErr:   parseErr,
	}
}

// Matchf creates a Gomega matcher from a formatted JQ expression.
//
// Example:
//
//	Expect(data).Should(jq.Matchf(`.status.phase == "%s"`, "Running"))
func Matchf(expressionFormat string, args ...any) types.GomegaMatcher {
	return Match(fmt.Sprintf(expressionFormat, args...))
}

var _ types.GomegaMatcher = &jqMatcher{}

type jqMatcher struct {
	expression string
	query      *gojq.Query
	parseErr   error
}

func (matcher *jqMatcher) Match(actual any) (bool, error) {
	if matcher.parseErr != nil {
		return false, terminalJQError(matcher.parseErr)
	}

	data, err := Convert(actual)
	if err != nil {
		return false, err
	}

	v, err := EvalFirst(matcher.query, data)
	if err != nil {
		return false, gomega.StopTrying(err.Error())
	}

	if match, ok := v.(bool); ok {
		return match, nil
	}

	return false, gomega.StopTrying(fmt.Sprintf("jq expression %q returned %T, expected bool", matcher.expression, v))
}

func (matcher *jqMatcher) FailureMessage(actual any) string {
	a := fmt.Sprintf("%v", actual)

	return format.Message(a, "to match expression", matcher.expression)
}

func (matcher *jqMatcher) NegatedFailureMessage(actual any) string {
	a := fmt.Sprintf("%v", actual)

	return format.Message(a, "not to match expression", matcher.expression)
}
