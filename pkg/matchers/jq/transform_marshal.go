package jq

import (
	"encoding/json"
	"fmt"
)

func AsJSON() func(in any) (any, error) {
	return func(in any) (any, error) {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal result, %w", err)
		}

		return data, nil
	}
}
