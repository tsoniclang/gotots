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
	if object.Pkg() == nil ||
		(object.Parent() != object.Pkg().Scope() && !isDeclaredMethod(object)) {
		return Root{}, &RootError{
			Object: object.Name(),
			Reason: "object is not a package declaration",
		}
	}
	return Root{object: object}, nil
}

func isDeclaredMethod(object types.Object) bool {
	method, ok := object.(*types.Func)
	if !ok {
		return false
	}
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	receiverType := signature.Recv().Type()
	if pointer, ok := types.Unalias(receiverType).(*types.Pointer); ok {
		receiverType = pointer.Elem()
	}
	named, ok := types.Unalias(receiverType).(*types.Named)
	return ok && named.Obj() != nil && named.Obj().Pkg() == object.Pkg()
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
		typeName, ok := object.(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := types.Unalias(typeName.Type()).(*types.Named)
		if !ok {
			continue
		}
		for index := range named.NumMethods() {
			method := named.Method(index)
			if !method.Exported() {
				continue
			}
			methodRoot, err := NewRoot(method)
			if err != nil {
				return nil, err
			}
			roots = append(roots, methodRoot)
		}
	}
	sort.Slice(roots, func(left, right int) bool {
		return compareObjects(roots[left].object, roots[right].object) < 0
	})
	return roots, nil
}
