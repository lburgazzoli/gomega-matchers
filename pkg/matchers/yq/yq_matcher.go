package yq

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

func Match(format string, args ...any) types.GomegaMatcher {
	return &yqMatcher{
		Expression: fmt.Sprintf(format, args...),
	}
}

var _ types.GomegaMatcher = &yqMatcher{}

type yqMatcher struct {
	Expression       string
	firstFailurePath []interface{}
}

func (matcher *yqMatcher) Match(actual interface{}) (bool, error) {
	results, err := evaluate(matcher.Expression, actual)
	if err != nil {
		return false, err
	}

	if results == nil {
		return false, nil
	}

	if results.Len() != 1 {
		return false, errors.New("TODO_1")
	}

	n, ok := results.Front().Value.(*yqlib.CandidateNode)
	if !ok {
		return false, errors.New("TODO_2")
	}

	match, err := strconv.ParseBool(n.Value)
	if err != nil {
		return false, fmt.Errorf("failure parsing result: %w", err)
	}

	return match, nil
}

func (matcher *yqMatcher) FailureMessage(actual interface{}) string {
	return formattedMessage(format.Message(fmt.Sprintf("%v", actual), "to match expression", matcher.Expression), matcher.firstFailurePath)
}

func (matcher *yqMatcher) NegatedFailureMessage(actual interface{}) string {
	return formattedMessage(format.Message(fmt.Sprintf("%v", actual), "not to match expression", matcher.Expression), matcher.firstFailurePath)
}
