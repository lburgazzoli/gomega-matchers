package k8s

import (
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Data returns a transform function that extracts the full .data field from
// supported Kubernetes objects.
//
// Supported inputs include typed ConfigMaps, typed Secrets, and
// *unstructured.Unstructured objects.
//
// Example:
//
//	WithTransform(k8s.Data(), HaveKeyWithValue("foo", "bar"))
func Data() func(any) (any, error) {
	return func(in any) (any, error) {
		switch obj := in.(type) {
		case *corev1.ConfigMap:
			return obj.Data, nil
		case *corev1.Secret:
			return obj.Data, nil
		case *unstructured.Unstructured:
			return obj.Object["data"], nil
		default:
			return nil, fmt.Errorf("expected *corev1.ConfigMap, *corev1.Secret, or *unstructured.Unstructured, got %T", in)
		}
	}
}

// ListItems returns a transform function that extracts the Items slice from
// supported Kubernetes list objects.
//
// Supported inputs include typed Kubernetes list objects and
// *unstructured.UnstructuredList values.
//
// Example:
//
//	WithTransform(k8s.ListItems(), HaveLen(2))
func ListItems() func(any) (any, error) {
	return func(in any) (any, error) {
		obj, err := runtimeListObject(in)
		if err != nil {
			return nil, err
		}

		items, err := meta.ExtractList(obj)
		if err != nil {
			return nil, fmt.Errorf("expected runtime.Object list, got %T", in)
		}

		return items, nil
	}
}

func runtimeListObject(in any) (runtime.Object, error) {
	v := reflect.ValueOf(in)
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return nil, fmt.Errorf("expected runtime.Object list, got %T", in)
	}

	obj, ok := in.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("expected runtime.Object list, got %T", in)
	}

	return obj, nil
}
