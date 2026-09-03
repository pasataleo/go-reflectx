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

// EnterFunc is called on the way down. The leave it returns, if non-nil, is
// called once the field's children have been walked and its WalkFunc has run -
// on every path out, including the ones that skip the WalkFunc: when a child
// errors, when Enter itself errors, and when any of them panics.
type EnterFunc func(Path, reflect.Value, reflect.StructField) (leave func(), err error)

// Walker traverses a struct with a callback for each direction of travel. The
// zero Walker does nothing.
//
// The ordering contract is enter -> children -> visit -> leave, and the pairing
// of an Enter with its leave is guaranteed. That guarantee is the point: a
// caller keeping state that must bracket a subtree opens it in Enter and closes
// it in leave, without having to compare paths or reason about which way the
// walk left the field.
type Walker struct {
	// Enter is called on a struct-typed field before its children are walked,
	// after any nil pointer has been initialized. It is not called for fields
	// Walk does not recurse into: non-structs, and types already being walked
	// further up the current path. Optional.
	Enter EnterFunc

	// Visit is called on every exported field after its children have been
	// walked. This is the callback Walk takes. Optional.
	Visit WalkFunc
}

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
//
// A caller that also needs a hook on the way down uses Walker directly.
func Walk(v interface{}, callback WalkFunc) error {
	return Walker{Visit: callback}.Walk(v)
}

// Walk traverses v, which must be a non-nil pointer to a struct, calling Enter
// and Visit as described on Walker. It panics on anything that is not a
// non-nil pointer to a struct.
func (w Walker) Walk(v interface{}) error {
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
	return walk(value, nil, w, seen)
}

func walk(v reflect.Value, path Path, w Walker, seen map[reflect.Type]bool) error {
	seen[v.Type()] = true
	defer delete(seen, v.Type())

	var errs error
	for i := 0; i < v.NumField(); i++ {
		if !v.Field(i).CanSet() {
			continue // skip unexported fields
		}

		if err := w.field(v, i, path.Append(v.Type().Field(i).Name), seen); err != nil {
			errs = errorsx.Append(errs, err)
			continue
		}
	}

	return errs
}

// field walks one field of v. It is a function of its own rather than the body
// of the loop above so that the leave Enter returns can be deferred: it then
// runs at the end of this field rather than at the end of the whole struct, and
// runs on the paths out that skip Visit as well as on the one that does not.
func (w Walker) field(v reflect.Value, i int, path Path, seen map[reflect.Type]bool) error {
	value, field := v.Field(i), v.Type().Field(i)

	currentType := UnpackType(field.Type)
	if currentType.Kind() == reflect.Struct && !seen[currentType] {
		// Unpack before Enter, not after: it is what initializes a nil pointer,
		// and Enter has no use for a field it cannot yet look inside.
		current := Unpack(value)

		if w.Enter != nil {
			leave, err := w.Enter(path, value, field)
			if leave != nil {
				defer leave()
			}
			if err != nil {
				return err
			}
		}

		if err := walk(current, path, w, seen); err != nil {
			// A child error skips this field's Visit, as it always has: the
			// field's children are not all populated, so a callback that was
			// promised them cannot do its job.
			return err
		}
	}

	if w.Visit == nil {
		return nil
	}
	return w.Visit(path, value, field)
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
