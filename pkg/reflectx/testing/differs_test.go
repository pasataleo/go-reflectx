package testing

import (
	"reflect"
	"testing"

	"github.com/pasataleo/go-testingx/pkg/diff"
	"github.com/pasataleo/go-testingx/pkg/mocks"
	"github.com/pasataleo/go-testingx/pkg/testingx"
)

type holder struct {
	Type reflect.Type
}

func TestTypeDiffer(t *testing.T) {
	t.Run("same type", func(t *testing.T) {
		mock := &mocks.T{T: t}
		testingx.Capture(mock, holder{Type: reflect.TypeFor[string]()}).Equal(holder{Type: reflect.TypeFor[string]()}, WithTypeDiffer)

		if mock.ErrorMessage != "" || mock.FatalMessage != "" {
			t.Errorf("unexpected failure: %s%s", mock.ErrorMessage, mock.FatalMessage)
		}
	})

	t.Run("different type", func(t *testing.T) {
		mock := &mocks.T{T: t}
		testingx.Capture(mock, holder{Type: reflect.TypeFor[string]()}).Equal(holder{Type: reflect.TypeFor[int]()}, WithTypeDiffer)

		if mock.ErrorMessage == "" {
			t.Error("expected a failure for different types")
		}
	})

	t.Run("result", func(t *testing.T) {
		opts := diff.NewOpts(t)
		result := new(TypeDiffer).Diff(reflect.TypeFor[string](), reflect.TypeFor[int](), opts)
		testingx.Call(t, result.Status).Equal(diff.StatusChanged)
		testingx.Call(t, result.RenderGot, opts.RenderOpts()).Equal("string")
	})
}
