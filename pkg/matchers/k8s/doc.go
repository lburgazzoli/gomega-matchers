// Package k8s provides Kubernetes test helpers that integrate seamlessly with Gomega matchers.
//
// All operations are package-level generic functions that take a client.Client directly.
// Both typed objects (e.g., *corev1.ConfigMap) and unstructured objects
// (*unstructured.Unstructured) work with the same functions — the type parameter is inferred.
//
// Example usage:
//
//	import (
//		. "github.com/onsi/gomega"
//		"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
//		"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
//		"sigs.k8s.io/controller-runtime/pkg/client"
//	)
//
//	cli := /* client.Client */
//
//	// Get a typed object — JQ matchers work transparently
//	Eventually(ctx, k8s.Get(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).Should(jq.Match(`.data.key == "value"`))
//
//	// Get an unstructured object — same function
//	pod := &unstructured.Unstructured{}
//	pod.SetGroupVersionKind(schema.GroupVersionKind{Version: "v1", Kind: "Pod"})
//	pod.SetName("my-pod")
//	pod.SetNamespace("default")
//	Eventually(ctx, k8s.Get(cli, pod)).
//		Should(jq.Match(`.status.phase == "Running"`))
//
//	// Load into an existing object and assert only on the lookup error
//	cm := &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}
//	Eventually(ctx, k8s.Lookup(cli, cm)).Should(Succeed())
//	Expect(cm.Data).To(HaveKeyWithValue("key", "value"))
//
//	// Wait for a resource to be deleted
//	Eventually(ctx, k8s.Absent(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).Should(BeTrue())
//
//	// List with typed list objects
//	Eventually(ctx, k8s.List(cli, &corev1.ConfigMapList{},
//		client.InNamespace("default"),
//	)).Should(WithTransform(k8s.ListItems(), HaveLen(2)))
//
//	// Create a typed object and assert on it
//	Eventually(ctx, k8s.Create(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//		Data: map[string]string{"key": "value"},
//	})).Should(jq.Match(`.data.key == "value"`))
//
//	// Update with a typed callback — no casting needed
//	Eventually(ctx, k8s.Update(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		cm.Data["key"] = "new-value"
//	})).Should(jq.Match(`.data.key == "new-value"`))
//
//	// Update a status subresource
//	Eventually(ctx, k8s.StatusUpdate(cli, &corev1.Pod{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-pod",
//			Namespace: "default",
//		},
//	}, func(pod *corev1.Pod) {
//		pod.Status.Phase = corev1.PodSucceeded
//	})).Should(jq.Match(`.status.phase == "Succeeded"`))
//
//	// Create or update idempotently
//	Eventually(ctx, k8s.Upsert(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	}, func(cm *corev1.ConfigMap) {
//		if cm.Data == nil {
//			cm.Data = map[string]string{}
//		}
//		cm.Data["key"] = "reconciled-value"
//	})).Should(jq.Match(`.data.key == "reconciled-value"`))
//
//	// Delete
//	Eventually(ctx, k8s.Delete(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).Should(Succeed())
//
//	// Query events
//	Eventually(ctx, k8s.Events(cli,
//		k8s.InNamespace("default"),
//		k8s.ForObject(corev1.ObjectReference{
//			Kind: "Pod",
//			Name: "my-pod",
//		}),
//	)).Should(ContainElement(
//		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
//			"Reason": Equal("Ready"),
//		}),
//	))
//
//	// Metadata matchers compose with SatisfyAll
//	Eventually(ctx, k8s.Get(cli, &corev1.ConfigMap{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      "my-config",
//			Namespace: "default",
//		},
//	})).Should(SatisfyAll(
//		k8s.HasName("my-config"),
//		k8s.HasLabel("env", "prod"),
//		k8s.HasAnnotation("team", "platform"),
//	))
//
//	// Extract conditions as []map[string]any for generic assertions
//	Eventually(ctx, k8s.Get(cli, deploy)).Should(
//		WithTransform(k8s.Conditions(), ContainElement(
//			HaveKeyWithValue("type", "Available"),
//		)),
//	)
//
//	// Extract conditions as a concrete type for typed assertions
//	Eventually(ctx, k8s.Get(cli, deploy)).Should(
//		WithTransform(k8s.ConditionsOf[metav1.Condition](), ContainElement(
//			SatisfyAll(
//				HaveField("Type", Equal("Available")),
//				HaveField("Status", Equal(metav1.ConditionTrue)),
//			),
//		)),
//	)
package k8s
