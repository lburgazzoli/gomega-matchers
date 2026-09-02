package k8s

import (
	"maps"
	"reflect"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type objectMutator[T client.Object] interface {
	~func(T) | ~func(client.Object)
}

func adaptMutator[T client.Object, F objectMutator[T]](fn F) func(T) {
	value := reflect.ValueOf(fn)
	if typedFn, ok := convertFunction[func(T)](value); ok {
		return typedFn
	}

	if objectFn, ok := convertFunction[func(client.Object)](value); ok {
		return func(obj T) {
			objectFn(obj)
		}
	}

	panic("unsupported object mutator")
}

func convertFunction[T any](value reflect.Value) (T, bool) {
	var zero T
	if !value.IsValid() {
		return zero, false
	}

	target := reflect.TypeFor[T]()
	if !value.Type().ConvertibleTo(target) {
		return zero, false
	}

	result, ok := value.Convert(target).Interface().(T)

	return result, ok
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
	maps.Copy(out, input)

	return out
}
