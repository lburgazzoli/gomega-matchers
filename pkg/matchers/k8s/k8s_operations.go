package k8s

import (
	"context"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

// Get retrieves a Kubernetes resource and returns it as the same concrete type.
// The returned function is compatible with Gomega's Eventually() and Expect().
func Get[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return fetchObject(ctx, cli, obj, opts...)
	}
}

// Lookup retrieves a Kubernetes resource into the passed object and returns
// only an error. The returned function is compatible with Gomega's Eventually()
// and Expect().
func Lookup[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return lookupObject(ctx, cli, obj, opts...)
	}
}

// Create creates a Kubernetes resource and returns the created object.
func Create[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.CreateOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return doCreate(ctx, cli, obj, opts...)
	}
}

// Delete deletes a Kubernetes resource.
func Delete[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return func(ctx context.Context) error {
		return doDelete(ctx, cli, obj, opts...)
	}
}

// Update retrieves a Kubernetes resource, applies an update function, and
// updates it. The callback can be either a typed function receiving the
// concrete object type or a reusable client.Object mutator.
func Update[T client.Object, F objectMutator[T]](
	cli client.Client,
	obj T,
	fn F,
	opts ...client.UpdateOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return doUpdate(ctx, cli, obj, adaptMutator[T](fn), opts...)
	}
}

// StatusUpdate retrieves a Kubernetes resource, applies a typed status
// update function, and updates its status subresource.
func StatusUpdate[T client.Object](
	cli client.Client,
	obj T,
	fn func(T),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return doStatusUpdate(ctx, cli, obj, fn, opts...)
	}
}

// Upsert creates a Kubernetes resource when it does not exist and otherwise
// updates the existing live resource using the provided callback. The callback
// can be either a typed function or a reusable client.Object mutator.
func Upsert[T client.Object, F objectMutator[T]](
	cli client.Client,
	obj T,
	fn F,
	createOpts ...client.CreateOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		return doUpsert(ctx, cli, obj, adaptMutator[T](fn), createOpts...)
	}
}

// Absent returns a function that reports whether a Kubernetes resource is absent.
// Returns true when the resource is not found OR the resource type has no REST mapping.
// Returns StopTrying for unexpected errors.
func Absent[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return isAbsent(Get(cli, obj, opts...))
}

// NotFound returns a function that reports whether a Kubernetes resource is not found.
// Returns true only when the specific object is not found (HTTP 404).
// Returns StopTrying if the resource type has no REST mapping or for other unexpected errors.
func NotFound[T client.Object](
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return isNotFound(Get(cli, obj, opts...))
}

// List retrieves a list of Kubernetes resources.
// The returned function is compatible with Gomega's Eventually() and Expect().
func List[T client.ObjectList](
	cli client.Client,
	list T,
	opts ...client.ListOption,
) func(context.Context) (T, error) {
	return func(ctx context.Context) (T, error) {
		result, ok := list.DeepCopyObject().(T)
		if !ok {
			return zero[T](), gomega.StopTrying("failed to deep copy list object")
		}

		if err := cli.List(ctx, result, opts...); err != nil {
			return zero[T](), err
		}

		return result, nil
	}
}

// Events lists typed Kubernetes events and returns them as a plain slice so
// callers can use standard Gomega collection matchers directly.
func Events(
	cli client.Client,
	opts ...EventOption,
) func(context.Context) ([]corev1.Event, error) {
	return func(ctx context.Context) ([]corev1.Event, error) {
		resolved := resolveEventOptions(opts...)
		events := &corev1.EventList{}

		if err := cli.List(ctx, events, resolved.listOptions...); err != nil {
			return nil, err
		}

		return resolved.filter(events.Items), nil
	}
}
