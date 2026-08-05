package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega"
)

func parseQuery(expression string) (*gojq.Query, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("unable to parse expression %s, %w", expression, err)
	}

	return query, nil
}

func terminalJQError(err error) error {
	return gomega.StopTrying("jq expression cannot be evaluated").Wrap(err)
}

// EvalFirst executes a compiled JQ query against the provided data and returns
// the first result. Returns false if the query produces no results, or an error
// if query execution fails.
//
// Only the first value produced by the query is returned; multi-value
// expressions (e.g. `.[] | .`) silently discard subsequent results.
func EvalFirst(query *gojq.Query, data any) (any, error) {
	return evalFirstOr(query, data, false)
}

func evalRequired(query *gojq.Query, data any, expression string) (any, error) {
	result, err := evalFirstOr(query, data, nil)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("jq transform %q produced no result", expression)
	}

	return result, nil
}

func evalFirstOr(query *gojq.Query, data any, noResult any) (any, error) {
	it := query.Run(data)

	v, ok := it.Next()
	if !ok {
		return noResult, nil
	}

	if err, ok := v.(error); ok {
		return nil, err
	}

	return v, nil
}
