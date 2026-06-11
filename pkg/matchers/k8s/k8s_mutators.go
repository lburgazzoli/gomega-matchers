package k8s

import "sigs.k8s.io/controller-runtime/pkg/client"

type objectMutator[T client.Object] interface {
	~func(T) | ~func(client.Object)
}

func adaptMutator[T client.Object, F objectMutator[T]](fn F) func(T) {
	return func(obj T) {
		switch typed := any(fn).(type) {
		case func(T):
			typed(obj)
		case func(client.Object):
			typed(obj)
		}
	}
}

// SetLabel returns a mutator that sets metadata.labels[key] = value.
func SetLabel(key string, value string) func(client.Object) {
	return func(obj client.Object) {
		labels := copyStringMap(obj.GetLabels())
		labels[key] = value
		obj.SetLabels(labels)
	}
}

// SetAnnotation returns a mutator that sets metadata.annotations[key] = value.
func SetAnnotation(key string, value string) func(client.Object) {
	return func(obj client.Object) {
		annotations := copyStringMap(obj.GetAnnotations())
		annotations[key] = value
		obj.SetAnnotations(annotations)
	}
}

// Apply composes multiple object mutators and applies them in order.
func Apply(fns ...func(client.Object)) func(client.Object) {
	return func(obj client.Object) {
		for _, fn := range fns {
			fn(obj)
		}
	}
}

func copyStringMap(input map[string]string) map[string]string {
	if input == nil {
		return map[string]string{}
	}

	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}

	return out
}
