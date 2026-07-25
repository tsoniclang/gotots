package emit

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type targetBinding struct {
	name         string
	sourceFile   *ast.File
	modulePath   string
	moduleExport bool
}

type nameOwner struct {
	byObject map[types.Object]targetBinding
}

func newNameOwner() *nameOwner {
	return &nameOwner{byObject: make(map[types.Object]targetBinding)}
}

func (n *nameOwner) Reserve(
	object types.Object,
	sourceFile *ast.File,
	modulePath string,
) (string, error) {
	if sourceFile == nil {
		return "", &api.NameError{Name: objectName(object), Reason: "source file is nil"}
	}
	if modulePath == "" {
		return "", &api.NameError{Name: objectName(object), Reason: "target module path is empty"}
	}
	return n.declare(object, targetBinding{
		name:         objectName(object),
		sourceFile:   sourceFile,
		modulePath:   modulePath,
		moduleExport: true,
	})
}

func (n *nameOwner) declare(object types.Object, binding targetBinding) (string, error) {
	if object == nil {
		return "", &api.NameError{Reason: "declaration object is nil"}
	}
	if existing, ok := n.byObject[object]; ok {
		if binding.sourceFile != nil &&
			(existing.sourceFile != binding.sourceFile ||
				existing.modulePath != binding.modulePath) {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "declaration was reserved by a different target module",
			}
		}
		return existing.name, nil
	}
	if object.Name() == "" {
		return "", &api.NameError{Reason: "declaration name is empty"}
	}
	binding.name = object.Name()
	n.byObject[object] = binding
	return object.Name(), nil
}

type fileNames struct {
	owner        *nameOwner
	sourceFile   *ast.File
	packageScope *types.Scope
	placement    *placementOwner
}

func (n *nameOwner) ForFile(
	sourceFile *ast.File,
	packageScope *types.Scope,
	placement *placementOwner,
) api.Names {
	return &fileNames{
		owner:        n,
		sourceFile:   sourceFile,
		packageScope: packageScope,
		placement:    placement,
	}
}

func (n *fileNames) Declare(object types.Object) (string, error) {
	if object != nil && object.Parent() == n.packageScope {
		binding, ok := n.owner.byObject[object]
		if !ok {
			return "", &api.NameError{
				Name:   object.Name(),
				Reason: "package declaration was not reserved",
			}
		}
		return binding.name, nil
	}
	return n.owner.declare(object, targetBinding{})
}

func (n *fileNames) Reference(object types.Object) (string, error) {
	if object == nil {
		return "", &api.NameError{Reason: "reference object is nil"}
	}
	binding, ok := n.owner.byObject[object]
	if !ok {
		return "", &api.NameError{Name: object.Name(), Reason: "object has no emitted declaration"}
	}
	if binding.sourceFile != nil && binding.sourceFile != n.sourceFile {
		return n.placement.ValueImport(binding.modulePath, binding.name)
	}
	return binding.name, nil
}

func (n *fileNames) ModuleExport(object types.Object) (bool, error) {
	if object == nil {
		return false, &api.NameError{Reason: "declaration object is nil"}
	}
	binding, ok := n.owner.byObject[object]
	if !ok {
		return false, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	return binding.moduleExport, nil
}

func objectName(object types.Object) string {
	if object == nil {
		return ""
	}
	return object.Name()
}
