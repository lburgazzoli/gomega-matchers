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
//	k := k8s.NewUnstructuredResources(client)
//	podGVK := schema.GroupVersionKind{Version: "v1", Kind: "Pod"}
//
//	// Wait for pod to be ready
//	Eventually(k.Get(podGVK, k8s.Named("my-pod").InNamespace("default"))).
//		WithContext(ctx).
//		Should(jq.Match(`.status.phase == "Running"`))
//
//	// Wait for pod to be deleted (Absent tolerates missing CRD; NotFound requires the type to exist)
//	Eventually(k.Absent(podGVK, k8s.Named("my-pod").InNamespace("default"))).
//		WithContext(ctx).
//		Should(BeTrue())
//
//	// Inspect list items with standard Gomega matchers
//	Eventually(k.List(podGVK, client.InNamespace("default"))).
//		WithContext(ctx).
//		Should(WithTransform(k8s.ListItems(), HaveLen(2)))
//
//	// Assert a list is empty
//	Eventually(k.List(podGVK, client.InNamespace("default"))).
//		WithContext(ctx).
//		Should(k8s.IsEmptyList())
//
//	// Create an unstructured object via the distinct top-level helper.
//	Eventually(k8s.CreateUnstructured(k, &unstructured.Unstructured{
//		Object: map[string]any{
//			"apiVersion": "v1",
//			"kind":       "ConfigMap",
//			"metadata": map[string]any{
//				"name":      "my-config",
//				"namespace": "default",
//			},
//			"data": map[string]any{
//				"key": "value",
//			},
//		},
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "value"`))
//
//	// Wait for an event for a specific object
//	typed := k8s.NewResources(client, scheme)
//	Eventually(typed.Events(
//		k8s.InNamespace("default"),
//		k8s.ForObject(corev1.ObjectReference{
//			Kind: "Pod",
//			Name: "my-pod",
//		}),
//	)).WithContext(ctx).Should(ContainElement(
//		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
//			"Reason": Equal("Ready"),
//		}),
//	))
//
//	// Create a typed object and assert on the created resource.
//	Eventually(k8s.Create(typed, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//		Data: map[string]string{
//			"key": "initial-value",
//		},
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "initial-value"`))
//
//	// Apply a typed update without casting inside the callback.
//	Eventually(k8s.Update(typed, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		cm.Data["key"] = "new-value"
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "new-value"`))
//
//	// Create or update using the same typed callback.
//	Eventually(k8s.Upsert(typed, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		if cm.Data == nil {
//			cm.Data = map[string]string{}
//		}
//		cm.Data["key"] = "reconciled-value"
//	})).WithContext(ctx).Should(jq.Match(`.data.key == "reconciled-value"`))
//
//	// Delete through the package-level typed helper.
//	Eventually(k8s.Delete(typed, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).WithContext(ctx).Should(Succeed())
package k8s
