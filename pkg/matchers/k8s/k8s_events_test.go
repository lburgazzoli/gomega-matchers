package k8s_test

import (
	"context"
	"errors"
	"testing"

	"github.com/onsi/gomega/gstruct"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/k8s"

	. "github.com/onsi/gomega"
)

func TestTypedEvents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	firstEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "frontend",
			},
		},
		Reason: "Created",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		},
	}

	secondEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "backend",
			},
		},
		Reason: "Ready",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-b",
			Namespace: "default",
		},
	}

	otherNamespaceEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-3",
			Namespace: "other",
		},
		Reason: "Created",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "other",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(firstEvent, secondEvent, otherNamespaceEvent).
		Build()

	k := k8s.New(c, scheme)

	g.Eventually(k.Events(k8s.InNamespace("default"))).
		WithContext(t.Context()).
		Should(HaveLen(2))
}

func TestTypedEventsWithMatchingLabels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	firstEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "frontend",
			},
		},
		Reason: "Created",
	}

	secondEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "backend",
			},
		},
		Reason: "Ready",
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(firstEvent, secondEvent).
		Build()

	k := k8s.New(c, scheme)

	g.Eventually(k.Events(
		k8s.InNamespace("default"),
		k8s.MatchingLabels(client.MatchingLabels{"app": "frontend"}),
	)).
		WithContext(t.Context()).
		Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Created"),
		})))
}

func TestTypedEventsForObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	firstEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
		},
		Reason: "Created",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		},
	}

	secondEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-2",
			Namespace: "default",
		},
		Reason: "Ready",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-b",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(firstEvent, secondEvent).
		Build()

	k := k8s.New(c, scheme)

	g.Eventually(k.Events(
		k8s.InNamespace("default"),
		k8s.ForObject(corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		}),
	)).
		WithContext(t.Context()).
		Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Created"),
		})))
}

func TestTypedHasEvent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
		},
		Reason: "Ready",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(event).
		Build()

	k := k8s.New(c, scheme)

	g.Eventually(k.HasEvent(
		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Ready"),
		}),
		k8s.InNamespace("default"),
		k8s.ForObject(corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		}),
	)).
		WithContext(t.Context()).
		Should(BeTrue())
}

func TestTypedHasEventNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "event-1",
			Namespace: "default",
		},
		Reason: "Created",
		InvolvedObject: corev1.ObjectReference{
			Kind:      "Workbench",
			Name:      "module-a",
			Namespace: "default",
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(event).
		Build()

	k := k8s.New(c, scheme)

	g.Eventually(k.HasEvent(
		gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Ready"),
		}),
		k8s.InNamespace("default"),
	)).
		WithContext(t.Context()).
		Should(BeFalse())
}

func TestTypedEventsListError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	expectedErr := errors.New("list failed")
	c := interceptor.NewClient(
		fake.NewClientBuilder().WithScheme(scheme).Build(),
		interceptor.Funcs{
			List: func(
				ctx context.Context,
				client client.WithWatch,
				list client.ObjectList,
				opts ...client.ListOption,
			) error {
				return expectedErr
			},
		},
	)

	k := k8s.New(c, scheme)

	_, err := k.Events()(t.Context())
	g.Expect(err).To(MatchError(expectedErr))
}
