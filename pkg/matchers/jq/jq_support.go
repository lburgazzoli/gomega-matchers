package jq

import (
	"errors"
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega"
)

// ErrMultipleResults indicates that an expression returned more than one result
// where exactly one result was required.
var ErrMultipleResults = errors.New("jq expression produced multiple results")

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
	result, found, err := evalExactlyOne(query, data, expression)
	if err != nil {
		if errors.Is(err, ErrMultipleResults) {
			return nil, gomega.StopTrying(err.Error()).Wrap(err)
		}

		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("jq transform %q produced no result", expression)
	}

	return result, nil
}

func evalOptional(query *gojq.Query, data any, expression string) (any, error) {
	result, _, err := evalExactlyOne(query, data, expression)

	return result, err
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

func evalExactlyOne(query *gojq.Query, data any, expression string) (any, bool, error) {
	it := query.Run(data)

	v, ok := it.Next()
	if !ok {
		return nil, false, nil
	}

	if err, ok := v.(error); ok {
		return nil, false, err
	}

	next, ok := it.Next()
	if !ok {
		return v, true, nil
	}

	if err, ok := next.(error); ok {
		return nil, false, err
	}

	return nil, false, fmt.Errorf("jq expression %q produced multiple results: %w", expression, ErrMultipleResults)
}

func evalAll(query *gojq.Query, data any) ([]any, error) {
	results := make([]any, 0)
	it := query.Run(data)

	for {
		v, ok := it.Next()
		if !ok {
			return results, nil
		}

		if err, ok := v.(error); ok {
			return nil, err
		}

		results = append(results, v)
	}
}
