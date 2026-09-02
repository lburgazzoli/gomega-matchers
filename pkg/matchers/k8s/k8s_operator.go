package k8s

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
)

// Operations is a client-bound facade for the package-level Kubernetes
// operations. Its methods use client.Object and client.ObjectList because Go
// does not support type parameters on methods; use the package-level functions
// when a concrete result type is required.
type Operations struct {
	cli client.Client
}

// Using returns a client-bound facade for Kubernetes operations.
func Using(cli client.Client) *Operations {
	return &Operations{cli: cli}
}

// Get retrieves a Kubernetes resource.
func (o *Operations) Get(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (client.Object, error) {
	return Get(o.cli, obj, opts...)
}

// Lookup retrieves a Kubernetes resource into the passed object.
func (o *Operations) Lookup(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) error {
	return Lookup(o.cli, obj, opts...)
}

// Singleton lists resources and returns the single matching object. The
// prototype supplies the concrete object type used by the underlying list.
func (o *Operations) Singleton(
	prototype client.Object,
	opts ...client.ListOption,
) func(context.Context) (client.Object, error) {
	return func(ctx context.Context) (client.Object, error) {
		return singletonObject(ctx, o.cli, prototype, opts...)
	}
}

// LookupSingleton lists resources and writes the single matching object into
// the passed output object.
func (o *Operations) LookupSingleton(
	obj client.Object,
	opts ...client.ListOption,
) func(context.Context) error {
	return LookupSingleton(o.cli, obj, opts...)
}

// Create creates a Kubernetes resource and returns the created object.
func (o *Operations) Create(
	obj client.Object,
	opts ...client.CreateOption,
) func(context.Context) (client.Object, error) {
	return Create(o.cli, obj, opts...)
}

// Delete deletes a Kubernetes resource.
func (o *Operations) Delete(
	obj client.Object,
	opts ...client.DeleteOption,
) func(context.Context) error {
	return Delete(o.cli, obj, opts...)
}

// Update retrieves a resource, applies a mutator, and updates it.
func (o *Operations) Update(
	obj client.Object,
	fn func(client.Object),
	opts ...client.UpdateOption,
) func(context.Context) (client.Object, error) {
	return Update(o.cli, obj, fn, opts...)
}

// StatusUpdate retrieves a resource, applies a status mutator, and updates its
// status subresource.
func (o *Operations) StatusUpdate(
	obj client.Object,
	fn func(client.Object),
	opts ...client.SubResourceUpdateOption,
) func(context.Context) (client.Object, error) {
	return StatusUpdate(o.cli, obj, fn, opts...)
}

// Upsert creates a resource when it does not exist and otherwise updates it.
func (o *Operations) Upsert(
	obj client.Object,
	fn func(client.Object),
	createOpts ...client.CreateOption,
) func(context.Context) (client.Object, error) {
	return Upsert(o.cli, obj, fn, createOpts...)
}

// Absent reports whether a Kubernetes resource is absent.
func (o *Operations) Absent(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return Absent(o.cli, obj, opts...)
}

// NotFound reports whether a Kubernetes resource is not found.
func (o *Operations) NotFound(
	obj client.Object,
	opts ...client.GetOption,
) func(context.Context) (bool, error) {
	return NotFound(o.cli, obj, opts...)
}

// List retrieves a list of Kubernetes resources.
func (o *Operations) List(
	list client.ObjectList,
	opts ...client.ListOption,
) func(context.Context) (client.ObjectList, error) {
	return List(o.cli, list, opts...)
}

// Events lists events.k8s.io/v1 Kubernetes events.
func (o *Operations) Events(
	opts ...EventOption,
) func(context.Context) ([]eventsv1.Event, error) {
	return Events(o.cli, opts...)
}

// CoreEvents lists legacy core/v1 Kubernetes events.
func (o *Operations) CoreEvents(
	opts ...EventOption,
) func(context.Context) ([]corev1.Event, error) {
	return CoreEvents(o.cli, opts...)
}
