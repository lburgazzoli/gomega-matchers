package k8s_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"
	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s/condition"

	. "github.com/onsi/gomega"
)

func TestDataExtractsConfigMapData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		Data: map[string]string{
			"foo": "bar",
		},
	}

	g.Expect(cm).To(WithTransform(k8s.Data(), Equal(map[string]string{
		"foo": "bar",
	})))
}

func TestDataExtractsSecretData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	secret := &corev1.Secret{
		Data: map[string][]byte{
			"token": []byte("secret"),
		},
	}

	g.Expect(secret).To(WithTransform(k8s.Data(), Equal(map[string][]byte{
		"token": []byte("secret"),
	})))
}

func TestDataExtractsUnstructuredData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name":      "test-config",
				"namespace": "default",
			},
			"data": map[string]any{
				"foo": "bar",
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Data(), Equal(map[string]any{
		"foo": "bar",
	})))
}

func TestDataReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Data()(42)

	g.Expect(err).To(MatchError("expected *corev1.ConfigMap, *corev1.Secret, or *unstructured.Unstructured, got int"))
}

func TestFinalizersExtractsTypedObjectFinalizers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Finalizers: []string{"example.com/finalizer"},
		},
	}

	g.Expect(cm).To(WithTransform(k8s.Finalizers(), Equal([]string{"example.com/finalizer"})))
}

func TestFinalizersExtractsUnstructuredObjectFinalizers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]any{
				"name": "test-config",
				"finalizers": []any{
					"example.com/finalizer",
				},
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Finalizers(), Equal([]string{"example.com/finalizer"})))
}

func TestFinalizersReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Finalizers()(42)

	g.Expect(err).To(MatchError("expected client.Object, got int"))
}

func TestListItemsExtractsTypedListItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	list := &corev1.ConfigMapList{
		Items: []corev1.ConfigMap{
			{
				Data: map[string]string{
					"key": "value",
				},
			},
		},
	}

	g.Expect(list).To(WithTransform(k8s.ListItems(), HaveLen(1)))
}

func TestListItemsExtractsUnstructuredListItems(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	list := &unstructured.UnstructuredList{
		Items: []unstructured.Unstructured{
			{
				Object: map[string]any{
					"metadata": map[string]any{
						"name": "test",
					},
				},
			},
		},
	}

	g.Expect(list).To(WithTransform(k8s.ListItems(), HaveLen(1)))
}

func TestListItemsReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.ListItems()(42)

	g.Expect(err).To(MatchError("expected runtime.Object list, got int"))
}

func TestListItemsReturnsErrorForNilUnstructuredList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var list *unstructured.UnstructuredList

	_, err := k8s.ListItems()(list)

	g.Expect(err).To(MatchError("expected runtime.Object list, got *unstructured.UnstructuredList"))
}

func TestConditionsExtractsFromTypedObjectWithMetav1Condition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	svc := &corev1.Service{
		Status: corev1.ServiceStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "AllGood",
				},
			},
		},
	}

	g.Expect(svc).To(WithTransform(k8s.Conditions(), ContainElement(
		SatisfyAll(
			HaveKeyWithValue("type", "Ready"),
			HaveKeyWithValue("status", "True"),
			HaveKeyWithValue("reason", "AllGood"),
		),
	)))
}

func TestConditionsComposeWithConditionMatchers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	svc := &corev1.Service{
		Status: corev1.ServiceStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "AllGood",
					Message: "service is ready",
				},
			},
		},
	}

	g.Expect(svc).To(WithTransform(k8s.Conditions(), ContainElement(
		SatisfyAll(
			condition.Is("Ready", "True"),
			condition.HasReason("AllGood"),
			condition.HasMessage("service is ready"),
		),
	)))
}

func TestConditionsExtractsFromTypedObjectWithPerKindCondition(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deploy := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
					Reason: "MinimumReplicasAvailable",
				},
			},
		},
	}

	g.Expect(deploy).To(WithTransform(k8s.Conditions(), ContainElement(
		SatisfyAll(
			HaveKeyWithValue("type", "Available"),
			HaveKeyWithValue("status", "True"),
			HaveKeyWithValue("reason", "MinimumReplicasAvailable"),
		),
	)))
}

func TestConditionsExtractsFromUnstructured(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "example.com/v1",
			"kind":       "Foo",
			"metadata":   map[string]any{"name": "test"},
			"status": map[string]any{
				"conditions": []any{
					map[string]any{
						"type":   "Ready",
						"status": "True",
						"reason": "AllGood",
					},
				},
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Conditions(), ContainElement(
		HaveKeyWithValue("type", "Ready"),
	)))
}

func TestConditionsReturnsNilWhenNoStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	result, err := k8s.Conditions()(cm)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestConditionsReturnsNilWhenNoConditionsField(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata":   map[string]any{"name": "test"},
			"status":     map[string]any{},
		},
	}

	result, err := k8s.Conditions()(obj)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestConditionsReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Conditions()(42)

	g.Expect(err).To(MatchError("expected client.Object, got int"))
}

func TestConditionsOfExtractsTypedMetav1Conditions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	svc := &corev1.Service{
		Status: corev1.ServiceStatus{
			Conditions: []metav1.Condition{
				{
					Type:   "Ready",
					Status: metav1.ConditionTrue,
					Reason: "AllGood",
				},
			},
		},
	}

	g.Expect(svc).To(WithTransform(k8s.ConditionsOf[metav1.Condition](), ContainElement(
		SatisfyAll(
			HaveField("Type", Equal("Ready")),
			HaveField("Status", Equal(metav1.ConditionTrue)),
			HaveField("Reason", Equal("AllGood")),
		),
	)))
}

func TestConditionsOfComposeWithConditionMatchers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	svc := &corev1.Service{
		Status: corev1.ServiceStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "AllGood",
					Message: "service is ready",
				},
			},
		},
	}

	g.Expect(svc).To(WithTransform(k8s.ConditionsOf[metav1.Condition](), ContainElement(
		SatisfyAll(
			condition.Is("Ready", metav1.ConditionTrue),
			condition.HasReason("AllGood"),
			condition.HasMessage("service is ready"),
		),
	)))
}

func TestConditionsOfExtractsTypedDeploymentConditions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deploy := &appsv1.Deployment{
		Status: appsv1.DeploymentStatus{
			Conditions: []appsv1.DeploymentCondition{
				{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
					Reason: "MinimumReplicasAvailable",
				},
			},
		},
	}

	g.Expect(deploy).To(WithTransform(k8s.ConditionsOf[appsv1.DeploymentCondition](), ContainElement(
		SatisfyAll(
			HaveField("Type", Equal(appsv1.DeploymentAvailable)),
			HaveField("Status", Equal(corev1.ConditionTrue)),
			HaveField("Reason", Equal("MinimumReplicasAvailable")),
		),
	)))
}

func TestConditionsOfReturnsNilWhenNoStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
	}

	result, err := k8s.ConditionsOf[metav1.Condition]()(cm)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestConditionsOfReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.ConditionsOf[metav1.Condition]()(42)

	g.Expect(err).To(MatchError("expected client.Object, got int"))
}

func TestPodTemplateExtractsFromTypedPodTemplate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	template := &corev1.PodTemplate{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Name: "app"},
				},
			},
		},
	}

	g.Expect(template).To(WithTransform(k8s.PodTemplate(), SatisfyAll(
		HaveField("Spec.Containers", HaveLen(1)),
		HaveField("Spec.Containers", ContainElement(HaveField("Name", Equal("app")))),
	)))
}

func TestPodTemplateExtractsFromTypedDeployment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app"},
					},
				},
			},
		},
	}

	g.Expect(deploy).To(WithTransform(k8s.PodTemplate(), SatisfyAll(
		HaveField("Spec.Containers", HaveLen(1)),
		HaveField("Spec.Containers", ContainElement(HaveField("Name", Equal("app")))),
	)))
}

func TestPodTemplateExtractsFromTypedCronJob(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cronJob := &batchv1.CronJob{
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "app"},
							},
						},
					},
				},
			},
		},
	}

	g.Expect(cronJob).To(WithTransform(k8s.PodTemplate(), SatisfyAll(
		HaveField("Spec.Containers", HaveLen(1)),
		HaveField("Spec.Containers", ContainElement(HaveField("Name", Equal("app")))),
	)))
}

func TestContainersExtractFromTypedPod(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
			},
		},
	}

	g.Expect(pod).To(WithTransform(k8s.Containers(), ContainElement(
		HaveField("Name", Equal("app")),
	)))
}

func TestContainersComposeWithPodTemplate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deploy := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							Env: []corev1.EnvVar{
								{Name: "LOG_LEVEL", Value: "debug"},
							},
						},
					},
				},
			},
		},
	}

	g.Expect(deploy).To(WithTransform(k8s.PodTemplate(), WithTransform(k8s.Containers(), ContainElement(
		SatisfyAll(
			HaveField("Name", Equal("app")),
			WithTransform(k8s.EnvVars(), ContainElement(SatisfyAll(
				HaveField("Name", Equal("LOG_LEVEL")),
				HaveField("Value", Equal("debug")),
			))),
		),
	))))
}

func TestContainersExtractFromUnstructuredDeployment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata":   map[string]any{"name": "test"},
			"spec": map[string]any{
				"template": map[string]any{
					"spec": map[string]any{
						"containers": []any{
							map[string]any{"name": "app"},
						},
					},
				},
			},
		},
	}

	g.Expect(obj).To(WithTransform(k8s.Containers(), ContainElement(
		HaveField("Name", Equal("app")),
	)))
}

func TestContainersExtractFromTypedCronJob(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cronJob := &batchv1.CronJob{
		Spec: batchv1.CronJobSpec{
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "app"},
							},
						},
					},
				},
			},
		},
	}

	g.Expect(cronJob).To(WithTransform(k8s.Containers(), ContainElement(
		HaveField("Name", Equal("app")),
	)))
}

func TestEnvVarsExtractFromTypedContainer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	container := corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "LOG_LEVEL", Value: "debug"},
		},
	}

	g.Expect(container).To(WithTransform(k8s.EnvVars(), ContainElement(SatisfyAll(
		HaveField("Name", Equal("LOG_LEVEL")),
		HaveField("Value", Equal("debug")),
	))))
}

func TestEnvVarsExtractFromUnstructuredContainer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	container := map[string]any{
		"name": "app",
		"env": []any{
			map[string]any{
				"name":  "LOG_LEVEL",
				"value": "debug",
			},
		},
	}

	g.Expect(container).To(WithTransform(k8s.EnvVars(), ContainElement(SatisfyAll(
		HaveField("Name", Equal("LOG_LEVEL")),
		HaveField("Value", Equal("debug")),
	))))
}

func TestContainersReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.Containers()(42)

	g.Expect(err).To(MatchError("expected struct, pointer to struct, or map[string]any, got int"))
}

func TestEnvVarsReturnsErrorForUnsupportedInput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := k8s.EnvVars()(42)

	g.Expect(err).To(MatchError("expected struct, pointer to struct, or map[string]any, got int"))
}
