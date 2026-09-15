package reflectx

import (
	"reflect"

	"github.com/pasataleo/go-errorsx/pkg/errorsx"
)

// WalkFunc is the callback type for Walk. It is called for every exported field
// in the struct, including fields that are themselves structs. The Path argument
// identifies the field's location within the struct hierarchy, the reflect.Value
// is the field's value, and the reflect.StructField contains the field's
// metadata. The final argument is the field's data: whatever the Walker's Enter
// returned for this field, or the data handed to Walk if there is no Enter.
type WalkFunc func(Path, reflect.Value, reflect.StructField, interface{}) error

// EnterFunc is called on the way down, before a field's children are walked.
// The data it returns is scoped to that field: the children below the field
// receive it, and so does the field's own WalkFunc, while the field's siblings
// receive the data their shared parent produced. Returning the data unchanged
// passes the parent's data through untouched.
type EnterFunc func(Path, reflect.Value, reflect.StructField, interface{}) (data interface{}, err error)

// Walker traverses a struct with a callback for each direction of travel. The
// zero Walker does nothing.
//
// The ordering contract is enter -> children -> visit. Data flows down: each
// Enter derives its field's data from the data of the field above it, and each
// Visit receives the data its own Enter returned.
type Walker struct {
	// Enter is called on every exported field before Walk recurses into it,
	// including the fields Walk does not recurse into: non-structs, and struct
	// types already being walked further up the current path. It receives the
	// field itself, as Visit does, with any nil pointer Walk is going to recurse
	// into already initialized. An error from Enter skips both the field's
	// children and its Visit. Optional.
	Enter EnterFunc

	// Visit is called on every exported field after its children have been
	// walked, with the data that field's own Enter returned. This is the
	// callback Walk takes. Optional.
	Visit WalkFunc
}

// Walk recursively traverses target, which must be a non-nil pointer to a
// struct, and calls the callback for every exported field. For fields that are
// structs (or pointers to structs), Walk recurses into the children first and
// then calls the callback on the struct field itself. This means the struct's
// child fields are already populated when the callback receives the parent,
// allowing the callback to use those values for more complex processing. Nil
// pointers to structs are initialized automatically. Errors from the callback
// are aggregated and returned together.
//
// data is handed to every callback unchanged, since there is no Enter to derive
// anything from it. A caller that wants per-field data, or a hook on the way
// down, uses Walker directly.
//
// Self-referential struct hierarchies are detected and handled gracefully: if a
// struct type is already being walked in the current path, its fields are not
// recursed into but the callback is still called on the field itself.
func Walk(target interface{}, data interface{}, callback WalkFunc) error {
	return Walker{Visit: callback}.Walk(target, data)
}

// Walk traverses target, which must be a non-nil pointer to a struct, calling
// Enter and Visit as described on Walker. data is the data for the top-level
// fields: the Enter of each receives it, and without an Enter it reaches every
// Visit unchanged. It panics on anything that is not a non-nil pointer to a
// struct.
func (w Walker) Walk(target interface{}, data interface{}) error {
	value := reflect.ValueOf(target)
	if value.Kind() == reflect.Interface {
		// automatically unpack interfaces
		value = value.Elem()
	}

	if value.Kind() != reflect.Pointer {
		panic("target must be a pointer to a struct")
	}
	if value.IsNil() {
		panic("target must not be nil")
	}
	value = value.Elem()

	if value.Kind() != reflect.Struct {
		panic("target must be a pointer to a struct")
	}

	return walk(value, data, nil, w, make(map[reflect.Type]bool))
}

func walk(value reflect.Value, data interface{}, path Path, w Walker, seen map[reflect.Type]bool) error {
	seen[value.Type()] = true
	defer delete(seen, value.Type())

	var errs error
	for i := 0; i < value.NumField(); i++ {
		if !value.Field(i).CanSet() {
			continue // skip unexported fields
		}

		if err := w.field(value, i, data, path.Append(value.Type().Field(i).Name), seen); err != nil {
			errs = errorsx.Append(errs, err)
			continue
		}
	}

	return errs
}

// field walks one field of v. It is a function of its own rather than the body
// of the loop above so that the data Enter returns can shadow the parameter for
// the duration of this field alone: the children below it and its own Visit see
// that data, while the fields after it in the loop still see the parent's.
func (w Walker) field(v reflect.Value, i int, data interface{}, path Path, seen map[reflect.Type]bool) error {
	value, field := v.Field(i), v.Type().Field(i)

	currentType := UnpackType(field.Type)
	recurse := currentType.Kind() == reflect.Struct && !seen[currentType]

	// Unpack initializes a nil pointer, which is what makes the field walkable
	// at all. Only the fields recursed into are unpacked, so a nil pointer whose
	// type is already on the current path stays nil.
	//
	// It runs before Enter for the side effect rather than the result: Enter and
	// Visit are handed the same field, so they should agree about it, and
	// without this Enter would see a nil pointer that the recursion had filled
	// in by the time Visit saw it. The unpacked value itself stays out of the
	// callbacks - it does not exist for the fields Walk does not recurse into,
	// and Enter is called on those too.
	var current reflect.Value
	if recurse {
		current = Unpack(value)
	}

	if w.Enter != nil {
		var err error
		if data, err = w.Enter(path, value, field, data); err != nil {
			return err
		}
	}

	if recurse {
		if err := walk(current, data, path, w, seen); err != nil {
			// A child error skips this field's Visit: the field's children are
			// not all populated, so a callback that was promised them cannot do
			// its job.
			return err
		}
	}

	if w.Visit == nil {
		return nil
	}
	return w.Visit(path, value, field, data)
}

// UnpackType strips all pointer wrappers from a reflect.Type, returning the
// underlying type. Interfaces are left as they are: a reflect.Type describing
// an interface has no dynamic type behind it to unwrap.
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
