package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
)

// Extract returns a transform function that extracts a value from input using a JQ expression.
// The returned function can be used with Gomega's WithTransform matcher combinator.
//
// Example:
//
//	WithTransform(jq.Extract(`.status`), Equal("ready"))
func Extract(expression string) func(in any) (any, error) {
	var query *gojq.Query

	return func(in any) (any, error) {
		if query == nil {
			q, err := parseQuery(expression)
			if err != nil {
				return nil, err
			}

			query = q
		}

		data, err := Convert(in)
		if err != nil {
			return nil, err
		}

		return Run(query, data)
	}
}

// Extractf returns a transform function from a formatted JQ expression.
func Extractf(expressionFormat string, args ...any) func(in any) (any, error) {
	return Extract(fmt.Sprintf(expressionFormat, args...))
}
