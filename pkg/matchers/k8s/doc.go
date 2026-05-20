// Package k8s provides Kubernetes test helpers that integrate seamlessly with Gomega matchers.
//
// The package wraps client.Client operations to return matcher functions compatible with
// Gomega's Eventually() and Expect(), making it easy to write tests that combine
// Kubernetes resource operations with JQ-based assertions.
//
// Example usage:
//
//	import (
//		. "github.com/onsi/gomega"
//		"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
//		"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
//		"k8s.io/apimachinery/pkg/runtime/schema"
//		"sigs.k8s.io/controller-runtime/pkg/client"
//	)
//
//	k := k8s.NewUnstructured(client)
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//
//	// Wait for pod to be ready
//	Eventually(k.Get(podGVK, k8s.Named("my-pod").InNamespace("default"))).
//		WithContext(ctx).
//		Should(jq.Match(`.status.phase == "Running"`))
//
//	// List all pods in namespace
//	Eventually(k.List(podGVK, client.InNamespace("default"))).
//		WithContext(ctx).
//		Should(jq.Match(`. | length > 0`))
package k8s
