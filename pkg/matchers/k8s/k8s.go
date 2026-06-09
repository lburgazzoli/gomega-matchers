package k8s

import (
	"context"

	"github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
)

func isAbsent[T any](get func(context.Context) (T, error)) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		_, err := get(ctx)
		if err == nil {
			return false, nil
		}

		if apierrors.IsNotFound(err) || apimeta.IsNoMatchError(err) {
			return true, nil
		}

		return false, gomega.StopTrying("failed to determine whether resource is absent").Wrap(err)
	}
}

func isNotFound[T any](get func(context.Context) (T, error)) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		_, err := get(ctx)
		if err == nil {
			return false, nil
		}

		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, gomega.StopTrying("failed to determine whether resource is not found").Wrap(err)
	}
}
