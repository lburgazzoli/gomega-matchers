package jq

import (
	"fmt"

	"github.com/itchyny/gojq"
	"github.com/onsi/gomega/format"
)

func Extract(expression string) func(in any) (any, error) {
	return func(in any) (any, error) {
		query, err := gojq.Parse(expression)
		if err != nil {
			return nil, fmt.Errorf("unable to parse expression %s, %w", expression, err)
		}

		actualString, ok := toString(in)
		if !ok {
			return false, fmt.Errorf("extract requires a string, stringer, or []byte. got:\n%s", format.Object(in, 1))
		}

		if len(actualString) == 0 {
			return nil, nil
		}

		b := []byte(actualString)

		it, err := runQuery(query, b)
		if err != nil {
			return false, fmt.Errorf("unable to run query, %w", err)
		}

		v, ok := it.Next()
		if !ok {
			return false, nil
		}

		if err, ok := v.(error); ok {
			return false, err
		}

		return v, nil
	}
}
