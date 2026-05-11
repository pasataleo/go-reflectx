package reflectx

import (
	"reflect"

	"github.com/pasataleo/go-testingx/pkg/diff"
	"github.com/pasataleo/go-testingx/pkg/render"
)

var (
	_ diff.Differ[reflect.Type]      = (*TypeDiffer)(nil)
	_ diff.Differ[reflect.Value]     = (*ValueDiffer)(nil)
	_ render.Renderer[reflect.Type]  = (*TypeRenderer)(nil)
	_ render.Renderer[reflect.Value] = (*ValueRenderer)(nil)
)

var WithTypeDiffer diff.OptsFn = func(opts *diff.Opts) {
	diff.WithDiffer(new(TypeDiffer))(opts)
}

type TypeDiffer struct{}

func (t *TypeDiffer) Diff(got, want reflect.Type, opts *diff.Opts) diff.Result {
	return &typeResult{got: got, want: want}
}

var WithTypeRenderer render.OptsFn = func(opts *render.Opts) {
	render.WithRenderer(new(TypeRenderer))(opts)
}

type TypeRenderer struct{}

func (t *TypeRenderer) Render(value reflect.Type, opts *render.Opts) string {
	return value.String()
}

var WithValueDiffer diff.OptsFn = func(opts *diff.Opts) {
	diff.WithDiffer(new(ValueDiffer))(opts)
}

type ValueDiffer struct{}

func (v *ValueDiffer) Diff(got, want reflect.Value, opts *diff.Opts) diff.Result {
	return diff.Of(got, want, opts)
}

var WithValueRenderer render.OptsFn = func(opts *render.Opts) {
	render.WithRenderer(new(ValueRenderer))(opts)
}

type ValueRenderer struct{}

func (v *ValueRenderer) Render(value reflect.Value, opts *render.Opts) string {
	return render.Render(value, opts)
}

// typeResult implements diff.Result for reflect.Type comparisons.
type typeResult struct {
	got  reflect.Type
	want reflect.Type
}

func (r *typeResult) Status() diff.Status {
	if r.got == r.want {
		return diff.StatusUnchanged
	}
	return diff.StatusChanged
}

func (r *typeResult) RenderGot(opts *render.Opts) string {
	return r.got.String()
}

func (r *typeResult) RenderWant(opts *render.Opts) string {
	return r.want.String()
}

func (r *typeResult) RenderDiff(opts *diff.Opts) string {
	if r.Status() == diff.StatusUnchanged {
		return "  " + r.got.String()
	}
	return opts.RenderOpts().Printf("{red}-{reset} %s\n{green}+{reset} %s", r.want.String(), r.got.String())
}
