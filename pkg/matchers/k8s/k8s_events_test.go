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

func TestEvents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-1", Namespace: "default", Labels: map[string]string{"app": "frontend"}},
			Reason:     "Created",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Workbench", Name: "module-a", Namespace: "default",
			},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-2", Namespace: "default", Labels: map[string]string{"app": "backend"}},
			Reason:     "Ready",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Workbench", Name: "module-b", Namespace: "default",
			},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-3", Namespace: "other"},
			Reason:     "Created",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Workbench", Name: "module-a", Namespace: "other",
			},
		},
	)

	g.Eventually(k8s.Events(c, k8s.InNamespace("default"))).
		WithContext(t.Context()).
		Should(HaveLen(2))
}

func TestEventsWithMatchingLabels(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-1", Namespace: "default", Labels: map[string]string{"app": "frontend"}},
			Reason:     "Created",
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-2", Namespace: "default", Labels: map[string]string{"app": "backend"}},
			Reason:     "Ready",
		},
	)

	g.Eventually(k8s.Events(c,
		k8s.InNamespace("default"),
		k8s.MatchingLabels(client.MatchingLabels{"app": "frontend"}),
	)).
		WithContext(t.Context()).
		Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Created"),
		})))
}

func TestEventsForObject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-1", Namespace: "default"},
			Reason:     "Created",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Workbench", Name: "module-a", Namespace: "default",
			},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "event-2", Namespace: "default"},
			Reason:     "Ready",
			InvolvedObject: corev1.ObjectReference{
				Kind: "Workbench", Name: "module-b", Namespace: "default",
			},
		},
	)

	g.Eventually(k8s.Events(c,
		k8s.InNamespace("default"),
		k8s.ForObject(corev1.ObjectReference{
			Kind: "Workbench", Name: "module-a", Namespace: "default",
		}),
	)).
		WithContext(t.Context()).
		Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Created"),
		})))
}

func TestEventsContainElement(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "event-1", Namespace: "default"},
		Reason:     "Ready",
		InvolvedObject: corev1.ObjectReference{
			Kind: "Workbench", Name: "module-a", Namespace: "default",
		},
	})

	g.Eventually(k8s.Events(c,
		k8s.InNamespace("default"),
		k8s.ForObject(corev1.ObjectReference{
			Kind: "Workbench", Name: "module-a", Namespace: "default",
		}),
	)).
		WithContext(t.Context()).
		Should(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Ready"),
		})))
}

func TestEventsDoNotContainElement(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	c := newFakeClient(&corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "event-1", Namespace: "default"},
		Reason:     "Created",
		InvolvedObject: corev1.ObjectReference{
			Kind: "Workbench", Name: "module-a", Namespace: "default",
		},
	})

	g.Eventually(k8s.Events(c, k8s.InNamespace("default"))).
		WithContext(t.Context()).
		Should(Not(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Reason": Equal("Ready"),
		}))))
}

func TestEventsListError(t *testing.T) {
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

	_, err := k8s.Events(c)(t.Context())
	g.Expect(err).To(MatchError(expectedErr))
}
