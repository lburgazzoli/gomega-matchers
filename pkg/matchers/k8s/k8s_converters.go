package k8s

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
)

//nolint:gochecknoinits
func init() {
	jq.RegisterConverter(k8sObjectConverter)
	jq.RegisterConverter(k8sListConverter)
}

func k8sListConverter(in any) (any, error) {
	switch v := in.(type) {
	case unstructured.UnstructuredList:
		return convertUnstructuredListItems(v.Items), nil
	case *unstructured.UnstructuredList:
		if v == nil {
			return nil, jq.ErrTypeNotSupported
		}

		return convertUnstructuredListItems(v.Items), nil
	case client.ObjectList:
		return convertTypedListItems(v)
	default:
		return nil, jq.ErrTypeNotSupported
	}
}

func convertUnstructuredListItems(items []unstructured.Unstructured) []any {
	result := make([]any, len(items))
	for i, item := range items {
		result[i] = item.Object
	}

	return result
}

func convertTypedListItems(list client.ObjectList) (any, error) {
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, jq.ErrTypeNotSupported
	}

	result := make([]any, len(items))
	for i, item := range items {
		data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
		if err != nil {
			return nil, err
		}

		result[i] = data
	}

	return result, nil
}

func k8sObjectConverter(in any) (any, error) {
	switch v := in.(type) {
	case unstructured.Unstructured:
		return v.Object, nil
	case *unstructured.Unstructured:
		if v == nil {
			return nil, jq.ErrTypeNotSupported
		}

		return v.Object, nil
	case client.Object:
		return runtime.DefaultUnstructuredConverter.ToUnstructured(v)
	default:
		return nil, jq.ErrTypeNotSupported
	}
}
