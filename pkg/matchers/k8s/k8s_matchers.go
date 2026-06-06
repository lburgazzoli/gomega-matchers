package k8s

import (
	"errors"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// HasName matches a Kubernetes object by metadata.name.
func HasName(name string) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) (string, error) {
		obj, err := asObject(actual)
		if err != nil {
			return "", err
		}

		return obj.GetName(), nil
	}, gomega.Equal(name))
}

// HasNamespace matches a Kubernetes object by metadata.namespace.
func HasNamespace(namespace string) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) (string, error) {
		obj, err := asObject(actual)
		if err != nil {
			return "", err
		}

		return obj.GetNamespace(), nil
	}, gomega.Equal(namespace))
}

// HasLabel matches a Kubernetes object by metadata.labels[key].
func HasLabel(key string, value string) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) (map[string]string, error) {
		obj, err := asObject(actual)
		if err != nil {
			return nil, err
		}

		return obj.GetLabels(), nil
	}, gomega.HaveKeyWithValue(key, value))
}

// HasAnnotation matches a Kubernetes object by metadata.annotations[key].
func HasAnnotation(key string, value string) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) (map[string]string, error) {
		obj, err := asObject(actual)
		if err != nil {
			return nil, err
		}

		return obj.GetAnnotations(), nil
	}, gomega.HaveKeyWithValue(key, value))
}

// HasOwnerReference matches a Kubernetes object containing an owner reference
// matching the given owner's Kind and Name (and UID when set on the owner).
func HasOwnerReference(owner client.Object) types.GomegaMatcher {
	return ownerRefTransform(owner, ownerRefFields(owner))
}

// IsControlledBy matches a Kubernetes object that has a controller owner reference
// (Controller: true) matching the given owner's Kind and Name (and UID when set).
func IsControlledBy(owner client.Object) types.GomegaMatcher {
	fields := ownerRefFields(owner)
	fields["Controller"] = gomega.HaveValue(gomega.BeTrue())

	return ownerRefTransform(owner, fields)
}

func ownerRefTransform(owner client.Object, fields gstruct.Fields) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) ([]metav1.OwnerReference, error) {
		if owner.GetObjectKind().GroupVersionKind().Kind == "" {
			return nil, errors.New("owner has empty Kind; set TypeMeta on the owner object")
		}

		obj, err := asObject(actual)
		if err != nil {
			return nil, err
		}

		return obj.GetOwnerReferences(), nil
	}, gomega.ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, fields)))
}

func ownerRefFields(owner client.Object) gstruct.Fields {
	fields := gstruct.Fields{
		"Kind": gomega.Equal(owner.GetObjectKind().GroupVersionKind().Kind),
		"Name": gomega.Equal(owner.GetName()),
	}

	if owner.GetUID() != "" {
		fields["UID"] = gomega.Equal(owner.GetUID())
	}

	return fields
}

// MatchesGroupVersion matches a Kubernetes object by group and version.
func MatchesGroupVersion(groupVersion schema.GroupVersion) types.GomegaMatcher {
	return gomega.WithTransform(func(actual any) (schema.GroupVersion, error) {
		gvk, err := objectGVK(actual)
		if err != nil {
			return schema.GroupVersion{}, err
		}

		return gvk.GroupVersion(), nil
	}, gomega.Equal(groupVersion))
}

// MatchesGroupVersionKind matches a Kubernetes object by full GroupVersionKind.
func MatchesGroupVersionKind(gvk schema.GroupVersionKind) types.GomegaMatcher {
	return gomega.WithTransform(objectGVK, gomega.Equal(gvk))
}

func asObject(actual any) (client.Object, error) {
	obj, ok := actual.(client.Object)
	if !ok {
		return nil, fmt.Errorf("expected client.Object, got %T", actual)
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
