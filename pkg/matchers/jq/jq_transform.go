package jq

import "fmt"

// Extract returns a transform function that extracts a value from input using a JQ expression.
// The returned function can be used with Gomega's WithTransform matcher combinator.
//
// Example:
//
//	WithTransform(jq.Extract(`.status`), Equal("ready"))
func Extract(expression string) func(in any) (any, error) {
	query, parseErr := parseQuery(expression)

	return func(in any) (any, error) {
		if parseErr != nil {
			return nil, terminalJQError(parseErr)
		}

		data, err := Convert(in)
		if err != nil {
			return nil, err
		}

		return runFirstResult(query, data, nil)
	}
}

// Extractf returns a transform function from a formatted JQ expression.
func Extractf(expressionFormat string, args ...any) func(in any) (any, error) {
	return Extract(fmt.Sprintf(expressionFormat, args...))
}

// Transform returns a function that applies a JQ transformation expression to the input
// and returns the full transformed result.
// Unlike Extract which returns nil when no result is produced, Transform returns an error,
// since a transformation that yields nothing indicates a problem with the expression.
//
// Example:
//
//	result, err := jq.Transform(`. + {"new_field": "value"}`)(input)
func Transform(expression string) func(in any) (any, error) {
	query, parseErr := parseQuery(expression)

	return func(in any) (any, error) {
		if parseErr != nil {
			return nil, terminalJQError(parseErr)
		}

		data, err := Convert(in)
		if err != nil {
			return nil, err
		}

		return runRequiredResult(query, data, expression)
	}
}

// Transformf returns a transform function from a formatted JQ expression.
func Transformf(expressionFormat string, args ...any) func(in any) (any, error) {
	return Transform(fmt.Sprintf(expressionFormat, args...))
}
