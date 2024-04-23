package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/itchyny/gojq"
)

func formattedMessage(comparisonMessage string, failurePath []interface{}) string {
	diffMessage := ""

	if len(failurePath) != 0 {
		diffMessage = "\n\nfirst mismatched key: " + formattedFailurePath(failurePath)
	}

	return comparisonMessage + diffMessage
}

func formattedFailurePath(failurePath []interface{}) string {
	formattedPaths := make([]string, 0)

	for i := len(failurePath) - 1; i >= 0; i-- {
		switch p := failurePath[i].(type) {
		case int:
			val := fmt.Sprintf(`[%d]`, p)
			formattedPaths = append(formattedPaths, val)
		default:
			if i != len(failurePath)-1 {
				formattedPaths = append(formattedPaths, ".")
			}

			val := fmt.Sprintf(`"%s"`, p)
			formattedPaths = append(formattedPaths, val)
		}
	}

	return strings.Join(formattedPaths, "")
}

func toString(a interface{}) (string, bool) {
	aString, isString := a.(string)
	if isString {
		return aString, true
	}

	aBytes, isBytes := a.([]byte)
	if isBytes {
		return string(aBytes), true
	}

	aStringer, isStringer := a.(fmt.Stringer)
	if isStringer {
		return aStringer.String(), true
	}

	aJSONRawMessage, isJSONRawMessage := a.(json.RawMessage)
	if isJSONRawMessage {
		return string(aJSONRawMessage), true
	}

	return "", false
}

func runQuery(query *gojq.Query, in []byte) (gojq.Iter, error) {
	var it gojq.Iter

	// rough check for object vs array
	switch in[0] {
	case '{':
		data := make(map[string]any)
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		it = query.Run(data)
	case '[':
		var data []any
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		it = query.Run(data)
	default:
		return nil, errors.New("a Json Array or Object is required")
	}

	return it, nil
}
