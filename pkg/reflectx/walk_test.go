package reflectx

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/pasataleo/go-testingx/pkg/testingx"
)

// node is a self-referential struct for cycle detection tests.
type node struct {
	Next *node
	Name string
}

// noopWalk is a WalkFunc that does nothing.
var noopWalk = WalkFunc(func(_ Path, _ reflect.Value, _ reflect.StructField) error {
	return nil
})

// collectPaths returns a WalkFunc that appends each visited path to the slice.
func collectPaths(visited *[]string) WalkFunc {
	return func(path Path, _ reflect.Value, _ reflect.StructField) error {
		*visited = append(*visited, path.String())
		return nil
	}
}

func TestWalk(t *testing.T) {
	t.Run("flat_struct", func(t *testing.T) {
		type s struct {
			A string
			B int
			C bool
		}
		var visited []string
		testingx.Call(t, Walk, &s{}, collectPaths(&visited)).NoError()
		testingx.Capture(t, visited).Equal([]string{"A", "B", "C"})
	})

	t.Run("empty_struct", func(t *testing.T) {
		var visited []string
		testingx.Call(t, Walk, &struct{}{}, collectPaths(&visited)).NoError()
		testingx.Capture(t, len(visited)).Equal(0)
	})

	t.Run("unexported_fields_skipped", func(t *testing.T) {
		type s struct {
			Exported   string
			unexported string //nolint:unused // intentionally unused to test that unexported fields are skipped
		}
		var visited []string
		testingx.Call(t, Walk, &s{}, collectPaths(&visited)).NoError()
		testingx.Capture(t, visited).Equal([]string{"Exported"})
	})

	t.Run("nested_struct_post_order", func(t *testing.T) {
		type inner struct {
			A string
			B int
		}
		type outer struct {
			Inner inner
			C     string
		}
		var visited []string
		testingx.Call(t, Walk, &outer{}, collectPaths(&visited)).NoError()
		// Children before parent: Inner.A, Inner.B, then Inner, then C
		testingx.Capture(t, visited).Equal([]string{"Inner.A", "Inner.B", "Inner", "C"})
	})

	t.Run("deeply_nested", func(t *testing.T) {
		type level2 struct {
			X string
		}
		type level1 struct {
			L2 level2
		}
		type root struct {
			L1 level1
		}
		var visited []string
		testingx.Call(t, Walk, &root{}, collectPaths(&visited)).NoError()
		testingx.Capture(t, visited).Equal([]string{"L1.L2.X", "L1.L2", "L1"})
	})

	t.Run("nil_pointer_to_struct_initialized", func(t *testing.T) {
		type inner struct {
			A string
		}
		type outer struct {
			Inner *inner
		}
		o := &outer{}
		testingx.Capture(t, o.Inner).Nil()

		var visited []string
		testingx.Call(t, Walk, o, collectPaths(&visited)).NoError()
		testingx.Capture(t, o.Inner).NotNil()
		testingx.Capture(t, visited).Equal([]string{"Inner.A", "Inner"})
	})

	t.Run("error_aggregation", func(t *testing.T) {
		type s struct {
			A string
			B string
		}
		failAll := WalkFunc(func(path Path, _ reflect.Value, _ reflect.StructField) error {
			return fmt.Errorf("error at %s", path)
		})
		err := Walk(&s{}, failAll)
		testingx.Capture(t, err).HasError("error at A")
		testingx.Capture(t, err).HasError("error at B")
	})

	t.Run("nested_child_error_skips_parent_callback", func(t *testing.T) {
		type inner struct {
			A string
		}
		type outer struct {
			Inner inner
			B     string
		}
		var visited []string
		failInner := WalkFunc(func(path Path, _ reflect.Value, _ reflect.StructField) error {
			visited = append(visited, path.String())
			if path.String() == "Inner.A" {
				return fmt.Errorf("fail")
			}
			return nil
		})
		err := Walk(&outer{}, failInner)
		testingx.Capture(t, err).HasError("fail")
		// Inner.A errors, so walk skips the Inner parent callback, but still visits B
		testingx.Capture(t, visited).Equal([]string{"Inner.A", "B"})
	})

	t.Run("callback_receives_correct_values", func(t *testing.T) {
		type s struct {
			Name string
			Age  int
		}
		in := &s{Name: "alice", Age: 30}
		var names []string
		var values []interface{}
		cb := WalkFunc(func(path Path, value reflect.Value, field reflect.StructField) error {
			names = append(names, field.Name)
			values = append(values, value.Interface())
			return nil
		})
		testingx.Call(t, Walk, in, cb).NoError()
		testingx.Capture(t, names).Equal([]string{"Name", "Age"})
		testingx.Capture(t, values[0]).Equal("alice")
		testingx.Capture(t, values[1]).Equal(30)
	})
}

func TestWalkCycleDetection(t *testing.T) {
	t.Run("self_referential_struct", func(t *testing.T) {
		var visited []string
		testingx.Call(t, Walk, &node{Name: "root"}, collectPaths(&visited)).NoError()
		// Next is not recursed into (cycle), but callback is still called for it
		testingx.Capture(t, visited).Equal([]string{"Next", "Name"})
	})

	t.Run("nil_pointer_not_initialized", func(t *testing.T) {
		n := &node{Name: "root"}
		testingx.Call(t, Walk, n, noopWalk).NoError()
		// Nil pointer should NOT be initialized when recursion is skipped
		testingx.Capture(t, n.Next).Nil()
	})

	t.Run("non_nil_self_reference", func(t *testing.T) {
		child := &node{Name: "child"}
		root := &node{Next: child, Name: "root"}
		var visited []string
		testingx.Call(t, Walk, root, collectPaths(&visited)).NoError()
		// Next is non-nil but recursion is still skipped due to type cycle
		testingx.Capture(t, visited).Equal([]string{"Next", "Name"})
	})

	t.Run("same_type_at_sibling_positions", func(t *testing.T) {
		type inner struct {
			Value string
		}
		type outer struct {
			A *inner
			B *inner
		}
		var visited []string
		testingx.Call(t, Walk, &outer{}, collectPaths(&visited)).NoError()
		// Both A and B should be fully walked — same type at siblings is not a cycle
		testingx.Capture(t, visited).Equal([]string{"A.Value", "A", "B.Value", "B"})
	})
}

func TestWalkPanics(t *testing.T) {
	t.Run("non_pointer", func(t *testing.T) {
		type s struct{ A string }
		testingx.Panics(t, nil, Walk, s{}, noopWalk)
	})

	t.Run("nil_pointer", func(t *testing.T) {
		testingx.Panics(t, nil, Walk, (*struct{ A string })(nil), noopWalk)
	})

	t.Run("pointer_to_non_struct", func(t *testing.T) {
		s := "hello"
		testingx.Panics(t, nil, Walk, &s, noopWalk)
	})
}
