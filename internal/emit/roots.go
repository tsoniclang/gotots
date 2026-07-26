package emit

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/load"
)

type Root struct {
	object types.Object
}

type RootError struct {
	Object string
	Reason string
}

func (e *RootError) Error() string {
	if e.Object == "" {
		return "create emission root: " + e.Reason
	}
	return fmt.Sprintf("create emission root for %q: %s", e.Object, e.Reason)
}

func NewRoot(object types.Object) (Root, error) {
	if object == nil {
		return Root{}, &RootError{Reason: "object is nil"}
	}
	if object.Pkg() == nil || object.Parent() != object.Pkg().Scope() {
		return Root{}, &RootError{
			Object: object.Name(),
			Reason: "object is not a package declaration",
		}
	}
	return Root{object: object}, nil
}

func (r Root) Object() types.Object {
	return r.object
}

func ExportedAPIRoots(source *load.Package) ([]Root, error) {
	if source == nil || source.Types() == nil {
		return nil, &RootError{Reason: "source package is nil"}
	}
	scope := source.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	roots := make([]Root, 0, len(names))
	for _, name := range names {
		object := scope.Lookup(name)
		if object == nil || !object.Exported() {
			continue
		}
		root, err := NewRoot(object)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	return roots, nil
}
