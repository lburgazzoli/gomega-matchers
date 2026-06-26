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

const (
	fieldContainers = "containers"
	fieldSpec       = "spec"
	fieldTemplate   = "template"
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

// PodTemplate returns a transform function that extracts a PodTemplate-like
// object as corev1.PodTemplateSpec.
//
// Supported inputs include:
//   - *corev1.PodTemplate objects via .template
//   - workload objects with .spec.template
//   - *unstructured.Unstructured values with either shape
//
// Returns nil when the input is supported but does not define a pod template.
//
// Example:
//
//	WithTransform(k8s.PodTemplate(), HaveField("Spec.Containers", HaveLen(1)))
func PodTemplate() func(any) (any, error) {
	return extractPodTemplate
}

// Containers returns a transform function that extracts pod spec containers as
// []corev1.Container.
//
// Supported inputs include:
//   - pod-like objects with .spec.containers
//   - PodTemplate-like objects with .template.spec.containers
//   - workload objects with .spec.template.spec.containers
//   - CronJob objects with .spec.jobTemplate.spec.template.spec.containers
//   - *unstructured.Unstructured values with any of the above shapes
//
// Returns nil when the input is supported but does not define containers.
//
// Example:
//
//	WithTransform(k8s.Containers(), ContainElement(HaveField("Name", Equal("app"))))
func Containers() func(any) (any, error) {
	return extractContainers
}

// EnvVars returns a transform function that extracts container environment
// variables as []corev1.EnvVar.
//
// Supported inputs include typed container structs, container maps, and values
// returned by Containers().
//
// Returns nil when the input is supported but does not define env vars.
//
// Example:
//
//	WithTransform(k8s.EnvVars(), ContainElement(HaveField("Name", Equal("FOO"))))
func EnvVars() func(any) (any, error) {
	return extractEnvVars
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

func extractPodTemplate(in any) (any, error) {
	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	for _, path := range [][]string{
		{fieldTemplate},
		{fieldSpec, fieldTemplate},
		{fieldSpec, "jobTemplate", fieldSpec, fieldTemplate},
	} {
		template, ok, nestedErr := nestedMap(m, path...)
		if nestedErr != nil {
			return nil, nestedErr
		}

		if ok {
			return convertValue[corev1.PodTemplateSpec](template, "pod template")
		}
	}

	return nil, nil //nolint:nilnil
}

func extractContainers(in any) (any, error) {
	switch obj := in.(type) { // PodSpec stores containers at the root, not under spec.
	case *corev1.PodSpec:
		return obj.Containers, nil
	case corev1.PodSpec:
		return obj.Containers, nil
	}

	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	paths := [][]string{
		{fieldSpec, fieldContainers},
		{fieldTemplate, fieldSpec, fieldContainers},
		{fieldSpec, fieldTemplate, fieldSpec, fieldContainers},
		{fieldSpec, "jobTemplate", fieldSpec, fieldTemplate, fieldSpec, fieldContainers},
	}

	for _, path := range paths {
		containers, ok, nestedErr := nestedSlice(m, path...)
		if nestedErr != nil {
			return nil, nestedErr
		}

		if ok {
			return convertValue[[]corev1.Container](containers, "containers")
		}
	}

	return nil, nil //nolint:nilnil
}

func extractEnvVars(in any) (any, error) {
	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	envVars, ok, err := nestedSlice(m, "env")
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil //nolint:nilnil
	}

	return convertValue[[]corev1.EnvVar](envVars, "env vars")
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

func toMap(in any) (map[string]any, error) {
	switch obj := in.(type) {
	case map[string]any:
		return obj, nil
	case *unstructured.Unstructured:
		return obj.Object, nil
	}

	normalized, err := normalizeStructPointer(in)
	if err != nil {
		return nil, err
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(normalized)
	if err != nil {
		return nil, fmt.Errorf("converting %T to map: %w", normalized, err)
	}

	return m, nil
}

func normalizeStructPointer(in any) (any, error) {
	v := reflect.ValueOf(in)
	if !v.IsValid() || (v.Kind() == reflect.Pointer && v.IsNil()) {
		return nil, fmt.Errorf("expected struct, pointer to struct, or map[string]any, got %T", in)
	}

	if v.Kind() == reflect.Struct {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)

		return ptr.Interface(), nil
	}

	if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Struct {
		return in, nil
	}

	return nil, fmt.Errorf("expected struct, pointer to struct, or map[string]any, got %T", in)
}

func convertValue[T any](raw any, what string) (T, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return zero[T](), fmt.Errorf("marshaling %s: %w", what, err)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero[T](), fmt.Errorf("unmarshaling %s into %T: %w", what, result, err)
	}

	return result, nil
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

func nestedMap(m map[string]any, fields ...string) (map[string]any, bool, error) {
	result, found, err := unstructured.NestedMap(m, fields...)
	if err != nil {
		return nil, false, fmt.Errorf("extracting %v: %w", fields, err)
	}

	return result, found, nil
}

func nestedSlice(m map[string]any, fields ...string) ([]any, bool, error) {
	result, found, err := unstructured.NestedSlice(m, fields...)
	if err != nil {
		return nil, false, fmt.Errorf("extracting %v: %w", fields, err)
	}

	return result, found, nil
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
