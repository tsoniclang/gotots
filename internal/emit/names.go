package emit

import (
	"go/ast"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type targetBinding struct {
	name         string
	sourceFile   *ast.File
	modulePath   string
	moduleExport bool
}

type nameOwner struct {
	byObject     map[types.Object]targetBinding
	namesByScope map[*types.Scope]map[string]types.Object
}

func newNameOwner() *nameOwner {
	return &nameOwner{
		byObject:     make(map[types.Object]targetBinding),
		namesByScope: make(map[*types.Scope]map[string]types.Object),
	}
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
	name, err := n.allocate(object)
	if err != nil {
		return "", err
	}
	binding.name = name
	n.byObject[object] = binding
	return name, nil
}

func (n *nameOwner) allocate(object types.Object) (string, error) {
	scope := object.Parent()
	base := object.Name()
	for suffix := uint64(0); ; suffix++ {
		candidate := base
		if suffix != 0 {
			candidate += "$" + strconv.FormatUint(suffix, 10)
		}
		if n.nameExists(scope, candidate) {
			continue
		}
		if scope != nil {
			names := n.namesByScope[scope]
			if names == nil {
				names = make(map[string]types.Object)
				n.namesByScope[scope] = names
			}
			names[candidate] = object
		}
		return candidate, nil
	}
}

func (n *nameOwner) nameExists(scope *types.Scope, name string) bool {
	for current := scope; current != nil; current = current.Parent() {
		if n.namesByScope[current][name] != nil {
			return true
		}
	}
	return false
}

type fileNames struct {
	owner        *nameOwner
	sourceFile   *ast.File
	packageScope *types.Scope
	factory      tsgo.Factory
	temporaries  map[api.TemporaryKind]uint64
}

func (n *nameOwner) ForFile(
	sourceFile *ast.File,
	packageScope *types.Scope,
	factory tsgo.Factory,
) api.Names {
	return &fileNames{
		owner:        n,
		sourceFile:   sourceFile,
		packageScope: packageScope,
		factory:      factory,
		temporaries:  make(map[api.TemporaryKind]uint64),
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

func (n *fileNames) Reference(object types.Object) (api.NameReference, error) {
	if object == nil {
		return api.NameReference{}, &api.NameError{Reason: "reference object is nil"}
	}
	binding, ok := n.owner.byObject[object]
	if !ok {
		return api.NameReference{}, &api.NameError{
			Name:   object.Name(),
			Reason: "object has no emitted declaration",
		}
	}
	if binding.sourceFile != nil && binding.sourceFile != n.sourceFile {
		request, err := api.NewImportRequest(
			n.factory,
			api.ImportPhaseValue,
			binding.modulePath,
			binding.name,
			binding.name,
		)
		if err != nil {
			return api.NameReference{}, err
		}
		return api.NewNameReference(binding.name, request)
	}
	return api.NewNameReference(binding.name)
}

func (n *fileNames) TypeImport(
	modulePath string,
	exportedName string,
) (api.NameReference, error) {
	request, err := api.NewImportRequest(
		n.factory,
		api.ImportPhaseType,
		modulePath,
		exportedName,
		exportedName,
	)
	if err != nil {
		return api.NameReference{}, err
	}
	return api.NewNameReference(exportedName, request)
}

func (n *fileNames) Temporary(kind api.TemporaryKind) (string, error) {
	prefix, err := api.TemporaryPrefix(kind)
	if err != nil {
		return "", err
	}
	index := n.temporaries[kind]
	n.temporaries[kind] = index + 1
	return prefix + strconv.FormatUint(index, 10), nil
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
