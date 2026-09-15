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
var noopWalk = WalkFunc(func(_ Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
	return nil
})

// collectPaths returns a WalkFunc that appends each visited path to the slice.
func collectPaths(visited *[]string) WalkFunc {
	return func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
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
		testingx.Call(t, Walk, &s{}, nil, collectPaths(&visited)).NoError()
		testingx.Capture(t, visited).Equal([]string{"A", "B", "C"})
	})

	t.Run("empty_struct", func(t *testing.T) {
		var visited []string
		testingx.Call(t, Walk, &struct{}{}, nil, collectPaths(&visited)).NoError()
		testingx.Capture(t, len(visited)).Equal(0)
	})

	t.Run("unexported_fields_skipped", func(t *testing.T) {
		type s struct {
			Exported   string
			unexported string //nolint:unused // intentionally unused to test that unexported fields are skipped
		}
		var visited []string
		testingx.Call(t, Walk, &s{}, nil, collectPaths(&visited)).NoError()
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
		testingx.Call(t, Walk, &outer{}, nil, collectPaths(&visited)).NoError()
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
		testingx.Call(t, Walk, &root{}, nil, collectPaths(&visited)).NoError()
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
		testingx.Call(t, Walk, o, nil, collectPaths(&visited)).NoError()
		testingx.Capture(t, o.Inner).NotNil()
		testingx.Capture(t, visited).Equal([]string{"Inner.A", "Inner"})
	})

	t.Run("data_reaches_every_callback", func(t *testing.T) {
		type inner struct {
			A string
		}
		type outer struct {
			Inner inner
			B     string
		}
		// Walk builds a Walker with no Enter, so nothing derives a new data
		// along the way and every field sees what the caller passed in.
		var seen []interface{}
		cb := WalkFunc(func(_ Path, _ reflect.Value, _ reflect.StructField, data interface{}) error {
			seen = append(seen, data)
			return nil
		})
		testingx.Call(t, Walk, &outer{}, "shared", cb).NoError()
		testingx.Capture(t, seen).Equal([]interface{}{"shared", "shared", "shared"})
	})

	t.Run("error_aggregation", func(t *testing.T) {
		type s struct {
			A string
			B string
		}
		failAll := WalkFunc(func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
			return fmt.Errorf("error at %s", path)
		})
		err := Walk(&s{}, nil, failAll)
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
		failInner := WalkFunc(func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
			visited = append(visited, path.String())
			if path.String() == "Inner.A" {
				return fmt.Errorf("fail")
			}
			return nil
		})
		err := Walk(&outer{}, nil, failInner)
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
		cb := WalkFunc(func(path Path, value reflect.Value, field reflect.StructField, _ interface{}) error {
			names = append(names, field.Name)
			values = append(values, value.Interface())
			return nil
		})
		testingx.Call(t, Walk, in, nil, cb).NoError()
		testingx.Capture(t, names).Equal([]string{"Name", "Age"})
		testingx.Capture(t, values[0]).Equal("alice")
		testingx.Capture(t, values[1]).Equal(30)
	})
}

func TestWalkCycleDetection(t *testing.T) {
	t.Run("self_referential_struct", func(t *testing.T) {
		var visited []string
		testingx.Call(t, Walk, &node{Name: "root"}, nil, collectPaths(&visited)).NoError()
		// Next is not recursed into (cycle), but callback is still called for it
		testingx.Capture(t, visited).Equal([]string{"Next", "Name"})
	})

	t.Run("nil_pointer_not_initialized", func(t *testing.T) {
		n := &node{Name: "root"}
		testingx.Call(t, Walk, n, nil, noopWalk).NoError()
		// Nil pointer should NOT be initialized when recursion is skipped
		testingx.Capture(t, n.Next).Nil()
	})

	t.Run("non_nil_self_reference", func(t *testing.T) {
		child := &node{Name: "child"}
		root := &node{Next: child, Name: "root"}
		var visited []string
		testingx.Call(t, Walk, root, nil, collectPaths(&visited)).NoError()
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
		testingx.Call(t, Walk, &outer{}, nil, collectPaths(&visited)).NoError()
		// Both A and B should be fully walked — same type at siblings is not a cycle
		testingx.Capture(t, visited).Equal([]string{"A.Value", "A", "B.Value", "B"})
	})
}

// trace records a Walker's callbacks in the order they fire, so a test can
// assert the enter -> children -> visit contract directly.
type trace struct {
	events []string
}

// walker returns a Walker that records every callback into the trace. failAt,
// when non-empty, is the path whose Visit returns an error.
func (tr *trace) walker(failAt string) Walker {
	return Walker{
		Enter: func(path Path, _ reflect.Value, _ reflect.StructField, data interface{}) (interface{}, error) {
			tr.events = append(tr.events, "enter "+path.String())
			return data, nil
		},
		Visit: func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
			tr.events = append(tr.events, "visit "+path.String())
			if failAt != "" && path.String() == failAt {
				return fmt.Errorf("fail")
			}
			return nil
		},
	}
}

func TestWalker(t *testing.T) {
	t.Run("enter_fires_before_children", func(t *testing.T) {
		type inner struct {
			A string
			B int
		}
		type outer struct {
			Inner inner
			C     string
		}
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &outer{}, nil).NoError()
		testingx.Capture(t, tr.events).Equal([]string{
			"enter Inner",
			"enter Inner.A",
			"visit Inner.A",
			"enter Inner.B",
			"visit Inner.B",
			"visit Inner",
			"enter C",
			"visit C",
		})
	})

	t.Run("enter_called_for_non_struct_fields", func(t *testing.T) {
		type s struct {
			A string
			B int
			C bool
		}
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &s{}, nil).NoError()
		testingx.Capture(t, tr.events).Equal([]string{
			"enter A", "visit A",
			"enter B", "visit B",
			"enter C", "visit C",
		})
	})

	t.Run("enter_called_for_seen_types", func(t *testing.T) {
		// Enter does not mirror walk's recursion condition: the back-reference
		// gets an Enter of its own even though nothing below it is walked.
		type leaf struct{ Value string }
		type selfish struct {
			Next *selfish
			Leaf leaf
		}
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &selfish{}, nil).NoError()
		testingx.Capture(t, tr.events).Equal([]string{
			"enter Next",
			"visit Next",
			"enter Leaf",
			"enter Leaf.Value",
			"visit Leaf.Value",
			"visit Leaf",
		})
	})

	t.Run("enter_called_after_nil_pointer_init", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct{ Inner *inner }
		o := &outer{}
		testingx.Capture(t, o.Inner).Nil()

		var sawNil []bool
		w := Walker{
			Enter: func(path Path, value reflect.Value, _ reflect.StructField, _ interface{}) (interface{}, error) {
				if path.String() == "Inner" {
					sawNil = append(sawNil, value.IsNil())
				}
				return nil, nil
			},
		}
		testingx.Call(t, w.Walk, o, nil).NoError()
		// Enter saw the pointer already allocated, not the nil it was declared
		// as - and it received the field itself, so IsNil is a legal question.
		testingx.Capture(t, sawNil).Equal([]bool{false})
		testingx.Capture(t, o.Inner).NotNil()
	})

	t.Run("enter_and_visit_agree_about_the_field", func(t *testing.T) {
		// The two callbacks are handed the same field, so a caller can share a
		// helper between them without it behaving differently on the way down.
		type inner struct{ A string }
		type outer struct{ Inner *inner }

		seen := map[string][]bool{}
		w := Walker{
			Enter: func(path Path, value reflect.Value, _ reflect.StructField, _ interface{}) (interface{}, error) {
				seen["enter"] = append(seen["enter"], value.Kind() == reflect.Pointer && value.IsNil())
				return nil, nil
			},
			Visit: func(path Path, value reflect.Value, _ reflect.StructField, _ interface{}) error {
				if path.String() == "Inner" {
					seen["visit"] = append(seen["visit"], value.Kind() == reflect.Pointer && value.IsNil())
				}
				return nil
			},
		}
		testingx.Call(t, w.Walk, &outer{}, nil).NoError()
		// Inner.A is a string, so it is not a nil pointer to either callback.
		testingx.Capture(t, seen["enter"]).Equal([]bool{false, false})
		testingx.Capture(t, seen["visit"]).Equal([]bool{false})
	})

	t.Run("enter_sees_a_skipped_pointer_still_nil", func(t *testing.T) {
		// Only the fields Walk recurses into are unpacked, so the cycle
		// back-reference reaches Enter as declared and stays that way.
		n := &node{Name: "root"}

		var sawNil []bool
		w := Walker{
			Enter: func(path Path, value reflect.Value, _ reflect.StructField, _ interface{}) (interface{}, error) {
				if path.String() == "Next" {
					sawNil = append(sawNil, value.IsNil())
				}
				return nil, nil
			},
		}
		testingx.Call(t, w.Walk, n, nil).NoError()
		testingx.Capture(t, sawNil).Equal([]bool{true})
		testingx.Capture(t, n.Next).Nil()
	})

	t.Run("data_reaches_children_and_own_visit", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct {
			Inner inner
			B     string
		}
		// Each Enter appends its own path to the data it was given, so the data
		// a Visit receives spells out the chain of Enters at and above it.
		var seen []string
		w := Walker{
			Enter: func(path Path, _ reflect.Value, _ reflect.StructField, data interface{}) (interface{}, error) {
				return fmt.Sprintf("%s>%s", data, path), nil
			},
			Visit: func(path Path, _ reflect.Value, _ reflect.StructField, data interface{}) error {
				seen = append(seen, fmt.Sprintf("%s=%s", path, data))
				return nil
			},
		}
		testingx.Call(t, w.Walk, &outer{}, "root").NoError()
		testingx.Capture(t, seen).Equal([]string{
			"Inner.A=root>Inner>Inner.A",
			"Inner=root>Inner",
			// B is Inner's sibling, so it derives from root, not from Inner.
			"B=root>B",
		})
	})

	t.Run("data_is_not_shared_between_sibling_subtrees", func(t *testing.T) {
		type inner struct{ Value string }
		type outer struct {
			A inner
			B inner
		}
		// Enter hands each field a fresh slice built from its parent's. If the
		// two subtrees shared one, B's children would see A's path too.
		var seen []string
		w := Walker{
			Enter: func(path Path, _ reflect.Value, _ reflect.StructField, data interface{}) (interface{}, error) {
				parent, _ := data.([]string)
				return append(append([]string{}, parent...), path.String()), nil
			},
			Visit: func(path Path, _ reflect.Value, _ reflect.StructField, data interface{}) error {
				seen = append(seen, fmt.Sprintf("%s=%v", path, data))
				return nil
			},
		}
		testingx.Call(t, w.Walk, &outer{}, nil).NoError()
		testingx.Capture(t, seen).Equal([]string{
			"A.Value=[A A.Value]",
			"A=[A]",
			"B.Value=[B B.Value]",
			"B=[B]",
		})
	})

	t.Run("child_error_skips_the_parent_visit", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct {
			Inner inner
			B     string
		}
		var tr trace
		err := tr.walker("Inner.A").Walk(&outer{}, nil)
		testingx.Capture(t, err).HasError("fail")
		// Inner's own Visit is skipped, and the walk carries on with B.
		testingx.Capture(t, tr.events).Equal([]string{
			"enter Inner", "enter Inner.A", "visit Inner.A", "enter B", "visit B",
		})
	})

	t.Run("enter_error_skips_children_and_visit", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct {
			Inner inner
			B     string
		}
		var events []string
		w := Walker{
			Enter: func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) (interface{}, error) {
				events = append(events, "enter "+path.String())
				if path.String() == "Inner" {
					return nil, fmt.Errorf("no entry")
				}
				return nil, nil
			},
			Visit: func(path Path, _ reflect.Value, _ reflect.StructField, _ interface{}) error {
				events = append(events, "visit "+path.String())
				return nil
			},
		}
		err := w.Walk(&outer{}, nil)
		testingx.Capture(t, err).HasError("no entry")
		// No children and no Visit for Inner itself, but the walk carries on
		// with the siblings.
		testingx.Capture(t, events).Equal([]string{"enter Inner", "enter B", "visit B"})
	})

	t.Run("nil_enter_is_a_no_op", func(t *testing.T) {
		// A Walker with only a Visit is what Walk itself builds, so it must
		// behave identically.
		type inner struct{ A string }
		type outer struct {
			Inner *inner
			B     string
		}
		var viaWalker, viaWalk []string
		testingx.Call(t, Walker{Visit: collectPaths(&viaWalker)}.Walk, &outer{}, nil).NoError()
		testingx.Call(t, Walk, &outer{}, nil, collectPaths(&viaWalk)).NoError()
		testingx.Capture(t, viaWalker).Equal([]string{"Inner.A", "Inner", "B"})
		testingx.Capture(t, viaWalker).Equal(viaWalk)
	})

	t.Run("zero_walker_does_nothing", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct{ Inner *inner }
		o := &outer{}
		testingx.Call(t, Walker{}.Walk, o, nil).NoError()
		// It still walks - nil pointers are initialized as ever - it just has
		// nowhere to report to.
		testingx.Capture(t, o.Inner).NotNil()
	})
}

func TestWalkPanics(t *testing.T) {
	t.Run("non_pointer", func(t *testing.T) {
		type s struct{ A string }
		testingx.Panics(t, nil, Walk, s{}, nil, noopWalk)
	})

	t.Run("nil_pointer", func(t *testing.T) {
		testingx.Panics(t, nil, Walk, (*struct{ A string })(nil), nil, noopWalk)
	})

	t.Run("pointer_to_non_struct", func(t *testing.T) {
		s := "hello"
		testingx.Panics(t, nil, Walk, &s, nil, noopWalk)
	})
}
