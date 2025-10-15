package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gbytes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ErrTypeNotSupported is returned by converters when the input type is not supported.
// When a converter returns this error, the registry will try the next converter.
var ErrTypeNotSupported = errors.New("type not supported by this converter")

// ConverterFunc converts an input value to a JQ-compatible type (map or slice).
// Returns ErrTypeNotSupported if the input type is not handled by this converter.
type ConverterFunc func(any) (any, error)

//nolint:gochecknoglobals
var (
	convertersMu sync.RWMutex
	converters   []ConverterFunc
)

// RegisterConverter registers a type converter function.
// User-registered converters are prepended to the list and checked before built-in converters.
func RegisterConverter(converter ConverterFunc) {
	convertersMu.Lock()
	defer convertersMu.Unlock()

	converters = append([]ConverterFunc{converter}, converters...)
}

// Convert converts an input value to a JQ-compatible type (map or slice).
// It iterates through registered converters until one successfully converts the value.
func Convert(in any) (any, error) {
	convertersMu.RLock()
	defer convertersMu.RUnlock()

	for _, converter := range converters {
		result, err := converter(in)
		if err == nil {
			return result, nil
		}

		if !errors.Is(err, ErrTypeNotSupported) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("unsuported type:\n%s", format.Object(in, 1))
}

func unmarshalJSON(in []byte) (any, error) {
	if len(in) == 0 {
		return nil, errors.New("a valid Json document is expected")
	}

	switch in[0] {
	case '{':
		data := make(map[string]any)
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		return data, nil
	case '[':
		var data []any
		if err := json.Unmarshal(in, &data); err != nil {
			return nil, fmt.Errorf("unable to unmarshal result, %w", err)
		}

		return data, nil
	default:
		return nil, errors.New("a Json Array or Object is required")
	}
}

//nolint:gochecknoinits
func init() {
	registerBuiltinConverters()
}

func registerBuiltinConverters() {
	RegisterConverter(StringConverter)
	RegisterConverter(ByteSliceConverter)
	RegisterConverter(RawMessageConverter)
	RegisterConverter(GBytesBufferConverter)
	RegisterConverter(ReaderConverter)
	RegisterConverter(UnstructuredConverter)
	RegisterConverter(UnstructuredPtrConverter)
	RegisterConverter(MapConverter)
	RegisterConverter(SliceConverter)
}

// StringConverter converts string to JQ-compatible type.
func StringConverter(in any) (any, error) {
	v, ok := in.(string)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return unmarshalJSON([]byte(v))
}

// ByteSliceConverter converts []byte to JQ-compatible type.
func ByteSliceConverter(in any) (any, error) {
	v, ok := in.([]byte)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return unmarshalJSON(v)
}

// RawMessageConverter converts json.RawMessage to JQ-compatible type.
func RawMessageConverter(in any) (any, error) {
	v, ok := in.(json.RawMessage)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return unmarshalJSON(v)
}

// GBytesBufferConverter converts *gbytes.Buffer to JQ-compatible type.
func GBytesBufferConverter(in any) (any, error) {
	v, ok := in.(*gbytes.Buffer)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return unmarshalJSON(v.Contents())
}

// ReaderConverter converts io.Reader to JQ-compatible type.
func ReaderConverter(in any) (any, error) {
	v, ok := in.(io.Reader)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	data, err := io.ReadAll(v)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}

	return unmarshalJSON(data)
}

// UnstructuredConverter converts unstructured.Unstructured to JQ-compatible type.
func UnstructuredConverter(in any) (any, error) {
	v, ok := in.(unstructured.Unstructured)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return v.Object, nil
}

// UnstructuredPtrConverter converts *unstructured.Unstructured to JQ-compatible type.
func UnstructuredPtrConverter(in any) (any, error) {
	v, ok := in.(*unstructured.Unstructured)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return v.Object, nil
}

// MapConverter converts map types to JQ-compatible type (pass-through).
func MapConverter(in any) (any, error) {
	if reflect.TypeOf(in).Kind() != reflect.Map {
		return nil, ErrTypeNotSupported
	}

	return in, nil
}

// SliceConverter converts slice types to JQ-compatible type (pass-through).
func SliceConverter(in any) (any, error) {
	if reflect.TypeOf(in).Kind() != reflect.Slice {
		return nil, ErrTypeNotSupported
	}

	// Exclude []byte which is handled by ByteSliceConverter
	if _, ok := in.([]byte); ok {
		return nil, ErrTypeNotSupported
	}

	return in, nil
}
