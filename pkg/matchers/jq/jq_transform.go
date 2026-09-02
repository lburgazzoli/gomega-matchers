package jq

import (
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega"
)

type resultRunner func(query *gojq.Query, data any) (any, error)

func (j *Instance) newTransform(expression string, runner resultRunner) func(in any) (any, error) {
	query, parseErr := parseQuery(expression)

	return func(in any) (any, error) {
		if parseErr != nil {
			return nil, terminalJQError(parseErr)
		}

		data, err := j.Convert(in)
		if err != nil {
			return nil, err
		}

		return runner(query, data)
	}
}

// Extract returns a transform function that extracts a value from input using a JQ expression.
// The returned function can be used with Gomega's WithTransform matcher combinator.
//
// Only the first value produced by the expression is returned; multi-value
// expressions silently discard subsequent results. Returns nil when the
// expression produces no results.
//
// Example:
//
//	WithTransform(jq.Extract(`.status`), Equal("ready"))
func Extract(expression string) func(in any) (any, error) {
	return defaultInstance.Extract(expression)
}

// Extract returns a transform function using the instance's converter registry.
func (j *Instance) Extract(expression string) func(in any) (any, error) {
	return j.newTransform(expression, func(q *gojq.Query, data any) (any, error) {
		result, err := evalOptional(q, data, expression)
		if errors.Is(err, ErrMultipleResults) {
			return nil, gomega.StopTrying(err.Error()).Wrap(err)
		}

		return result, err
	})
}

// ExtractAll returns a transform function that extracts all values produced by
// a JQ expression. It returns an empty slice when the expression produces no
// results.
func ExtractAll(expression string) func(in any) (any, error) {
	return defaultInstance.ExtractAll(expression)
}

// ExtractAll returns a transform function using the instance's converter registry.
func (j *Instance) ExtractAll(expression string) func(in any) (any, error) {
	return j.newTransform(expression, func(q *gojq.Query, data any) (any, error) {
		return evalAll(q, data)
	})
}

// Extractf returns a transform function from a formatted JQ expression.
func Extractf(expressionFormat string, args ...any) func(in any) (any, error) {
	return defaultInstance.Extractf(expressionFormat, args...)
}

// Extractf returns a transform function from a formatted JQ expression using
// the instance's converter registry.
func (j *Instance) Extractf(expressionFormat string, args ...any) func(in any) (any, error) {
	return j.Extract(fmt.Sprintf(expressionFormat, args...))
}

// ExtractAllf returns a transform function from a formatted JQ expression.
func ExtractAllf(expressionFormat string, args ...any) func(in any) (any, error) {
	return defaultInstance.ExtractAllf(expressionFormat, args...)
}

// ExtractAllf returns a transform function from a formatted JQ expression
// using the instance's converter registry.
func (j *Instance) ExtractAllf(expressionFormat string, args ...any) func(in any) (any, error) {
	return j.ExtractAll(fmt.Sprintf(expressionFormat, args...))
}

// Transform returns a function that applies a JQ transformation expression to the input
// and returns the full transformed result.
// Unlike Extract which returns nil when no result is produced, Transform returns an error,
// since a transformation that yields nothing indicates a problem with the expression.
//
// Only the first value produced by the expression is returned; multi-value
// expressions silently discard subsequent results.
//
// Example:
//
//	result, err := jq.Transform(`. + {"new_field": "value"}`)(input)
func Transform(expression string) func(in any) (any, error) {
	return defaultInstance.Transform(expression)
}

// Transform returns a transform function using the instance's converter registry.
func (j *Instance) Transform(expression string) func(in any) (any, error) {
	return j.newTransform(expression, func(q *gojq.Query, data any) (any, error) {
		return evalRequired(q, data, expression)
	})
}

// TransformAll returns a function that applies a JQ expression and returns all
// transformed results. It returns an error when the expression produces no
// results.
func TransformAll(expression string) func(in any) (any, error) {
	return defaultInstance.TransformAll(expression)
}

// TransformAll returns a transform function using the instance's converter registry.
func (j *Instance) TransformAll(expression string) func(in any) (any, error) {
	return j.newTransform(expression, func(q *gojq.Query, data any) (any, error) {
		results, err := evalAll(q, data)
		if err != nil {
			return nil, err
		}

		if len(results) == 0 {
			return nil, fmt.Errorf("jq transform %q produced no result", expression)
		}

		return results, nil
	})
}

// Transformf returns a transform function from a formatted JQ expression.
func Transformf(expressionFormat string, args ...any) func(in any) (any, error) {
	return defaultInstance.Transformf(expressionFormat, args...)
}

// Transformf returns a transform function from a formatted JQ expression
// using the instance's converter registry.
func (j *Instance) Transformf(expressionFormat string, args ...any) func(in any) (any, error) {
	return j.Transform(fmt.Sprintf(expressionFormat, args...))
}

// TransformAllf returns a transform function from a formatted JQ expression.
func TransformAllf(expressionFormat string, args ...any) func(in any) (any, error) {
	return defaultInstance.TransformAllf(expressionFormat, args...)
}

// TransformAllf returns a transform function from a formatted JQ expression
// using the instance's converter registry.
func (j *Instance) TransformAllf(expressionFormat string, args ...any) func(in any) (any, error) {
	return j.TransformAll(fmt.Sprintf(expressionFormat, args...))
}
