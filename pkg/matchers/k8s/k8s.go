package k8s

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Matcher wraps a Kubernetes client.Client and provides helper functions
// that return matcher-compatible functions for use with Gomega assertions.
type Matcher struct {
	client client.Client
}

// New creates a new Matcher wrapping the provided client.Client.
func New(cli client.Client) *Matcher {
	return &Matcher{
		client: cli,
	}
}

// ObjectKey is an alias for types.NamespacedName with additional helper methods.
type ObjectKey types.NamespacedName

// Named creates an ObjectKey with just a name (for cluster-scoped resources).
func Named(name string) ObjectKey {
	return ObjectKey{Name: name}
}

// NamespacedNamed creates an ObjectKey with both namespace and name.
func NamespacedNamed(namespace string, name string) ObjectKey {
	return ObjectKey{Namespace: namespace, Name: name}
}

// InNamespace sets the namespace for the ObjectKey, enabling fluent API like Named("foo").InNamespace("bar").
func (k ObjectKey) InNamespace(namespace string) ObjectKey {
	k.Namespace = namespace
	return k
}

// ToNamespacedName converts ObjectKey back to types.NamespacedName.
func (k ObjectKey) ToNamespacedName() types.NamespacedName {
	return types.NamespacedName(k)
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
func (m *Matcher) Get(
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
func (m *Matcher) List(
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
func (m *Matcher) Delete(
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
