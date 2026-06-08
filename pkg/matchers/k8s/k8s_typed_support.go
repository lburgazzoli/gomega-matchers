package k8s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func createTyped[T client.Object](
	ctx context.Context,
	m *Resources,
	obj T,
	opts ...client.CreateOption,
) (*unstructured.Unstructured, error) {
	current, err := cloneTyped(obj)
	if err != nil {
		return nil, err
	}

	gvk, _, err := m.extractGVKAndKey(current)
	if err != nil {
		return nil, err
	}

	if err := m.client.Create(ctx, current, opts...); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	return fetchUnstructured(ctx, m, gvk, objectKeyFromObject(current))
}

func deleteTyped[T client.Object](
	ctx context.Context,
	m *Resources,
	obj T,
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

func updateTyped[T client.Object](
	ctx context.Context,
	m *Resources,
	obj T,
	fn func(T),
	opts ...client.UpdateOption,
) (*unstructured.Unstructured, error) {
	gvk, key, err := m.extractGVKAndKey(obj)
	if err != nil {
		return nil, err
	}

	current, ok := obj.DeepCopyObject().(T)
	if !ok {
		return nil, fmt.Errorf("failed to convert deep copy to %T", obj)
	}

	err = m.client.Get(ctx, key.ToNamespacedName(), current)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource for update: %w", err)
	}

	fn(current)

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

func upsertTyped[T client.Object](
	ctx context.Context,
	m *Resources,
	obj T,
	fn func(T),
	createOpts ...client.CreateOption,
) (*unstructured.Unstructured, error) {
	gvk, key, err := m.extractGVKAndKey(obj)
	if err != nil {
		return nil, err
	}

	current, err := cloneTyped(obj)
	if err != nil {
		return nil, err
	}

	err = m.client.Get(ctx, key.ToNamespacedName(), current)
	if err == nil {
		fn(current)

		if err := m.client.Update(ctx, current); err != nil {
			return nil, fmt.Errorf("failed to update resource: %w", err)
		}

		return fetchUnstructured(ctx, m, gvk, key)
	}

	if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to get resource for upsert: %w", err)
	}

	return createUpsertedTyped(ctx, m, obj, fn, createOpts...)
}

func cloneTyped[T client.Object](obj T) (T, error) {
	current, ok := obj.DeepCopyObject().(T)
	if !ok {
		var zero T

		return zero, fmt.Errorf("failed to convert deep copy to %T", obj)
	}

	return current, nil
}

func fetchUnstructured(
	ctx context.Context,
	m *Resources,
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

func createUpsertedTyped[T client.Object](
	ctx context.Context,
	m *Resources,
	obj T,
	fn func(T),
	createOpts ...client.CreateOption,
) (*unstructured.Unstructured, error) {
	current, err := cloneTyped(obj)
	if err != nil {
		return nil, err
	}

	fn(current)

	if err := m.client.Create(ctx, current, createOpts...); err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	gvk, _, err := m.extractGVKAndKey(current)
	if err != nil {
		return nil, err
	}

	return fetchUnstructured(ctx, m, gvk, objectKeyFromObject(current))
}

func objectKeyFromObject(obj client.Object) ObjectKey {
	return ObjectKey{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}
