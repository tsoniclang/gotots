package command

import (
	"fmt"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/config"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func selectRoots(program *load.Program, mode config.RootMode) ([]emit.Root, error) {
	rootPackage := program.Roots()[0]
	switch mode {
	case config.RootModeMain:
		object := rootPackage.Types().Scope().Lookup("main")
		if object == nil {
			return nil, commandError("select roots", "selected package has no main")
		}
		root, err := emit.NewRoot(object)
		if err != nil {
			return nil, err
		}
		return []emit.Root{root}, nil
	case config.RootModeExported:
		return emit.ExportedAPIRoots(rootPackage)
	case config.RootModePackage:
		return packageRoots(rootPackage)
	case config.RootModeAll:
		var roots []emit.Root
		for _, sourcePackage := range program.Packages() {
			selected, err := packageRoots(sourcePackage)
			if err != nil {
				return nil, err
			}
			roots = append(roots, selected...)
		}
		return roots, nil
	default:
		return nil, commandError("select roots", fmt.Sprintf("mode %q is invalid", mode))
	}
}

func packageRoots(sourcePackage *load.Package) ([]emit.Root, error) {
	names := sourcePackage.Types().Scope().Names()
	sort.Strings(names)
	var roots []emit.Root
	for _, name := range names {
		object := sourcePackage.Types().Scope().Lookup(name)
		if constant, ok := object.(*types.Const); ok && isUntyped(constant.Type()) {
			continue
		}
		selected, err := objectRoots(object)
		if err != nil {
			return nil, err
		}
		roots = append(roots, selected...)
	}
	return roots, nil
}

func objectRoots(object types.Object) ([]emit.Root, error) {
	root, err := emit.NewRoot(object)
	if err != nil {
		return nil, err
	}
	result := []emit.Root{root}
	typeName, ok := object.(*types.TypeName)
	if !ok {
		return result, nil
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok {
		return result, nil
	}
	for index := range named.NumMethods() {
		method, err := emit.NewRoot(named.Method(index))
		if err != nil {
			return nil, err
		}
		result = append(result, method)
	}
	return result, nil
}

func isUntyped(source types.Type) bool {
	basic, ok := types.Unalias(source).(*types.Basic)
	return ok && basic.Info()&types.IsUntyped != 0
}
