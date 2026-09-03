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

// trace records a Walker's callbacks in the order they fire, so a test can
// assert the enter -> children -> visit -> leave contract directly.
type trace struct {
	events []string
}

// walker returns a Walker that records every callback into the trace. failAt,
// when non-empty, is the path whose Visit returns an error.
func (tr *trace) walker(failAt string) Walker {
	return Walker{
		Enter: func(path Path, _ reflect.Value, _ reflect.StructField) (func(), error) {
			tr.events = append(tr.events, "enter "+path.String())
			return func() { tr.events = append(tr.events, "leave "+path.String()) }, nil
		},
		Visit: func(path Path, _ reflect.Value, _ reflect.StructField) error {
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
		testingx.Call(t, tr.walker("").Walk, &outer{}).NoError()
		testingx.Capture(t, tr.events).Equal([]string{
			"enter Inner",
			"visit Inner.A",
			"visit Inner.B",
			"visit Inner",
			"leave Inner",
			"visit C",
		})
	})

	t.Run("leave_runs_after_visit", func(t *testing.T) {
		// The same contract stated on its own, because it is the half callers
		// depend on: whatever Enter opened is still open while Visit runs, so a
		// Visit that acts on the field itself sees the subtree's state.
		type inner struct{ A string }
		type outer struct{ Inner inner }
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &outer{}).NoError()
		testingx.Capture(t, tr.events).Equal([]string{"enter Inner", "visit Inner.A", "visit Inner", "leave Inner"})
	})

	t.Run("enter_not_called_for_non_struct_fields", func(t *testing.T) {
		type s struct {
			A string
			B int
			C bool
		}
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &s{}).NoError()
		testingx.Capture(t, tr.events).Equal([]string{"visit A", "visit B", "visit C"})
	})

	t.Run("enter_not_called_for_seen_types", func(t *testing.T) {
		// Enter mirrors walk's own recursion condition, so the back-reference
		// gets no Enter while the fresh nested struct beside it does.
		type leaf struct{ Value string }
		type selfish struct {
			Next *selfish
			Leaf leaf
		}
		var tr trace
		testingx.Call(t, tr.walker("").Walk, &selfish{}).NoError()
		testingx.Capture(t, tr.events).Equal([]string{
			"visit Next",
			"enter Leaf",
			"visit Leaf.Value",
			"visit Leaf",
			"leave Leaf",
		})
	})

	t.Run("enter_called_after_nil_pointer_init", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct{ Inner *inner }
		o := &outer{}
		testingx.Capture(t, o.Inner).Nil()

		var entered []bool
		w := Walker{
			Enter: func(_ Path, value reflect.Value, _ reflect.StructField) (func(), error) {
				entered = append(entered, value.IsNil())
				return nil, nil
			},
		}
		testingx.Call(t, w.Walk, o).NoError()
		// Enter saw the pointer already allocated, not the nil it was declared
		// as - and it received the field itself, so IsNil is a legal question.
		testingx.Capture(t, entered).Equal([]bool{false})
		testingx.Capture(t, o.Inner).NotNil()
	})

	t.Run("leave_runs_when_a_child_errors", func(t *testing.T) {
		// The parent's Visit is skipped, exactly as it is without a Walker, but
		// the leave still runs. Anything Enter opened is closed regardless of
		// which way the walk left the field.
		type inner struct{ A string }
		type outer struct {
			Inner inner
			B     string
		}
		var tr trace
		err := tr.walker("Inner.A").Walk(&outer{})
		testingx.Capture(t, err).HasError("fail")
		testingx.Capture(t, tr.events).Equal([]string{"enter Inner", "visit Inner.A", "leave Inner", "visit B"})
	})

	t.Run("enter_error_skips_recursion_but_still_leaves", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct {
			Inner inner
			B     string
		}
		var events []string
		w := Walker{
			Enter: func(path Path, _ reflect.Value, _ reflect.StructField) (func(), error) {
				events = append(events, "enter "+path.String())
				return func() { events = append(events, "leave "+path.String()) }, fmt.Errorf("no entry")
			},
			Visit: func(path Path, _ reflect.Value, _ reflect.StructField) error {
				events = append(events, "visit "+path.String())
				return nil
			},
		}
		err := w.Walk(&outer{})
		testingx.Capture(t, err).HasError("no entry")
		// No children, no Visit for the field itself, but a leave for the Enter
		// that returned one alongside its error - and the walk carries on with
		// the siblings.
		testingx.Capture(t, events).Equal([]string{"enter Inner", "leave Inner", "visit B"})
	})

	t.Run("leave_runs_when_visit_panics", func(t *testing.T) {
		// The leave is a defer, so it survives a panic out of a callback. That
		// is what lets a caller keep the state Enter opened balanced even when
		// a Visit reports a programmer error the loud way.
		type inner struct{ A string }
		type outer struct{ Inner inner }
		var events []string
		w := Walker{
			Enter: func(path Path, _ reflect.Value, _ reflect.StructField) (func(), error) {
				return func() { events = append(events, "leave "+path.String()) }, nil
			},
			Visit: func(path Path, _ reflect.Value, _ reflect.StructField) error {
				panic("boom")
			},
		}
		testingx.Panics(t, nil, w.Walk, &outer{}).Contains("boom")
		testingx.Capture(t, events).Equal([]string{"leave Inner"})
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
		testingx.Call(t, Walker{Visit: collectPaths(&viaWalker)}.Walk, &outer{}).NoError()
		testingx.Call(t, Walk, &outer{}, collectPaths(&viaWalk)).NoError()
		testingx.Capture(t, viaWalker).Equal([]string{"Inner.A", "Inner", "B"})
		testingx.Capture(t, viaWalker).Equal(viaWalk)
	})

	t.Run("zero_walker_does_nothing", func(t *testing.T) {
		type inner struct{ A string }
		type outer struct{ Inner *inner }
		o := &outer{}
		testingx.Call(t, Walker{}.Walk, o).NoError()
		// It still walks - nil pointers are initialized as ever - it just has
		// nowhere to report to.
		testingx.Capture(t, o.Inner).NotNil()
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
