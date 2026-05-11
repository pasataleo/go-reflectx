package reflectx

import (
	"reflect"
	"testing"

	"github.com/pasataleo/go-testingx/pkg/testingx"
)

// pointerSettable implements StringSettable via pointer receiver.
type pointerSettable struct {
	value string
}

func (p *pointerSettable) SetString(s string) error {
	p.value = "custom:" + s
	return nil
}

// addrOf creates an addressable zero value of type T.
func addrOf[T any]() reflect.Value {
	return reflect.New(reflect.TypeOf((*T)(nil)).Elem()).Elem()
}

func TestCanSetString(t *testing.T) {
	tcs := map[string]struct {
		typ  reflect.Type
		want bool
	}{
		"string":           {reflect.TypeOf(""), true},
		"bool":             {reflect.TypeOf(false), true},
		"int":              {reflect.TypeOf(0), true},
		"int8":             {reflect.TypeOf(int8(0)), true},
		"int64":            {reflect.TypeOf(int64(0)), true},
		"uint":             {reflect.TypeOf(uint(0)), true},
		"uint8":            {reflect.TypeOf(uint8(0)), true},
		"float32":          {reflect.TypeOf(float32(0)), true},
		"float64":          {reflect.TypeOf(float64(0)), true},
		"complex64":        {reflect.TypeOf(complex64(0)), true},
		"complex128":       {reflect.TypeOf(complex128(0)), true},
		"pointer_settable": {reflect.TypeOf(pointerSettable{}), true},
		"ptr_to_settable":  {reflect.TypeOf((*pointerSettable)(nil)), true},
		"ptr_to_string":    {reflect.TypeOf((*string)(nil)), true},
		"struct":           {reflect.TypeOf(struct{}{}), false},
		"slice":            {reflect.TypeOf([]string{}), false},
		"map":              {reflect.TypeOf(map[string]string{}), false},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			testingx.Call(t, CanSetString, tc.typ).Equal(tc.want)
		})
	}
}

func TestSetString(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		v := addrOf[string]()
		testingx.Call(t, SetString, v, "hello").NoError()
		testingx.Capture(t, v.Interface()).Equal("hello")
	})

	t.Run("bool", func(t *testing.T) {
		tcs := map[string]struct {
			input string
			want  bool
		}{
			"true":  {"true", true},
			"false": {"false", false},
			"1":     {"1", true},
			"0":     {"0", false},
			"t":     {"t", true},
			"f":     {"f", false},
		}
		for name, tc := range tcs {
			t.Run(name, func(t *testing.T) {
				v := addrOf[bool]()
				v.SetBool(!tc.want)
				testingx.Call(t, SetString, v, tc.input).NoError()
				testingx.Capture(t, v.Interface()).Equal(tc.want)
			})
		}
	})

	t.Run("bool_empty_without_opt", func(t *testing.T) {
		v := addrOf[bool]()
		testingx.Call(t, SetString, v, "").Error()
	})

	t.Run("bool_empty_with_opt", func(t *testing.T) {
		v := addrOf[bool]()
		testingx.Call(t, SetString, v, "", EmptyStringIsTrue()).NoError()
		testingx.Capture(t, v.Interface()).Equal(true)
	})

	t.Run("bool_invalid", func(t *testing.T) {
		v := addrOf[bool]()
		testingx.Call(t, SetString, v, "notabool").Error()
	})

	t.Run("int", func(t *testing.T) {
		v := addrOf[int]()
		testingx.Call(t, SetString, v, "42").NoError()
		testingx.Capture(t, v.Interface()).Equal(42)
	})

	t.Run("int_negative", func(t *testing.T) {
		v := addrOf[int]()
		testingx.Call(t, SetString, v, "-7").NoError()
		testingx.Capture(t, v.Interface()).Equal(-7)
	})

	t.Run("int8_overflow", func(t *testing.T) {
		v := addrOf[int8]()
		testingx.Call(t, SetString, v, "128").Error()
	})

	t.Run("int_invalid", func(t *testing.T) {
		v := addrOf[int]()
		testingx.Call(t, SetString, v, "abc").Error()
	})

	t.Run("uint", func(t *testing.T) {
		v := addrOf[uint]()
		testingx.Call(t, SetString, v, "99").NoError()
		testingx.Capture(t, v.Interface()).Equal(uint(99))
	})

	t.Run("uint8_overflow", func(t *testing.T) {
		v := addrOf[uint8]()
		testingx.Call(t, SetString, v, "256").Error()
	})

	t.Run("uint_negative", func(t *testing.T) {
		v := addrOf[uint]()
		testingx.Call(t, SetString, v, "-1").Error()
	})

	t.Run("float64", func(t *testing.T) {
		v := addrOf[float64]()
		testingx.Call(t, SetString, v, "3.14").NoError()
		testingx.Capture(t, v.Interface()).Equal(3.14)
	})

	t.Run("float32_overflow", func(t *testing.T) {
		v := addrOf[float32]()
		testingx.Call(t, SetString, v, "3.5e+38").Error()
	})

	t.Run("float_invalid", func(t *testing.T) {
		v := addrOf[float64]()
		testingx.Call(t, SetString, v, "notafloat").Error()
	})

	t.Run("complex128", func(t *testing.T) {
		v := addrOf[complex128]()
		testingx.Call(t, SetString, v, "(1+2i)").NoError()
		testingx.Capture(t, v.Interface()).Equal(complex(1, 2))
	})

	t.Run("complex_invalid", func(t *testing.T) {
		v := addrOf[complex128]()
		testingx.Call(t, SetString, v, "notcomplex").Error()
	})

	t.Run("pointer_to_string", func(t *testing.T) {
		type s struct{ V *string }
		in := &s{}
		v := reflect.ValueOf(in).Elem().Field(0)
		testingx.Call(t, SetString, v, "hello").NoError()
		testingx.Capture(t, in.V).NotNil()
		testingx.Capture(t, *in.V).Equal("hello")
	})

	t.Run("double_pointer_to_int", func(t *testing.T) {
		type s struct{ V **int }
		in := &s{}
		v := reflect.ValueOf(in).Elem().Field(0)
		testingx.Call(t, SetString, v, "42").NoError()
		testingx.Capture(t, in.V).NotNil()
		testingx.Capture(t, *in.V).NotNil()
		testingx.Capture(t, **in.V).Equal(42)
	})

	t.Run("pointer_settable", func(t *testing.T) {
		v := addrOf[pointerSettable]()
		testingx.Call(t, SetString, v, "test").NoError()
		got := v.Interface().(pointerSettable)
		testingx.Capture(t, got.value).Equal("custom:test")
	})

	t.Run("nil_ptr_to_settable", func(t *testing.T) {
		type s struct{ V *pointerSettable }
		in := &s{}
		v := reflect.ValueOf(in).Elem().Field(0)
		testingx.Call(t, SetString, v, "test").NoError()
		testingx.Capture(t, in.V).NotNil()
		testingx.Capture(t, in.V.value).Equal("custom:test")
	})

	t.Run("unsupported_type_panics", func(t *testing.T) {
		v := addrOf[[]string]()
		testingx.Panics(t, nil, SetString, v, "anything")
	})
}
