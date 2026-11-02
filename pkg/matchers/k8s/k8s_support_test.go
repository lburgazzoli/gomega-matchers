package k8s_test

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	. "github.com/onsi/gomega"
)

func asUnstructured(t *testing.T, object client.Object) *unstructured.Unstructured {
	t.Helper()

	data, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())

	return &unstructured.Unstructured{Object: data}
}

func completeDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default", Generation: 2},
		Spec:       appsv1.DeploymentSpec{Replicas: new(int32(2))},
		Status:     appsv1.DeploymentStatus{ObservedGeneration: 2, UpdatedReplicas: 2, Replicas: 2, AvailableReplicas: 2},
	}
}
