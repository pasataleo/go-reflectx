package reflectx

import (
	"reflect"
	"strconv"

	"github.com/pasataleo/go-errorsx/pkg/errorsx"
)

var (
	settableType = reflect.TypeOf((*StringSettable)(nil)).Elem()
)

// StringSettable is implemented by types that can populate themselves from a
// string representation. SetString and CanSetString check for this interface
// before falling back to built-in type handling.
type StringSettable interface {
	SetString(v string) error
}

// CanSetString reports whether a value of type t can be populated from a string.
// Returns true for types implementing StringSettable (including via pointer
// receiver), and for all primitive types: string, bool, int*, uint*, float*,
// and complex*.
func CanSetString(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		if t.Implements(settableType) {
			return true
		}
		t = t.Elem()
	}

	if t.Implements(settableType) {
		return true
	}
	if reflect.PointerTo(t).Implements(settableType) {
		return true
	}

	switch t.Kind() {
	case reflect.String:
		return true
	case reflect.Bool:
		return true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}

// SetStringOptFn is a functional option for SetString.
type SetStringOptFn func(opts *SetStringOpts) error

// SetStringOpts holds configuration for SetString behaviour.
type SetStringOpts struct {
	EmptyStringIsTrue bool
}

// EmptyStringIsTrue returns an option that causes SetString to treat an empty
// string as true when setting a bool field. This is useful for flag-style
// parsing where --verbose (with no value) should mean true.
func EmptyStringIsTrue() SetStringOptFn {
	return func(opts *SetStringOpts) error {
		opts.EmptyStringIsTrue = true
		return nil
	}
}

// SetString sets value from the string s. Pointer chains are automatically
// unwrapped and nil pointers initialized. If the value's type implements
// StringSettable, that method is used. Otherwise the built-in type conversion
// handles string, bool, int*, uint*, float*, and complex*. Panics if the
// type is not supported — use CanSetString to check first.
func SetString(value reflect.Value, s string, fns ...SetStringOptFn) error {
	opts := new(SetStringOpts)
	for _, fn := range fns {
		if err := fn(opts); err != nil {
			return err
		}
	}

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}

		if value.Type().Implements(settableType) {
			return value.Interface().(StringSettable).SetString(s)
		}

		value = value.Elem()
	}

	if value.Type().Implements(settableType) {
		return value.Interface().(StringSettable).SetString(s)
	}
	if value.CanAddr() && reflect.PointerTo(value.Type()).Implements(settableType) {
		return value.Addr().Interface().(StringSettable).SetString(s)
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(s)
		return nil
	case reflect.Bool:
		if s == "" && opts.EmptyStringIsTrue {
			value.SetBool(true)
			return nil
		}
		// Accept 1/0 in addition to strconv.ParseBool's true/false/t/f
		if s == "1" {
			value.SetBool(true)
			return nil
		}
		if s == "0" {
			value.SetBool(false)
			return nil
		}
		b, err := strconv.ParseBool(s)
		if err != nil {
			return errorsx.Wrapf(err, "failed to parse %q; bool required", s)
		}
		value.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return errorsx.Wrapf(err, "failed to parse %q; int required", s)
		}
		if value.OverflowInt(i) {
			return errorsx.Newf(errorsx.Unknown, nil, "failed to parse %q; value overflows int type", s)
		}
		value.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return errorsx.Wrapf(err, "failed to parse %q; uint required", s)
		}
		if value.OverflowUint(u) {
			return errorsx.Newf(errorsx.Unknown, nil, "failed to parse %q; value overflows uint type", s)
		}
		value.SetUint(u)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return errorsx.Wrapf(err, "failed to parse %q; float required", s)
		}
		if value.OverflowFloat(f) {
			return errorsx.Newf(errorsx.Unknown, nil, "failed to parse %q; value overflows float type", s)
		}
		value.SetFloat(f)
		return nil
	case reflect.Complex64, reflect.Complex128:
		bitSize := 64
		if value.Kind() == reflect.Complex128 {
			bitSize = 128
		}
		c, err := strconv.ParseComplex(s, bitSize)
		if err != nil {
			return errorsx.Wrapf(err, "failed to parse %q; complex required", s)
		}
		value.SetComplex(c)
		return nil
	default:
		panic("SetString does not support type: " + value.Kind().String())
	}
}
