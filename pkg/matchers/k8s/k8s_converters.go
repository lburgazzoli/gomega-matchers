package k8s

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
)

//nolint:gochecknoinits
func init() {
	jq.RegisterConverter(unstructuredConverter)
	jq.RegisterConverter(unstructuredPtrConverter)
	jq.RegisterConverter(unstructuredListPtrConverter)
}

func unstructuredConverter(in any) (any, error) {
	v, ok := in.(unstructured.Unstructured)
	if !ok {
		return nil, jq.ErrTypeNotSupported
	}

	return v.Object, nil
}

func unstructuredPtrConverter(in any) (any, error) {
	v, ok := in.(*unstructured.Unstructured)
	if !ok {
		return nil, jq.ErrTypeNotSupported
	}

	return v.Object, nil
}

func unstructuredListPtrConverter(in any) (any, error) {
	v, ok := in.(*unstructured.UnstructuredList)
	if !ok {
		return nil, jq.ErrTypeNotSupported
	}

	items := make([]any, len(v.Items))
	for i, item := range v.Items {
		items[i] = item.Object
	}

	return items, nil
}
