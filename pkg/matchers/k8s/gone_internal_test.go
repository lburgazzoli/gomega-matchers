package k8s

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"

	. "github.com/onsi/gomega"
)

func TestAbsentNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isAbsent, err := absent(func(context.Context) (struct{}, error) {
		return struct{}{}, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "pods"}, "my-pod")
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isAbsent).To(BeTrue())
}

func TestAbsentNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isAbsent, err := absent(func(context.Context) (struct{}, error) {
		return struct{}{}, &apimeta.NoKindMatchError{
			GroupKind:        schema.GroupKind{Group: "apps", Kind: "Deployment"},
			SearchedVersions: []string{"v1"},
		}
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isAbsent).To(BeTrue())
}

func TestAbsentResourceExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isAbsent, err := absent(func(context.Context) (struct{}, error) {
		return struct{}{}, nil
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isAbsent).To(BeFalse())
}

func TestAbsentUnexpectedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := absent(func(context.Context) (struct{}, error) {
		return struct{}{}, errors.New("connection refused")
	})(t.Context())

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("connection refused"))
}

func TestNotFoundReturnsTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isNotFound, err := notFound(func(context.Context) (struct{}, error) {
		return struct{}{}, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "pods"}, "my-pod")
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isNotFound).To(BeTrue())
}

func TestNotFoundStopTryingOnNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := notFound(func(context.Context) (struct{}, error) {
		return struct{}{}, &apimeta.NoKindMatchError{
			GroupKind:        schema.GroupKind{Group: "apps", Kind: "Deployment"},
			SearchedVersions: []string{"v1"},
		}
	})(t.Context())

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to determine whether resource is not found"))
}

func TestNotFoundResourceExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	isNotFound, err := notFound(func(context.Context) (struct{}, error) {
		return struct{}{}, nil
	})(t.Context())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isNotFound).To(BeFalse())
}

func TestNotFoundUnexpectedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := notFound(func(context.Context) (struct{}, error) {
		return struct{}{}, errors.New("connection refused")
	})(t.Context())

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("connection refused"))
}
