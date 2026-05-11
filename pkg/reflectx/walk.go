package reflectx

import (
	"reflect"

	"github.com/pasataleo/go-errorsx/pkg/errorsx"
)

// WalkFunc is the callback type for Walk. It is called for every exported field
// in the struct, including fields that are themselves structs. The Path argument
// identifies the field's location within the struct hierarchy, the reflect.Value
// is the field's value, and the reflect.StructField contains the field's metadata.
type WalkFunc func(Path, reflect.Value, reflect.StructField) error

// Walk recursively traverses v, which must be a non-nil pointer to a struct,
// and calls the callback for every exported field. For fields that are structs
// (or pointers to structs), Walk recurses into the children first and then
// calls the callback on the struct field itself. This means the struct's child
// fields are already populated when the callback receives the parent, allowing
// the callback to use those values for more complex processing. Nil pointers
// to structs are initialized automatically. Errors from the callback are
// aggregated and returned together.
//
// Self-referential struct hierarchies are detected and handled gracefully: if a
// struct type is already being walked in the current path, its fields are not
// recursed into but the callback is still called on the field itself.
func Walk(v interface{}, callback WalkFunc) error {
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Interface {
		// automatically unpack interfaces
		value = value.Elem()
	}

	if value.Kind() != reflect.Pointer {
		panic("v must be a pointer to a struct")
	}
	if value.IsNil() {
		panic("v must not be nil")
	}
	value = value.Elem()

	if value.Kind() != reflect.Struct {
		panic("v must be a pointer to a struct")
	}

	seen := make(map[reflect.Type]bool)
	return walk(value, nil, callback, seen)
}

func walk(v reflect.Value, path Path, callback WalkFunc, seen map[reflect.Type]bool) error {
	seen[v.Type()] = true
	defer delete(seen, v.Type())

	var errs error
	for i := 0; i < v.NumField(); i++ {
		path := path.Append(v.Type().Field(i).Name)

		if !v.Field(i).CanSet() {
			continue // skip unexported fields
		}

		currentType := UnpackType(v.Type().Field(i).Type)
		if currentType.Kind() == reflect.Struct && !seen[currentType] {
			current := Unpack(v.Field(i))
			if err := walk(current, path, callback, seen); err != nil {
				errs = errorsx.Append(errs, err)
				continue
			}
		}

		if err := callback(path, v.Field(i), v.Type().Field(i)); err != nil {
			errs = errorsx.Append(errs, err)
			continue
		}
	}

	return errs
}

// UnpackType strips all pointer and interface wrappers from a reflect.Type,
// returning the underlying concrete type.
func UnpackType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// Unpack strips all pointer and interface wrappers from a reflect.Value,
// returning the underlying concrete value. Nil pointers are initialized
// automatically.
func Unpack(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Interface {
		value = value.Elem()
	}

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()

		if value.Kind() == reflect.Interface {
			value = value.Elem()
		}
	}
	return value
}
