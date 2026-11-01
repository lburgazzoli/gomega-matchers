package k8s

import (
	"errors"
	"fmt"
	"maps"
	"reflect"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HasName matches a Kubernetes object by metadata.name.
func HasName(name string) types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (string, error) {
			obj, err := asObject(actual)
			if err != nil {
				return "", err
			}

			return obj.GetName(), nil
		},
		gomega.Equal(name),
	)
}

// HasNamespace matches a Kubernetes object by metadata.namespace.
func HasNamespace(namespace string) types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (string, error) {
			obj, err := asObject(actual)
			if err != nil {
				return "", err
			}

			return obj.GetNamespace(), nil
		},
		gomega.Equal(namespace),
	)
}

// HasLabel matches a Kubernetes object by metadata.labels[key].
func HasLabel(key string, value string) types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (map[string]string, error) {
			obj, err := asObject(actual)
			if err != nil {
				return nil, err
			}

			return obj.GetLabels(), nil
		},
		gomega.HaveKeyWithValue(key, value),
	)
}

// HasAnnotation matches a Kubernetes object by metadata.annotations[key].
func HasAnnotation(key string, value string) types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (map[string]string, error) {
			obj, err := asObject(actual)
			if err != nil {
				return nil, err
			}

			return obj.GetAnnotations(), nil
		},
		gomega.HaveKeyWithValue(key, value),
	)
}

// HasFinalizer matches a Kubernetes object containing the given finalizer.
func HasFinalizer(finalizer string) types.GomegaMatcher {
	return gomega.WithTransform(
		Finalizers(),
		gomega.ContainElement(finalizer),
	)
}

// IsDeleting matches a Kubernetes object with a non-zero deletion timestamp.
func IsDeleting() types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (bool, error) {
			obj, err := asObject(actual)
			if err != nil {
				return false, err
			}

			return !obj.GetDeletionTimestamp().IsZero(), nil
		},
		gomega.BeTrue(),
	)
}

// HasOwnerReference matches a Kubernetes object containing an owner reference
// matching the given owner's API group, Kind and Name (and UID when set).
// The owner must have APIVersion, Kind and Name set. API versions within the
// same group are interchangeable.
func HasOwnerReference(owner client.Object) types.GomegaMatcher {
	return ownerRefTransform(owner, gstruct.Fields{})
}

// IsControlledBy matches a Kubernetes object that has a controller owner reference
// (Controller: true) matching the owner's API group, Kind and Name (and UID
// when set). The owner must have APIVersion, Kind and Name set.
func IsControlledBy(owner client.Object) types.GomegaMatcher {
	return ownerRefTransform(owner, gstruct.Fields{"Controller": gomega.HaveValue(gomega.BeTrue())})
}

func ownerRefTransform(owner client.Object, additionalFields gstruct.Fields) types.GomegaMatcher {
	fields, ownerErr := ownerRefFields(owner)
	maps.Copy(fields, additionalFields)

	return gomega.WithTransform(
		func(actual any) ([]metav1.OwnerReference, error) {
			if ownerErr != nil {
				return nil, ownerErr
			}

			obj, err := asObject(actual)
			if err != nil {
				return nil, err
			}

			return obj.GetOwnerReferences(), nil
		},
		gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, fields)),
	)
}

func ownerRefFields(owner client.Object) (gstruct.Fields, error) {
	fields := gstruct.Fields{}
	if _, err := asObject(owner); err != nil {
		return fields, fmt.Errorf("invalid owner: %w", err)
	}

	gvk := owner.GetObjectKind().GroupVersionKind()
	switch {
	case gvk.Kind == "":
		return fields, errors.New("owner has empty Kind; set TypeMeta on the owner object")
	case gvk.Version == "":
		return fields, errors.New("owner has empty APIVersion; set TypeMeta on the owner object")
	case owner.GetName() == "":
		return fields, errors.New("owner has empty Name")
	}

	gv, err := schema.ParseGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return fields, fmt.Errorf("invalid owner APIVersion: %w", err)
	}

	fields["Kind"] = gomega.Equal(gvk.Kind)
	fields["Name"] = gomega.Equal(owner.GetName())
	fields["APIVersion"] = gomega.Satisfy(func(version string) bool {
		actual, parseErr := schema.ParseGroupVersion(version)

		return parseErr == nil && actual.Version != "" && actual.Group == gv.Group
	})

	if owner.GetUID() != "" {
		fields["UID"] = gomega.Equal(owner.GetUID())
	}

	return fields, nil
}

// MatchesGroupVersion matches a Kubernetes object by group and version.
// This reads the GVK from the object's TypeMeta, which is populated for
// unstructured objects and real apiserver responses but typically empty
// for typed objects returned by the fake client.
func MatchesGroupVersion(groupVersion schema.GroupVersion) types.GomegaMatcher {
	return gomega.WithTransform(
		func(actual any) (schema.GroupVersion, error) {
			gvk, err := objectGVK(actual)
			if err != nil {
				return schema.GroupVersion{}, err
			}

			return gvk.GroupVersion(), nil
		},
		gomega.Equal(groupVersion),
	)
}

// MatchesGroupVersionKind matches a Kubernetes object by full GroupVersionKind.
// This reads the GVK from the object's TypeMeta, which is populated for
// unstructured objects and real apiserver responses but typically empty
// for typed objects returned by the fake client.
func MatchesGroupVersionKind(gvk schema.GroupVersionKind) types.GomegaMatcher {
	return gomega.WithTransform(
		objectGVK,
		gomega.Equal(gvk),
	)
}

// IsEmptyList matches a Kubernetes list object whose Items slice is empty.
func IsEmptyList() types.GomegaMatcher {
	return gomega.WithTransform(
		ListItems(),
		gomega.BeEmpty(),
	)
}

func asObject(actual any) (client.Object, error) {
	obj, ok := actual.(client.Object)
	if !ok {
		return nil, fmt.Errorf("expected client.Object, got %T", actual)
	}
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil, fmt.Errorf("expected non-nil client.Object, got %T", actual)
	}

	return obj, nil
}

func objectGVK(actual any) (schema.GroupVersionKind, error) {
	obj, err := asObject(actual)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}

	return obj.GetObjectKind().GroupVersionKind(), nil
}
