package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/onsi/gomega/format"
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

func toType(in any) (any, error) {
	switch v := in.(type) {
	case string:
		d, err := byteToType([]byte(v))
		if err != nil {
			return nil, err
		}

		return d, nil
	case []byte:
		d, err := byteToType(v)
		if err != nil {
			return nil, err
		}

		return d, nil
	case json.RawMessage:
		d, err := byteToType(v)
		if err != nil {
			return nil, err
		}

		return d, nil
	}

	//nolint:exhaustive
	switch reflect.TypeOf(in).Kind() {
	case reflect.Map:
		return in, nil
	case reflect.Slice:
		return in, nil
	default:
		return nil, fmt.Errorf("unsuported type:\n%s", format.Object(in, 1))
	}
}

func byteToType(in []byte) (any, error) {
	switch in[0] {
	case '{':
		data := make(map[string]any)
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		return data, nil
	case '[':
		var data []any
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		return data, nil
	default:
		return nil, errors.New("a Json Array or Object is required")
	}
}
