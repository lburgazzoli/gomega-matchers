package k8s

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// MatchPartially matches a Kubernetes resource using
// apiequality.Semantic.DeepDerivative(expected, actual). Expected and actual may
// be typed resource values/pointers or unstructured resources. The expected
// resource is copied when constructing the matcher.
//
// Kubernetes' native rules apply: nil pointers, empty strings, and empty
// maps/slices in expected are ignored. Numeric zero and false remain assertions,
// even when omitted from a typed struct literal. Map keys in expected must exist
// in actual. Slices match by position and allow additional trailing actual
// elements; containers are not looked up by name. Use Container, Volume, or
// EnvVar with other Gomega matchers for named lookup, and explicit field
// matchers to assert empty or nil values.
//
// Typed inputs must have the same concrete resource type. Mixed typed and
// unstructured inputs require matching APIVersion and Kind on both sides.
// An unstructured actual is converted into the expected Go type. An
// unstructured expected is compared with the actual's unstructured map.
// Quantities and times compare semantically in typed comparisons; strings in
// unstructured comparisons are literal. No API defaulting or version conversion
// is performed. Diagnostics show actual and expected, not individual field paths.
func MatchPartially(expected any) types.GomegaMatcher {
	matcher := &partialMatcher{}
	obj, err := matcher.object(expected)
	if err != nil {
		matcher.err = fmt.Errorf("invalid partial pattern: %w", err)

		return matcher
	}
	copyOfObject, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		matcher.err = errors.New("partial pattern copy must implement client.Object")

		return matcher
	}
	matcher.expected = copyOfObject

	return matcher
}

type partialMatcher struct {
	expected client.Object
	err      error
}

func (m *partialMatcher) Match(actual any) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	obj, err := m.object(actual)
	if err != nil {
		return false, err
	}
	matches, identityErr := m.identity(obj)
	switch {
	case identityErr != nil:
		return false, identityErr
	case !matches:
		return false, nil
	}

	return m.matchObject(obj)
}

func (m *partialMatcher) FailureMessage(actual any) string {
	return format.Message(actual, "to partially match", m.expected)
}

func (m *partialMatcher) NegatedFailureMessage(actual any) string {
	return format.Message(actual, "not to partially match", m.expected)
}

func (m *partialMatcher) matchObject(obj client.Object) (bool, error) {
	expectedUnstructured, expectedIsUnstructured := m.expected.(*unstructured.Unstructured)
	actualUnstructured, actualIsUnstructured := obj.(*unstructured.Unstructured)
	switch {
	case expectedIsUnstructured:
		data, conversionErr := toUnstructuredMap(obj)
		if conversionErr != nil {
			return false, conversionErr
		}

		return apiequality.Semantic.DeepDerivative(expectedUnstructured.Object, data), nil
	case actualIsUnstructured:
		converted := reflect.New(reflect.TypeOf(m.expected).Elem()).Interface()
		conversionErr := runtime.DefaultUnstructuredConverter.FromUnstructured(actualUnstructured.Object, converted)
		if conversionErr != nil {
			return false, fmt.Errorf("converting actual into %T: %w", m.expected, conversionErr)
		}

		return apiequality.Semantic.DeepDerivative(m.expected, converted), nil
	default:
		return apiequality.Semantic.DeepDerivative(m.expected, obj), nil
	}
}

func (m *partialMatcher) identity(actual client.Object) (bool, error) {
	_, expectedUnstructured := m.expected.(*unstructured.Unstructured)
	_, actualUnstructured := actual.(*unstructured.Unstructured)
	if expectedUnstructured == actualUnstructured {
		return true, nil
	}
	gvk := m.expected.GetObjectKind().GroupVersionKind()
	actualGVK := actual.GetObjectKind().GroupVersionKind()
	if gvk.Kind == "" || gvk.Version == "" || actualGVK.Kind == "" || actualGVK.Version == "" {
		return false, errors.New("mixed typed/unstructured matching requires APIVersion and Kind on both objects")
	}

	return gvk == actualGVK, nil
}

func (*partialMatcher) object(value any) (client.Object, error) {
	normalized, err := normalizeStructPointer(value)
	if err != nil {
		return nil, err
	}

	return asObject(normalized)
}
