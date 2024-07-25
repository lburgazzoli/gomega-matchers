package yq

import (
	"bufio"
	"container/list"
	"fmt"
	"strings"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/onsi/gomega/format"
)

const (
	defaultIndent = 2
)

var (
	//nolint:gochecknoglobals
	decoder = yqlib.NewYamlDecoder(yqlib.YamlPreferences{
		Indent:                      defaultIndent,
		ColorsEnabled:               false,
		LeadingContentPreProcessing: true,
		PrintDocSeparators:          true,
		UnwrapScalar:                true,
		EvaluateTogether:            false,
	})

	//nolint:gochecknoglobals
	encoder = yqlib.NewYamlEncoder(yqlib.YamlPreferences{
		Indent:                      defaultIndent,
		ColorsEnabled:               false,
		LeadingContentPreProcessing: true,
		PrintDocSeparators:          true,
		UnwrapScalar:                true,
		EvaluateTogether:            false,
	})

	//nolint:gochecknoglobals
	evaluator = yqlib.NewAllAtOnceEvaluator()
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

func toString(in any) (string, error) {
	switch v := in.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("unsupported type:\n%s", format.Object(in, 1))
	}
}

func evaluate(expression string, actual interface{}) (*list.List, error) {
	data, err := toString(actual)
	if err != nil {
		return nil, err
	}

	reader := bufio.NewReader(strings.NewReader(data))

	documents, err := yqlib.ReadDocuments(reader, decoder)
	if err != nil {
		return nil, fmt.Errorf("failure reading document: %w", err)
	}

	results, err := evaluator.EvaluateCandidateNodes(expression, documents)
	if err != nil {
		return nil, fmt.Errorf("failure evaluating expression: %w", err)
	}

	return results, nil
}
