package k8s

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
