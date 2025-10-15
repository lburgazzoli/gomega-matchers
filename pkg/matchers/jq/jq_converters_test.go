package jq_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lburgazzoli/gomega-matchers/pkg/matchers/jq"
	"github.com/onsi/gomega/gbytes"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/onsi/gomega"
)

func TestStringConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.StringConverter(`{"foo":"bar"}`)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar"}))
}

func TestByteSliceConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.ByteSliceConverter([]byte(`{"foo":"bar"}`))

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar"}))
}

func TestRawMessageConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	result, err := jq.RawMessageConverter(json.RawMessage(`{"foo":"bar"}`))

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar"}))
}

func TestGBytesBufferConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	buf := gbytes.NewBuffer()
	_, err := buf.Write([]byte(`{"foo":"bar"}`))
	g.Expect(err).ShouldNot(HaveOccurred())

	result, err := jq.GBytesBufferConverter(buf)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar"}))
}

func TestReaderConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	reader := strings.NewReader(`{"foo":"bar"}`)
	result, err := jq.ReaderConverter(reader)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(map[string]any{"foo": "bar"}))
}

func TestUnstructuredConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	obj := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "test-pod",
			},
		},
	}

	result, err := jq.UnstructuredConverter(obj)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(obj.Object))
}

func TestUnstructuredPtrConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]any{
				"name": "test-pod",
			},
		},
	}

	result, err := jq.UnstructuredPtrConverter(obj)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(obj.Object))
}

func TestMapConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := map[string]any{"foo": "bar"}
	result, err := jq.MapConverter(input)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(input))
}

func TestSliceConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	input := []any{"foo", "bar"}
	result, err := jq.SliceConverter(input)

	g.Expect(err).ShouldNot(HaveOccurred())
	g.Expect(result).Should(Equal(input))
}

func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "string JSON object",
			input:    `{"foo":"bar"}`,
			expected: map[string]any{"foo": "bar"},
		},
		{
			name:     "string JSON array",
			input:    `["foo","bar"]`,
			expected: []any{"foo", "bar"},
		},
		{
			name:     "byte slice JSON object",
			input:    []byte(`{"foo":"bar"}`),
			expected: map[string]any{"foo": "bar"},
		},
		{
			name:     "byte slice JSON array",
			input:    []byte(`["foo","bar"]`),
			expected: []any{"foo", "bar"},
		},
		{
			name:     "map pass-through",
			input:    map[string]any{"foo": "bar", "num": 42},
			expected: map[string]any{"foo": "bar", "num": 42},
		},
		{
			name:     "slice pass-through",
			input:    []any{"foo", "bar", 42},
			expected: []any{"foo", "bar", 42},
		},
		{
			name: "unstructured.Unstructured",
			input: unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Pod",
				},
			},
			expected: map[string]any{
				"apiVersion": "v1",
				"kind":       "Pod",
			},
		},
		{
			name: "*unstructured.Unstructured",
			input: &unstructured.Unstructured{
				Object: map[string]any{
					"apiVersion": "v1",
					"kind":       "Service",
				},
			},
			expected: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			result, err := jq.Convert(tt.input)

			g.Expect(err).ShouldNot(HaveOccurred())
			g.Expect(result).Should(Equal(tt.expected))
		})
	}
}

func TestConvertUnsupportedType(t *testing.T) {
	t.Parallel()

	type UnsupportedStruct struct {
		Value string
	}

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "int",
			input: 42,
		},
		{
			name:  "float",
			input: 3.14,
		},
		{
			name:  "bool",
			input: true,
		},
		{
			name:  "struct without converter",
			input: UnsupportedStruct{Value: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			_, err := jq.Convert(tt.input)

			g.Expect(err).Should(HaveOccurred())
			g.Expect(err.Error()).Should(ContainSubstring("unsuported type"))
		})
	}
}

func TestConvertInvalidJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "empty byte slice",
			input: []byte{},
		},
		{
			name:  "malformed JSON string",
			input: `{"foo": invalid}`,
		},
		{
			name:  "malformed JSON bytes",
			input: []byte(`{"foo": invalid}`),
		},
		{
			name:  "JSON primitive string",
			input: `"just a string"`,
		},
		{
			name:  "JSON primitive number",
			input: `42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			_, err := jq.Convert(tt.input)

			g.Expect(err).Should(HaveOccurred())
		})
	}
}

func TestConvertErrorPropagation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	type ErrorType struct {
		Value string
	}

	customErr := errors.New("custom conversion error")

	jq.RegisterConverter(func(v any) (any, error) {
		_, ok := v.(ErrorType)
		if !ok {
			return nil, jq.ErrTypeNotSupported
		}

		return nil, customErr
	})

	_, err := jq.Convert(ErrorType{Value: "test"})

	g.Expect(err).Should(HaveOccurred())
	g.Expect(errors.Is(err, customErr)).Should(BeTrue())
}

func TestCustomConverterRegistration(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	type CustomType struct {
		Value string
	}

	jq.RegisterConverter(func(v any) (any, error) {
		ct, ok := v.(CustomType)
		if !ok {
			return nil, jq.ErrTypeNotSupported
		}

		return map[string]any{"value": ct.Value}, nil
	})

	g.Expect(CustomType{Value: "test"}).Should(
		jq.Match(`.value == "test"`),
	)
}

func TestCustomConverterPrecedence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	type CustomString string

	called := false

	jq.RegisterConverter(func(v any) (any, error) {
		_, ok := v.(CustomString)
		if !ok {
			return nil, jq.ErrTypeNotSupported
		}

		called = true

		return map[string]any{"custom": "override"}, nil
	})

	g.Expect(CustomString("some string")).Should(
		jq.Match(`.custom == "override"`),
	)

	g.Expect(called).Should(BeTrue())
}

func TestCustomStructConverter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	type Person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	jq.RegisterConverter(func(v any) (any, error) {
		p, ok := v.(Person)
		if !ok {
			return nil, jq.ErrTypeNotSupported
		}

		data, err := json.Marshal(p)
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err //nolint:wrapcheck
		}

		return result, nil
	})

	g.Expect(Person{Name: "Alice", Age: 30}).Should(
		jq.Match(`.name == "Alice" and .age == 30`),
	)
}
