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
	fieldContainers  = "containers"
	fieldLimits      = "limits"
	fieldRequests    = "requests"
	fieldResources   = "resources"
	fieldSpec        = "spec"
	fieldTemplate    = "template"
	fieldJobTemplate = "jobTemplate"
	fieldVolumes     = "volumes"
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

// Container returns a transform function that extracts a named container as
// corev1.Container.
//
// Supported inputs are the same as Containers().
//
// Returns an error when the container is not found.
//
// Example:
//
//	WithTransform(k8s.Container("app"), HaveField("Image", Equal("example/app:latest")))
func Container(name string) func(any) (any, error) {
	return func(in any) (any, error) {
		items, err := extractContainers(in)
		if err != nil {
			return nil, err
		}

		return namedContainer(items, name)
	}
}

// Volumes returns a transform function that extracts pod volumes as
// []corev1.Volume.
//
// Supported inputs include pod-like objects with .spec.volumes, PodTemplate
// objects, workload objects with .spec.template.spec.volumes, CronJob objects,
// and unstructured values with any of those shapes.
//
// Returns nil when the input is supported but does not define volumes.
//
// Example:
//
//	WithTransform(k8s.Volumes(), ContainElement(HaveField("Name", Equal("config"))))
func Volumes() func(any) (any, error) {
	return extractVolumes
}

// Volume returns a transform function that extracts a named pod volume as
// corev1.Volume.
//
// Supported inputs are the same as Volumes(). Returns an error when the volume
// is not found.
//
// Example:
//
//	WithTransform(k8s.Volume("config"), HaveField("ConfigMap.Name", Equal("settings")))
func Volume(name string) func(any) (any, error) {
	return func(in any) (any, error) {
		items, err := extractVolumes(in)
		if err != nil {
			return nil, err
		}

		return namedVolume(items, name)
	}
}

// ResourceRequests returns a transform function that extracts a container's
// resource requests as corev1.ResourceList.
//
// Supported inputs include typed containers and container maps, including
// values returned by Container(). Missing resource requests return nil.
//
// Example:
//
//	WithTransform(k8s.Container("app"), WithTransform(k8s.ResourceRequests(), HaveKey(corev1.ResourceCPU)))
func ResourceRequests() func(any) (any, error) {
	return func(in any) (any, error) {
		return extractResourceList(in, fieldRequests)
	}
}

// ResourceLimits returns a transform function that extracts a container's
// resource limits as corev1.ResourceList.
//
// Supported inputs include typed containers and container maps, including
// values returned by Container(). Missing resource limits return nil.
//
// Example:
//
//	WithTransform(k8s.Container("app"), WithTransform(k8s.ResourceLimits(), HaveKey(corev1.ResourceMemory)))
func ResourceLimits() func(any) (any, error) {
	return func(in any) (any, error) {
		return extractResourceList(in, fieldLimits)
	}
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

// EnvVar returns a transform function that extracts a named environment
// variable as corev1.EnvVar.
//
// Supported inputs are the same as EnvVars().
//
// Returns an error when the env var is not found.
//
// Example:
//
//	WithTransform(k8s.EnvVar("LOG_LEVEL"), HaveField("Value", Equal("debug")))
func EnvVar(name string) func(any) (any, error) {
	return func(in any) (any, error) {
		items, err := extractEnvVars(in)
		if err != nil {
			return nil, err
		}

		return namedEnvVar(items, name)
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

func extractPodTemplate(in any) (any, error) {
	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	for _, path := range [][]string{
		{fieldTemplate},
		{fieldSpec, fieldTemplate},
		{fieldSpec, fieldJobTemplate, fieldSpec, fieldTemplate},
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

func extractVolumes(in any) (any, error) {
	switch obj := in.(type) { // PodSpec stores volumes at the root, not under spec.
	case *corev1.PodSpec:
		return obj.Volumes, nil
	case corev1.PodSpec:
		return obj.Volumes, nil
	}

	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	paths := [][]string{
		{fieldSpec, fieldVolumes},
		{fieldTemplate, fieldSpec, fieldVolumes},
		{fieldSpec, fieldTemplate, fieldSpec, fieldVolumes},
		{fieldSpec, fieldJobTemplate, fieldSpec, fieldTemplate, fieldSpec, fieldVolumes},
	}

	for _, path := range paths {
		volumes, ok, nestedErr := nestedSlice(m, path...)
		if nestedErr != nil {
			return nil, nestedErr
		}

		if ok {
			return convertValue[[]corev1.Volume](volumes, "volumes")
		}
	}

	return nil, nil //nolint:nilnil
}

func extractEnvVars(in any) (any, error) {
	switch obj := in.(type) {
	case *corev1.Container:
		return obj.Env, nil
	case corev1.Container:
		return obj.Env, nil
	case *corev1.EnvVar:
		return []corev1.EnvVar{*obj}, nil
	case corev1.EnvVar:
		return []corev1.EnvVar{obj}, nil
	}

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

func extractResourceList(in any, field string) (any, error) {
	switch obj := in.(type) {
	case *corev1.Container:
		if obj == nil {
			return nil, fmt.Errorf("expected *corev1.Container or container map, got %T", in)
		}

		return resourceListFor(obj.Resources, field), nil
	case corev1.Container:
		return resourceListFor(obj.Resources, field), nil
	}

	m, err := toMap(in)
	if err != nil {
		return nil, err
	}

	return extractResourceListFromMap(m, field)
}

func extractResourceListFromMap(m map[string]any, field string) (any, error) {
	resources, ok, nestedErr := nestedMap(m, fieldResources)
	if nestedErr != nil {
		return nil, nestedErr
	}
	if !ok {
		return nil, nil //nolint:nilnil
	}

	values, ok, nestedErr := nestedMap(resources, field)
	if nestedErr != nil {
		return nil, nestedErr
	}
	if !ok {
		return nil, nil //nolint:nilnil
	}

	return convertValue[corev1.ResourceList](values, field+" resources")
}

func resourceListFor(resources corev1.ResourceRequirements, field string) corev1.ResourceList {
	if field == fieldRequests {
		return resources.Requests
	}

	return resources.Limits
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

func namedContainer(items any, name string) (any, error) {
	if items == nil {
		return nil, notFoundError("container", name)
	}

	containers, ok := items.([]corev1.Container)
	if !ok {
		return nil, unexpectedCollectionType("container", items)
	}

	return firstContainerNamed(containers, name)
}

func namedVolume(items any, name string) (any, error) {
	if items == nil {
		return nil, notFoundError("volume", name)
	}

	volumes, ok := items.([]corev1.Volume)
	if !ok {
		return nil, unexpectedCollectionType("volume", items)
	}

	return firstVolumeNamed(volumes, name)
}

func namedEnvVar(items any, name string) (any, error) {
	if items == nil {
		return nil, notFoundError("env var", name)
	}

	envVars, ok := items.([]corev1.EnvVar)
	if !ok {
		return nil, unexpectedCollectionType("env var", items)
	}

	return firstEnvVarNamed(envVars, name)
}

func firstContainerNamed(items []corev1.Container, name string) (any, error) {
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}

	return nil, notFoundError("container", name)
}

func firstVolumeNamed(items []corev1.Volume, name string) (any, error) {
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}

	return nil, notFoundError("volume", name)
}

func firstEnvVarNamed(items []corev1.EnvVar, name string) (any, error) {
	for _, item := range items {
		if item.Name == name {
			return item, nil
		}
	}

	return nil, notFoundError("env var", name)
}

func unexpectedCollectionType(what string, items any) error {
	return fmt.Errorf("unexpected %s collection type %T", what, items)
}

func notFoundError(what string, name string) error {
	return fmt.Errorf("%s %q not found", what, name)
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
