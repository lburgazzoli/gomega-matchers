package k8s

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Data returns a transform function that extracts the full .data field from
// supported Kubernetes objects.
//
// Supported inputs include typed ConfigMaps, typed Secrets, and
// *unstructured.Unstructured objects.
//
// Example:
//
//	WithTransform(k8s.Data(), HaveKeyWithValue("foo", "bar"))
func Data() func(any) (any, error) {
	return func(in any) (any, error) {
		switch obj := in.(type) {
		case *corev1.ConfigMap:
			return obj.Data, nil
		case *corev1.Secret:
			return obj.Data, nil
		case *unstructured.Unstructured:
			return obj.Object["data"], nil
		default:
			return nil, fmt.Errorf("expected *corev1.ConfigMap, *corev1.Secret, or *unstructured.Unstructured, got %T", in)
		}
	}
}
