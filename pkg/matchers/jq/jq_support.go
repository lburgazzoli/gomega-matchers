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

// Run executes a compiled JQ query against the provided data and returns the first result.
// Returns false if the query produces no results, or an error if query execution fails.
func Run(query *gojq.Query, data any) (any, error) {
	return runFirstResult(query, data, false)
}

func runRequiredResult(query *gojq.Query, data any, expression string) (any, error) {
	result, err := runFirstResult(query, data, nil)
	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, fmt.Errorf("jq transform %q produced no result", expression)
	}

	return result, nil
}

func runFirstResult(query *gojq.Query, data any, noResult any) (any, error) {
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
