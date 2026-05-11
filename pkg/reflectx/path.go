package reflectx

import (
	"slices"
	"strings"
)

// Path represents a field's location within a struct hierarchy as a sequence
// of field names. For example, a field Host inside a Database struct has the
// path ["Database", "Host"].
type Path []string

// Append returns a new Path with name appended, without modifying the original.
func (path Path) Append(name string) Path {
	return append(slices.Clone(path), name)
}

// String returns the dot-separated path, e.g. "Database.Host".
func (path Path) String() string {
	return strings.Join(path, ".")
}
