package k8s

import (
	"encoding/json"
	"fmt"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Data returns a transform function that extracts the full .data field from
// supported Kubernetes objects.
//
// The supported set is intentionally closed to ConfigMap, Secret, and
// Unstructured — these are the types where .data has a well-defined
// meaning. For custom resources, use jq.Extract(`.data`) instead.
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

// Finalizers returns a transform function that extracts metadata.finalizers
// from supported Kubernetes objects.
//
// Supported inputs include typed Kubernetes objects and
// *unstructured.Unstructured values.
//
// Example:
//
//	WithTransform(k8s.Finalizers(), ContainElement("example.com/finalizer"))
func Finalizers() func(any) (any, error) {
	return func(in any) (any, error) {
		obj, err := asObject(in)
		if err != nil {
			return nil, err
		}

		return obj.GetFinalizers(), nil
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

// Conditions returns a transform function that extracts .status.conditions
// from supported Kubernetes objects as []map[string]any.
//
// This works uniformly with typed objects (regardless of their concrete
// condition type) and unstructured objects by converting through the
// unstructured representation.
//
// Returns nil when .status or .status.conditions is absent.
//
// Example:
//
//	WithTransform(k8s.Conditions(), ContainElement(HaveField("type", "Ready")))
func Conditions() func(any) (any, error) {
	return extractConditions
}

// ConditionsOf returns a transform function that extracts .status.conditions
// from supported Kubernetes objects, converting each condition into the
// concrete type T via JSON round-trip.
//
// Use this when you need typed condition structs for precise assertions.
//
// Returns nil when .status or .status.conditions is absent.
//
// Example:
//
//	WithTransform(k8s.ConditionsOf[metav1.Condition](), ContainElement(
//	    HaveField("Type", Equal("Ready")),
//	))
func ConditionsOf[T any]() func(any) (any, error) {
	return func(in any) (any, error) {
		raw, err := extractConditions(in)
		if err != nil {
			return nil, err
		}

		if raw == nil {
			return nil, nil
		}

		return convertConditions[T](raw)
	}
}

func convertConditions[T any](raw any) (any, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshaling conditions: %w", err)
	}

	var conditions []T
	if err := json.Unmarshal(data, &conditions); err != nil {
		return nil, fmt.Errorf("unmarshaling conditions into %T: %w", conditions, err)
	}

	return conditions, nil
}

func extractConditions(in any) (any, error) {
	m, err := toUnstructuredMap(in)
	if err != nil {
		return nil, err
	}

	status, ok := m["status"].(map[string]any)
	if !ok {
		return nil, nil //nolint:nilnil
	}

	conditions, ok := status["conditions"]
	if !ok {
		return nil, nil //nolint:nilnil
	}

	return conditions, nil
}

func toUnstructuredMap(in any) (map[string]any, error) {
	switch obj := in.(type) {
	case *unstructured.Unstructured:
		return obj.Object, nil
	case client.Object:
		return runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	default:
		return nil, fmt.Errorf("expected client.Object, got %T", in)
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
