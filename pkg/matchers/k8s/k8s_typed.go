package k8s

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Matcher wraps a Kubernetes client.Client and provides typed helper functions
// that work with typed Kubernetes objects (e.g., *corev1.ConfigMap) and return
// unstructured results compatible with Gomega assertions and JQ matchers.
type Matcher struct {
	client client.Client
	scheme *runtime.Scheme
}

// New creates a new typed Matcher wrapping the provided client.Client.
// The scheme is used to extract GVK information from typed objects.
func New(cli client.Client, scheme *runtime.Scheme) *Matcher {
	return &Matcher{
		client: cli,
		scheme: scheme,
	}
}

// Get retrieves a Kubernetes resource using a typed object.
// The GVK is automatically extracted from the object type using the scheme.
// Returns an unstructured object compatible with JQ matchers and Gomega assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	obj, err := k.Get(ctx, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})
//	// obj is *unstructured.Unstructured
func (m *Matcher) Get(
	ctx context.Context,
	obj client.Object,
	opts ...client.GetOption,
) (*unstructured.Unstructured, error) {
	gvk, key, err := m.extractGVKAndKey(obj)
	if err != nil {
		return nil, err
	}

	result := &unstructured.Unstructured{}
	result.SetGroupVersionKind(gvk)

	err = m.client.Get(ctx, key.ToNamespacedName(), result, opts...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// List retrieves a list of Kubernetes resources using a typed list object.
// The GVK is automatically extracted from the list type using the scheme.
// Returns an unstructured list compatible with JQ matchers and Gomega assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	list, err := k.List(ctx, &corev1.ConfigMapList{},
//		client.InNamespace("default"),
//		client.MatchingLabels{"app": "myapp"},
//	)
//	// list is *unstructured.UnstructuredList
func (m *Matcher) List(
	ctx context.Context,
	list client.ObjectList,
	opts ...client.ListOption,
) (*unstructured.UnstructuredList, error) {
	gvks, _, err := m.scheme.ObjectKinds(list)
	if err != nil {
		return nil, fmt.Errorf("failed to get GVK from list: %w", err)
	}

	if len(gvks) == 0 {
		return nil, fmt.Errorf("no GVK found for list type %T", list)
	}

	result := &unstructured.UnstructuredList{}
	result.SetGroupVersionKind(gvks[0])

	err = m.client.List(ctx, result, opts...)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Delete deletes a Kubernetes resource using a typed object.
// The GVK and object key are automatically extracted from the object.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	err := k.Delete(ctx, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})
func (m *Matcher) Delete(
	ctx context.Context,
	obj client.Object,
	opts ...client.DeleteOption,
) error {
	gvk, _, err := m.extractGVKAndKey(obj)
	if err != nil {
		return err
	}

	target := &unstructured.Unstructured{}
	target.SetGroupVersionKind(gvk)
	target.SetName(obj.GetName())
	target.SetNamespace(obj.GetNamespace())

	return m.client.Delete(ctx, target, opts...)
}

// Update retrieves a Kubernetes resource, applies an update function, and updates it.
// This follows the Komega-style pattern for type-safe updates.
// Returns the updated unstructured object.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	obj, err := k.Update(ctx, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		cm.Data["key"] = "new-value"
//	})
func (m *Matcher) Update(
	ctx context.Context,
	obj client.Object,
	updateFunc func(client.Object),
	opts ...client.UpdateOption,
) (*unstructured.Unstructured, error) {
	gvk, key, err := m.extractGVKAndKey(obj)
	if err != nil {
		return nil, err
	}

	current, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return nil, errors.New("failed to convert deep copy to client.Object")
	}

	err = m.client.Get(ctx, key.ToNamespacedName(), current)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for update: %w", err)
	}

	updateFunc(current)

	err = m.client.Update(ctx, current, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	result := &unstructured.Unstructured{}
	result.SetGroupVersionKind(gvk)

	err = m.client.Get(ctx, key.ToNamespacedName(), result)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated resource: %w", err)
	}

	return result, nil
}

// extractGVKAndKey extracts GroupVersionKind and ObjectKey from a typed Kubernetes object.
func (m *Matcher) extractGVKAndKey(obj client.Object) (schema.GroupVersionKind, ObjectKey, error) {
	gvks, _, err := m.scheme.ObjectKinds(obj)
	if err != nil {
		return schema.GroupVersionKind{}, ObjectKey{}, fmt.Errorf("failed to get GVK from object: %w", err)
	}

	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, ObjectKey{}, fmt.Errorf("no GVK found for object type %T", obj)
	}

	key := ObjectKey{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	return gvks[0], key, nil
}
