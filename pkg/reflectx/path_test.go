package reflectx

import (
	"testing"

	"github.com/pasataleo/go-testingx/pkg/testingx"
)

func TestPathAppend(t *testing.T) {
	original := Path{"a", "b"}
	testingx.Call(t, original.Append, "c").Equal(Path{"a", "b", "c"})

	// Verify the original was not mutated.
	testingx.Capture(t, len(original)).Equal(2)
	testingx.Capture(t, original[0]).Equal("a")
	testingx.Capture(t, original[1]).Equal("b")
}

func TestPathString(t *testing.T) {
	tcs := map[string]struct {
		path Path
		want string
	}{
		"empty": {
			path: Path{},
			want: "",
		},
		"single": {
			path: Path{"Host"},
			want: "Host",
		},
		"multiple": {
			path: Path{"Database", "Host"},
			want: "Database.Host",
		},
		"deep": {
			path: Path{"A", "B", "C", "D"},
			want: "A.B.C.D",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			testingx.Call(t, tc.path.String).Equal(tc.want)
		})
	}
}
