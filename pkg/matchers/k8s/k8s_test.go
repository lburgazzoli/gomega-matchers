package k8s_test

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	podGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "Pod",
	}

	namespaceGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "Namespace",
	}
)
