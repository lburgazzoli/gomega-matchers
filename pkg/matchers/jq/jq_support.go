package jq

import (
	"github.com/itchyny/gojq"
)

// Run executes a compiled JQ query against the provided data and returns the first result.
// Returns false if the query produces no results, or an error if query execution fails.
func Run(query *gojq.Query, data any) (any, error) {
	it := query.Run(data)

	v, ok := it.Next()
	if !ok {
		return false, nil
	}

	if err, ok := v.(error); ok {
		return false, err
	}

	return v, nil
}
