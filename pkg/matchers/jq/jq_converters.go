package jq

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sync"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gbytes"
)

// ErrTypeNotSupported is returned by converters when the input type is not supported.
// When a converter returns this error, the registry will try the next converter.
var ErrTypeNotSupported = errors.New("type not supported by this converter")

// ErrInvalidConverter indicates that a nil converter was registered.
var ErrInvalidConverter = errors.New("converter cannot be nil")

// ConverterFunc converts an input value to a JQ-compatible type (map or slice).
// Returns ErrTypeNotSupported if the input type is not handled by this converter.
type ConverterFunc func(any) (any, error)

// Instance evaluates JQ expressions using its own converter registry.
//
// Use New to create an isolated instance. The package-level functions use an
// unexported default instance for convenient global configuration.
type Instance struct {
	convertersMu sync.RWMutex
	converters   []ConverterFunc
}

var defaultInstance = New()

// New returns an isolated JQ instance initialized with the built-in converters.
func New() *Instance {
	return &Instance{converters: builtinConverters()}
}

// RegisterConverter registers a type converter on the default instance.
func RegisterConverter(converter ConverterFunc) error {
	return defaultInstance.RegisterConverter(converter)
}

// RegisterConverter registers a type converter on the instance.
// User-registered converters are prepended to the list and checked before built-in converters.
func (j *Instance) RegisterConverter(converter ConverterFunc) error {
	if converter == nil {
		return ErrInvalidConverter
	}

	j.convertersMu.Lock()
	j.converters = append([]ConverterFunc{converter}, j.converters...)
	j.convertersMu.Unlock()

	return nil
}

// ResetConverters restores the default instance to its built-in converters.
func ResetConverters() {
	defaultInstance.ResetConverters()
}

// ResetConverters restores the instance to its built-in converters.
func (j *Instance) ResetConverters() {
	j.convertersMu.Lock()
	defer j.convertersMu.Unlock()

	j.converters = builtinConverters()
}

// Convert converts an input value using the default instance.
func Convert(in any) (any, error) {
	return defaultInstance.Convert(in)
}

// Convert converts an input value using the instance's converter registry.
func (j *Instance) Convert(in any) (any, error) {
	j.convertersMu.RLock()
	converters := append([]ConverterFunc(nil), j.converters...)
	j.convertersMu.RUnlock()

	for _, converter := range converters {
		result, err := converter(in)
		if err == nil {
			return normalizeForJQ(result), nil
		}

		if !errors.Is(err, ErrTypeNotSupported) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("unsupported type:\n%s", format.Object(in, 1))
}

// UnmarshalJSON unmarshals JSON bytes into a JQ-compatible type (map or slice).
// Returns an error if the input is not valid JSON or is a JSON primitive.
func UnmarshalJSON(in []byte) (any, error) {
	if len(in) == 0 {
		return nil, errors.New("a valid JSON document is expected")
	}

	var result any
	if err := json.Unmarshal(in, &result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON document: %w", err)
	}

	if result == nil {
		return nil, errors.New("a JSON array or object is required")
	}

	kind := reflect.TypeOf(result).Kind()

	if kind != reflect.Map && kind != reflect.Slice {
		return nil, errors.New("a JSON array or object is required")
	}

	return result, nil
}

func builtinConverters() []ConverterFunc {
	return []ConverterFunc{
		MapConverter,
		GBytesBufferConverter,
		RawMessageConverter,
		ByteSliceConverter,
		SliceConverter,
		StringConverter,
	}
}

// StringConverter converts string to JQ-compatible type.
func StringConverter(in any) (any, error) {
	v, ok := in.(string)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return UnmarshalJSON([]byte(v))
}

// ByteSliceConverter converts []byte to JQ-compatible type.
func ByteSliceConverter(in any) (any, error) {
	v, ok := in.([]byte)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return UnmarshalJSON(v)
}

// RawMessageConverter converts json.RawMessage to JQ-compatible type.
func RawMessageConverter(in any) (any, error) {
	v, ok := in.(json.RawMessage)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return UnmarshalJSON(v)
}

// GBytesBufferConverter converts *gbytes.Buffer to JQ-compatible type.
func GBytesBufferConverter(in any) (any, error) {
	v, ok := in.(*gbytes.Buffer)
	if !ok {
		return nil, ErrTypeNotSupported
	}

	return UnmarshalJSON(v.Contents())
}

// MapConverter converts map types to JQ-compatible type (pass-through).
func MapConverter(in any) (any, error) {
	if in == nil {
		return nil, ErrTypeNotSupported
	}

	if reflect.TypeOf(in).Kind() != reflect.Map {
		return nil, ErrTypeNotSupported
	}

	return in, nil
}

// SliceConverter converts slice types to JQ-compatible type (pass-through).
func SliceConverter(in any) (any, error) {
	if in == nil {
		return nil, ErrTypeNotSupported
	}

	if reflect.TypeOf(in).Kind() != reflect.Slice {
		return nil, ErrTypeNotSupported
	}

	if _, ok := in.([]byte); ok {
		return nil, ErrTypeNotSupported
	}

	if _, ok := in.(json.RawMessage); ok {
		return nil, ErrTypeNotSupported
	}

	return in, nil
}

// normalizeForJQ converts Go numeric types that gojq does not accept
// (int64, int32, uint64, etc.) into the types gojq supports: int,
// float64, and *big.Int. Kubernetes unstructured objects typically
// store integers as int64, which gojq >= 0.12.18 no longer normalizes.
//
// Recursion is limited to map[string]any and []any containers, which is
// sufficient because all built-in converters (JSON unmarshal, map/slice
// pass-through) and K8s converters (unstructured .Object, ToUnstructured)
// produce these types.
func normalizeForJQ(v any) any {
	switch v := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for k, val := range v {
			result[k] = normalizeForJQ(val)
		}

		return result
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			result[i] = normalizeForJQ(val)
		}

		return result
	default:
		return normalizeNumeric(v)
	}
}

func normalizeNumeric(v any) any {
	switch v := v.(type) {
	case int64, int32, int16, int8:
		return normalizeSignedInt(v)
	case uint64, uint32, uint16, uint8, uint:
		return normalizeUnsignedInt(v)
	case float32:
		return float64(v)
	default:
		return v
	}
}

func normalizeSignedInt(v any) any {
	switch v := v.(type) {
	case int64:
		if v >= math.MinInt && v <= math.MaxInt {
			return int(v)
		}

		return big.NewInt(v)
	case int32:
		return int(v)
	case int16:
		return int(v)
	case int8:
		return int(v)
	default:
		return v
	}
}

func normalizeUnsignedInt(v any) any {
	switch v := v.(type) {
	case uint64:
		if v <= math.MaxInt {
			return int(v)
		}

		return new(big.Int).SetUint64(v)
	case uint32:
		return int(v)
	case uint16:
		return int(v)
	case uint8:
		return int(v)
	case uint:
		if v <= math.MaxInt {
			return int(v)
		}

		return new(big.Int).SetUint64(uint64(v))
	default:
		return v
	}
}
