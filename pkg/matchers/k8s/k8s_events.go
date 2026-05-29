package k8s

import (
	"context"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

// EventOption configures how Events retrieves and filters Kubernetes events.
type EventOption interface {
	applyToEvents(opts *eventOptions)
}

type eventOptions struct {
	listOptions []client.ListOption
	filters     []func(*corev1.Event) bool
}

type eventOptionFunc func(*eventOptions)

func (f eventOptionFunc) applyToEvents(opts *eventOptions) {
	f(opts)
}

// InNamespace limits event listing to a single namespace.
func InNamespace(namespace string) EventOption {
	return eventOptionFunc(func(opts *eventOptions) {
		opts.listOptions = append(opts.listOptions, client.InNamespace(namespace))
	})
}

// MatchingLabels applies a label selector to the event list request.
func MatchingLabels(labels client.MatchingLabels) EventOption {
	return eventOptionFunc(func(opts *eventOptions) {
		opts.listOptions = append(opts.listOptions, labels)
	})
}

// ForObject filters events whose involved object matches the non-empty fields
// from the provided object reference.
func ForObject(ref corev1.ObjectReference) EventOption {
	return eventOptionFunc(func(opts *eventOptions) {
		opts.filters = append(opts.filters, func(event *corev1.Event) bool {
			return matchesObjectReference(ref, event.InvolvedObject)
		})
	})
}

// Events lists typed Kubernetes events and returns them as a plain slice so
// callers can use standard Gomega collection matchers directly.
func (m *Matcher) Events(opts ...EventOption) func(context.Context) ([]corev1.Event, error) {
	return func(ctx context.Context) ([]corev1.Event, error) {
		resolved := resolveEventOptions(opts...)
		events := &corev1.EventList{}

		if err := m.client.List(ctx, events, resolved.listOptions...); err != nil {
			return nil, err
		}

		return resolved.filter(events.Items), nil
	}
}

// HasEvent reports whether the listed events contain at least one element that
// matches the provided Gomega matcher.
func (m *Matcher) HasEvent(
	eventMatcher types.GomegaMatcher,
	opts ...EventOption,
) func(context.Context) (bool, error) {
	return func(ctx context.Context) (bool, error) {
		events, err := m.Events(opts...)(ctx)
		if err != nil {
			return false, err
		}

		return gomega.ContainElement(eventMatcher).Match(events)
	}
}

func resolveEventOptions(opts ...EventOption) eventOptions {
	resolved := eventOptions{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}

		opt.applyToEvents(&resolved)
	}

	return resolved
}

func (opts eventOptions) filter(events []corev1.Event) []corev1.Event {
	if len(opts.filters) == 0 {
		return events
	}

	filtered := make([]corev1.Event, 0, len(events))
	for i := range events {
		event := &events[i]
		if opts.matches(event) {
			filtered = append(filtered, *event)
		}
	}

	return filtered
}

func (opts eventOptions) matches(event *corev1.Event) bool {
	for _, filter := range opts.filters {
		if !filter(event) {
			return false
		}
	}

	return true
}

func matchesObjectReference(expected corev1.ObjectReference, actual corev1.ObjectReference) bool {
	return matchesWhenSet(expected.Kind, actual.Kind) &&
		matchesWhenSet(expected.Namespace, actual.Namespace) &&
		matchesWhenSet(expected.Name, actual.Name) &&
		matchesWhenSet(expected.UID, actual.UID) &&
		matchesWhenSet(expected.APIVersion, actual.APIVersion) &&
		matchesWhenSet(expected.ResourceVersion, actual.ResourceVersion) &&
		matchesWhenSet(expected.FieldPath, actual.FieldPath)
}

func matchesWhenSet[T comparable](expected T, actual T) bool {
	var zero T

	return expected == zero || expected == actual
}
