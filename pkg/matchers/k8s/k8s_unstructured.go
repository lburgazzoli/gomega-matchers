package k8s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// UnstructuredResources wraps a Kubernetes client.Client and provides GVK-based
// helper functions that return matcher-compatible functions for use with Gomega assertions.
type UnstructuredResources struct {
	client client.Client
}

// NewUnstructuredResources creates a new UnstructuredResources wrapping the provided client.Client.
func NewUnstructuredResources(cli client.Client) *UnstructuredResources {
	return &UnstructuredResources{
		client: cli,
	}
}

// Get returns a function that retrieves a Kubernetes resource by GVK and ObjectKey.
// The returned function is compatible with Gomega's Eventually() and Expect().
//
// For cluster-scoped resources, use Named("resource-name").
//
// For namespaced resources, use Named("resource-name").InNamespace("namespace")
// or NamespacedNamed("namespace", "name").
//
// Example:
//
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//	Eventually(k.Get(podGVK, Named("my-pod").InNamespace("default"))).
//		WithContext(ctx).
//		Should(jq.Match(".status.phase == \"Running\""))
func (m *UnstructuredResources) Get(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	opts ...client.GetOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)

		err := m.client.Get(ctx, key.ToNamespacedName(), obj, opts...)
		if err != nil {
			return nil, err
		}

		return obj, nil
	}
}

// Absent returns a function that reports whether a Kubernetes resource is absent.
// Returns true when the resource is not found OR the resource type has no REST mapping.
// Returns StopTrying for unexpected errors.
func (m *UnstructuredResources) Absent(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return absent(m.Get(gvk, key, opts...))
}

// NotFound returns a function that reports whether a Kubernetes resource is not found.
// Returns true only when the specific object is not found (HTTP 404).
// Returns StopTrying if the resource type has no REST mapping or for other unexpected errors.
func (m *UnstructuredResources) NotFound(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return notFound(m.Get(gvk, key, opts...))
}

// List returns a function that retrieves a list of Kubernetes resources by GVK.
// The returned function is compatible with Gomega's Eventually() and Expect().
//
// Options can include client.InNamespace() to filter by namespace,
// client.MatchingLabels() for label selectors, etc.
//
// Example:
//
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//	Eventually(k.List(podGVK, client.InNamespace("default"))).
//		WithContext(ctx).
//		Should(jq.Match(".items | length > 0"))
func (m *UnstructuredResources) List(
	gvk schema.GroupVersionKind,
	opts ...client.ListOption,
) func(context.Context) (*unstructured.UnstructuredList, error) {
	return func(ctx context.Context) (*unstructured.UnstructuredList, error) {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(gvk)

		err := m.client.List(ctx, list, opts...)
		if err != nil {
			return nil, err
		}

		return list, nil
	}
}

// CreateUnstructured creates an unstructured Kubernetes resource and returns the
// created object for matcher-oriented assertions.
func CreateUnstructured(
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	opts ...client.CreateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		current := obj.DeepCopy()
		gvk := current.GroupVersionKind()

		if err := m.client.Create(ctx, current, opts...); err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}

		return fetchUnstructuredResource(ctx, m, gvk, objectKeyFromUnstructured(current))
	}
}

// Update retrieves a Kubernetes resource, applies an update function, and updates it.
// Returns a function that can be used with Gomega's Eventually for async assertions.
//
// Example:
//
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//	Eventually(k.Update(podGVK, Named("my-pod").InNamespace("default"),
//		func(obj *unstructured.Unstructured) {
//			labels := obj.GetLabels()
//			labels["updated"] = "true"
//			obj.SetLabels(labels)
//		},
//	)).WithContext(ctx).Should(jq.Match(`.metadata.labels.updated == "true"`))
func (m *UnstructuredResources) Update(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	fn func(*unstructured.Unstructured),
	opts ...client.UpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)

		err := m.client.Get(ctx, key.ToNamespacedName(), obj)
		if err != nil {
			return nil, fmt.Errorf("failed to get resource for update: %w", err)
		}

		fn(obj)

		err = m.client.Update(ctx, obj, opts...)
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
}

// StatusUpdate retrieves a Kubernetes resource, applies a status update
// function, and updates its status subresource.
func (m *UnstructuredResources) StatusUpdate(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	fn func(*unstructured.Unstructured),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(key.Name)
		obj.SetNamespace(key.Namespace)

		return applyUnstructuredStatusUpdate(ctx, m, obj, fn, opts...)
	}
}

// UpdateUnstructured retrieves an unstructured resource, applies an update
// function, and returns the updated object.
func UpdateUnstructured(
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	opts ...client.UpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return applyUnstructuredUpdate(ctx, m, obj, fn, opts...)
	}
}

// StatusUpdateUnstructured retrieves an unstructured resource, applies a
// status update function, and returns the updated object.
func StatusUpdateUnstructured(
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return applyUnstructuredStatusUpdate(ctx, m, obj, fn, opts...)
	}
}

// UpsertUnstructured creates an unstructured resource when missing and otherwise
// updates the existing live resource using the provided callback.
func UpsertUnstructured(
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	fn func(*unstructured.Unstructured),
	createOpts ...client.CreateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return applyUnstructuredUpsert(ctx, m, obj, fn, createOpts...)
	}
}

// Delete returns a function that deletes a Kubernetes resource by GVK and ObjectKey.
// The returned function is compatible with Gomega's Eventually() and Expect().
//
// For cluster-scoped resources, use Named("resource-name").
//
// For namespaced resources, use Named("resource-name").InNamespace("namespace")
// or NamespacedNamed("namespace", "name").
//
// Options can include client.GracePeriodSeconds() for deletion grace period, etc.
//
// Example:
//
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//	Expect(k.Delete(podGVK, Named("my-pod").InNamespace("default"))(ctx)).To(Succeed())
func (m *UnstructuredResources) Delete(
	gvk schema.GroupVersionKind,
	key ObjectKey,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(key.Name)
		obj.SetNamespace(key.Namespace)

		return m.client.Delete(ctx, obj, opts...)
	}
}

// DeleteUnstructured deletes an unstructured resource via a top-level helper.
func DeleteUnstructured(
	m *UnstructuredResources,
	obj *unstructured.Unstructured,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		target := obj.DeepCopy()

		return m.client.Delete(ctx, target, opts...)
	}
}
