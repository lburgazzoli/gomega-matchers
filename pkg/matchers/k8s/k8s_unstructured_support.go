package k8s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func fetchUnstructuredResource(
	ctx context.Context,
	m *UnstructuredResources,
	gvk schema.GroupVersionKind,
	key ObjectKey,
) (*unstructured.Unstructured, error) {
	result := &unstructured.Unstructured{}
	result.SetGroupVersionKind(gvk)

	err := m.client.Get(ctx, key.ToNamespacedName(), result)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	return result, nil
}

func objectKeyFromUnstructured(obj *unstructured.Unstructured) ObjectKey {
	return ObjectKey{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

func createUpsertedUnstructured(
	ctx context.Context,
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	createOpts ...client.CreateOption,
) (*unstructured.Unstructured, error) {
	current := obj.DeepCopy()
	fn(current)

	if err := m.client.Create(ctx, current, createOpts...); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return fetchUnstructuredResource(ctx, m, current.GroupVersionKind(), objectKeyFromUnstructured(current))
}

func applyUnstructuredUpdate(
	ctx context.Context,
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	opts ...client.UpdateOption,
) (*unstructured.Unstructured, error) {
	gvk := obj.GroupVersionKind()
	key := objectKeyFromUnstructured(obj)
	current := obj.DeepCopy()

	err := m.client.Get(ctx, key.ToNamespacedName(), current)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for update: %w", err)
	}

	fn(current)

	err = m.client.Update(ctx, current, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource: %w", err)
	}

	return fetchUnstructuredResource(ctx, m, gvk, key)
}

func applyUnstructuredStatusUpdate(
	ctx context.Context,
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	opts ...client.SubResourceUpdateOption,
) (*unstructured.Unstructured, error) {
	gvk := obj.GroupVersionKind()
	key := objectKeyFromUnstructured(obj)
	current := obj.DeepCopy()

	err := m.client.Get(ctx, key.ToNamespacedName(), current)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for status update: %w", err)
	}

	fn(current)

	err = m.client.Status().Update(ctx, current, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource status: %w", err)
	}

	return fetchUnstructuredResource(ctx, m, gvk, key)
}

func applyUnstructuredUpsert(
	ctx context.Context,
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	createOpts ...client.CreateOption,
) (*unstructured.Unstructured, error) {
	gvk := obj.GroupVersionKind()
	key := objectKeyFromUnstructured(obj)
	current := obj.DeepCopy()

	err := m.client.Get(ctx, key.ToNamespacedName(), current)
	if err == nil {
		fn(current)

		if err := m.client.Update(ctx, current); err != nil {
			return nil, fmt.Errorf("failed to update resource: %w", err)
		}

		return fetchUnstructuredResource(ctx, m, gvk, key)
	}

	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get resource for upsert: %w", err)
	}

	return createUpsertedUnstructured(ctx, m, obj, fn, createOpts...)
}
