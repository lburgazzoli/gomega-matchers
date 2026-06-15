package k8s

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func zero[T any]() T {
	var v T

	return v
}

func cloneObject[T client.Object](obj T) (T, error) {
	result, ok := obj.DeepCopyObject().(T)
	if !ok {
		return zero[T](), fmt.Errorf("failed to deep copy %T", obj)
	}

	return result, nil
}

func fetchObject[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) (T, error) {
	result, err := cloneObject(obj)
	if err != nil {
		return zero[T](), err
	}

	if err := cli.Get(ctx, client.ObjectKeyFromObject(obj), result, opts...); err != nil {
		return zero[T](), err
	}

	return result, nil
}

func lookupObject[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	opts ...client.GetOption,
) error {
	return cli.Get(ctx, client.ObjectKeyFromObject(obj), obj, opts...)
}

func newObject[T client.Object]() (T, error) {
	return newObjectLike(zero[T]())
}

func newObjectLike[T client.Object](obj T) (T, error) {
	typ := reflect.TypeOf(obj)
	if typ == nil || typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return zero[T](), fmt.Errorf("expected non-nil pointer object type, got %T", obj)
	}

	result, ok := reflect.New(typ.Elem()).Interface().(T)
	if !ok {
		return zero[T](), fmt.Errorf("failed to instantiate %T", obj)
	}

	return result, nil
}

func copyObjectInto[T client.Object](dst T, src T) error {
	dstValue := reflect.ValueOf(dst)
	srcValue := reflect.ValueOf(src)
	if !dstValue.IsValid() || dstValue.Kind() != reflect.Pointer || dstValue.IsNil() {
		return fmt.Errorf("expected non-nil destination object, got %T", dst)
	}
	if !srcValue.IsValid() || srcValue.Kind() != reflect.Pointer || srcValue.IsNil() {
		return fmt.Errorf("expected non-nil source object, got %T", src)
	}

	dstValue.Elem().Set(srcValue.Elem())

	return nil
}

func singletonObject[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	opts ...client.ListOption,
) (T, error) {
	list, gvk, err := listObjectFor(obj, cli.Scheme())
	if err != nil {
		return zero[T](), gomega.StopTrying("failed to resolve list object for singleton lookup").Wrap(err)
	}

	if err := cli.List(ctx, list, opts...); err != nil {
		return zero[T](), err
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return zero[T](), gomega.StopTrying("failed to extract singleton list items").Wrap(err)
	}

	if len(items) == 0 {
		return zero[T](), singletonNotFound(gvk)
	}

	if len(items) > 1 {
		return zero[T](), gomega.StopTrying(
			fmt.Sprintf("expected exactly one matching resource, found %d", len(items)),
		)
	}

	return convertObject[T](items[0], obj)
}

func listObjectFor(obj client.Object, scheme *runtime.Scheme) (client.ObjectList, schema.GroupVersionKind, error) {
	gvk, err := objectGVKForList(obj, scheme)
	if err != nil {
		return nil, schema.GroupVersionKind{}, err
	}

	listGVK := gvk.GroupVersion().WithKind(gvk.Kind + "List")
	if _, ok := obj.(*unstructured.Unstructured); ok {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(listGVK)

		return list, gvk, nil
	}

	listObj, err := scheme.New(listGVK)
	if err == nil {
		list, ok := listObj.(client.ObjectList)
		if !ok {
			return nil, schema.GroupVersionKind{}, fmt.Errorf(
				"expected %s to implement client.ObjectList, got %T",
				listGVK,
				listObj,
			)
		}

		return list, gvk, nil
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(listGVK)

	return list, gvk, nil
}

func objectGVKForList(obj client.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		gvk := u.GroupVersionKind()
		if gvk.Empty() {
			return schema.GroupVersionKind{}, errors.New("unstructured object is missing GroupVersionKind")
		}

		return gvk, nil
	}

	return apiutil.GVKForObject(obj, scheme)
}

func convertObject[T client.Object](item runtime.Object, prototype T) (T, error) {
	if typed, ok := item.(T); ok {
		return typed, nil
	}

	result, err := newObjectLike(prototype)
	if err != nil {
		return zero[T](), gomega.StopTrying("failed to allocate singleton result object").Wrap(err)
	}

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
	if err != nil {
		return zero[T](), gomega.StopTrying("failed to convert singleton result").Wrap(err)
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(data, result); err != nil {
		return zero[T](), gomega.StopTrying("failed to decode singleton result").Wrap(err)
	}

	if current, ok := item.(client.Object); ok {
		result.GetObjectKind().SetGroupVersionKind(current.GetObjectKind().GroupVersionKind())
	}

	return result, nil
}

func singletonNotFound(gvk schema.GroupVersionKind) error {
	resource, _ := meta.UnsafeGuessKindToResource(gvk)

	return apierrors.NewNotFound(resource.GroupResource(), gvk.Kind+" matching list options")
}

// Write operations (create, update, upsert) re-fetch the object after the
// write so the returned value reflects server-side defaults and mutations
// (e.g. resourceVersion, generation, defaulted fields). This costs one
// extra GET per write — acceptable for a test helper.

func doCreate[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	opts ...client.CreateOption,
) (T, error) {
	current, err := cloneObject(obj)
	if err != nil {
		return zero[T](), err
	}

	if err := cli.Create(ctx, current, opts...); err != nil {
		return zero[T](), fmt.Errorf("failed to create resource: %w", err)
	}

	return fetchObject(ctx, cli, current)
}

func doDelete[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	opts ...client.DeleteOption,
) error {
	current, err := cloneObject(obj)
	if err != nil {
		return err
	}

	return cli.Delete(ctx, current, opts...)
}

func doUpdate[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	fn func(T),
	opts ...client.UpdateOption,
) (T, error) {
	current, err := fetchObject(ctx, cli, obj)
	if err != nil {
		return zero[T](), fmt.Errorf("failed to get resource for update: %w", err)
	}

	fn(current)

	if err := cli.Update(ctx, current, opts...); err != nil {
		return zero[T](), fmt.Errorf("failed to update resource: %w", err)
	}

	return fetchObject(ctx, cli, current)
}

func doStatusUpdate[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	fn func(T),
	opts ...client.SubResourceUpdateOption,
) (T, error) {
	current, err := fetchObject(ctx, cli, obj)
	if err != nil {
		return zero[T](), fmt.Errorf("failed to get resource for status update: %w", err)
	}

	fn(current)

	if err := cli.Status().Update(ctx, current, opts...); err != nil {
		return zero[T](), fmt.Errorf("failed to update resource status: %w", err)
	}

	return fetchObject(ctx, cli, current)
}

func doUpsert[T client.Object](
	ctx context.Context,
	cli client.Client,
	obj T,
	fn func(T),
	createOpts ...client.CreateOption,
) (T, error) {
	current, err := fetchObject(ctx, cli, obj)

	if err == nil {
		fn(current)

		if err := cli.Update(ctx, current); err != nil {
			return zero[T](), fmt.Errorf("failed to update resource: %w", err)
		}

		return fetchObject(ctx, cli, current)
	}

	if !apierrors.IsNotFound(err) {
		return zero[T](), fmt.Errorf("failed to get resource for upsert: %w", err)
	}

	created, err := cloneObject(obj)
	if err != nil {
		return zero[T](), err
	}

	fn(created)

	if err := cli.Create(ctx, created, createOpts...); err != nil {
		return zero[T](), fmt.Errorf("failed to create resource: %w", err)
	}

	return fetchObject(ctx, cli, created)
}
