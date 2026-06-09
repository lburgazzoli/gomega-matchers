package k8s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Resources wraps a Kubernetes client.Client and provides typed helper functions
// that work with typed Kubernetes objects (e.g., *corev1.ConfigMap) and return
// unstructured results compatible with Gomega assertions and JQ matchers.
type Resources struct {
	client client.Client
	scheme *runtime.Scheme
}

// NewResources creates a new typed Resources wrapping the provided client.Client.
// The scheme is used to extract GVK information from typed objects.
func NewResources(cli client.Client, scheme *runtime.Scheme) *Resources {
	return &Resources{
		client: cli,
		scheme: scheme,
	}
}

// Get retrieves a Kubernetes resource using a typed object.
// The GVK is automatically extracted from the object type using the scheme.
// Returns a function that can be used with Gomega's Eventually for async assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	Eventually(k.Get(&corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "value"`))
func (m *Resources) Get(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
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
}

// Absent returns a function that reports whether a typed Kubernetes resource is absent.
// Returns true when the resource is not found OR the resource type has no REST mapping.
// Returns StopTrying for unexpected errors.
func (m *Resources) Absent(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return absent(m.Get(obj, opts...))
}

// NotFound returns a function that reports whether a typed Kubernetes resource is not found.
// Returns true only when the specific object is not found (HTTP 404).
// Returns StopTrying if the resource type has no REST mapping or for other unexpected errors.
func (m *Resources) NotFound(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return notFound(m.Get(obj, opts...))
}

// List retrieves a list of Kubernetes resources using a typed list object.
// The GVK is automatically extracted from the list type using the scheme.
// Returns a function that can be used with Gomega's Eventually for async assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	Eventually(k.List(&corev1.ConfigMapList{},
//		client.InNamespace("default"),
//		client.MatchingLabels{"app": "myapp"},
//	)).WithContext(ctx).Should(jq.Match(`.items | length > 0`))
func (m *Resources) List(
	list client.ObjectList,
	opts ...client.ListOption,
) func(context.Context) (*unstructured.UnstructuredList, error) {
	return func(ctx context.Context) (*unstructured.UnstructuredList, error) {
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
}

// Delete deletes a Kubernetes resource using a typed object.
// The GVK and object key are automatically extracted from the object.
// For new typed code, prefer the package-level k8s.Delete helper for consistency
// with k8s.Create, k8s.Update, and k8s.Upsert.
// Returns a function that can be used with Gomega's Eventually for async assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	Eventually(k.Delete(&corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).WithContext(ctx).Should(Succeed())
func (m *Resources) Delete(
	obj client.Object,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return deleteTyped(ctx, m, obj, opts...)
	}
}

// Create creates a typed Kubernetes resource and returns the created object as
// unstructured data so it can be used with matcher-oriented assertions.
//
// Example:
//
//	k := k8s.NewResources(client, scheme)
//	Eventually(k8s.Create(k, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//		Data: map[string]string{
//			"key": "value",
//		},
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "value"`))
func Create[T client.Object](
	m *Resources,
	obj T,
	opts ...client.CreateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return createTyped(ctx, m, obj, opts...)
	}
}

// Delete deletes a typed Kubernetes resource via the package-level generic API.
//
// Example:
//
//	k := k8s.NewResources(client, scheme)
//	Eventually(k8s.Delete(k, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).WithContext(ctx).Should(Succeed())
func Delete[T client.Object](
	m *Resources,
	obj T,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return deleteTyped(ctx, m, obj, opts...)
	}
}

// Update retrieves a typed Kubernetes resource, applies a typed update function,
// and updates it.
//
// This package-level helper uses generics so the update callback receives the
// concrete object type and does not need a cast.
//
// Example:
//
//	k := k8s.NewResources(client, scheme)
//	Eventually(k8s.Update(k, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		cm.Data["key"] = "new-value"
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "new-value"`))
func Update[T client.Object](
	m *Resources,
	obj T,
	fn func(T),
	opts ...client.UpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return updateTyped(ctx, m, obj, fn, opts...)
	}
}

// StatusUpdate retrieves a typed Kubernetes resource, applies a typed status
// update function, and updates its status subresource.
func StatusUpdate[T client.Object](
	m *Resources,
	obj T,
	fn func(T),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return statusUpdateTyped(ctx, m, obj, fn, opts...)
	}
}

// Upsert creates a typed Kubernetes resource when it does not exist and
// otherwise updates the existing live resource using the provided typed callback.
//
// Example:
//
//	k := k8s.NewResources(client, scheme)
//	Eventually(k8s.Upsert(k, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		if cm.Data == nil {
//			cm.Data = map[string]string{}
//		}
//		cm.Data["key"] = "value"
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "value"`))
func Upsert[T client.Object](
	m *Resources,
	obj T,
	fn func(T),
	createOpts ...client.CreateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return upsertTyped(ctx, m, obj, fn, createOpts...)
	}
}

// Update retrieves a Kubernetes resource, applies an update function, and updates it.
// This method preserves the original Komega-style callback API.
//
// For new typed code, prefer the package-level k8s.Update helper so the callback
// receives the concrete object type without a cast.
// Returns a function that can be used with Gomega's Eventually for async assertions.
//
// Example:
//
//	k := k8s.New(client, scheme)
//	Eventually(k.Update(&corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(obj client.Object) {
//		cm := obj.(*corev1.ConfigMap)
//		cm.Data["key"] = "new-value"
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "new-value"`))
func (m *Resources) Update(
	obj client.Object,
	fn func(client.Object),
	opts ...client.UpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return updateTyped(ctx, m, obj, fn, opts...)
	}
}

// StatusUpdate retrieves a Kubernetes resource, applies a status update
// function, and updates its status subresource.
func (m *Resources) StatusUpdate(
	obj client.Object,
	fn func(client.Object),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (*unstructured.Unstructured, error) {
	return func(ctx context.Context) (*unstructured.Unstructured, error) {
		return statusUpdateTyped(ctx, m, obj, fn, opts...)
	}
}

// extractGVKAndKey extracts GroupVersionKind and ObjectKey from a typed Kubernetes object.
func (m *Resources) extractGVKAndKey(obj client.Object) (schema.GroupVersionKind, ObjectKey, error) {
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
