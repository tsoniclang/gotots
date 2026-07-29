package gocontract

import (
	"fmt"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	interfacecontract "github.com/tsoniclang/gotots/internal/emit/runtime/interfacevalue/contract"
	"github.com/tsoniclang/gotots/internal/load"
)

type Contract struct {
	runtimePackage  *types.Package
	errorType       *types.Named
	panicNilType    *types.Named
	panicNilPointer *types.Pointer
}

type Error struct {
	Reason string
}

func (e *Error) Error() string {
	return "resolve selected Go runtime contract: " + e.Reason
}

func Resolve(program *load.Program) (*Contract, error) {
	if program == nil {
		return nil, &Error{Reason: "program is nil"}
	}
	runtimePackage, err := selectedRuntime(program)
	if err != nil {
		return nil, err
	}
	if runtimePackage == nil {
		return &Contract{}, nil
	}
	errorType, err := namedType(runtimePackage, "Error")
	if err != nil {
		return nil, err
	}
	errorInterface, ok := errorType.Underlying().(*types.Interface)
	if !ok {
		return nil, &Error{Reason: "runtime.Error is not an interface"}
	}
	if err := verifyMethods(
		"runtime.Error",
		types.NewMethodSet(errorInterface.Complete()),
	); err != nil {
		return nil, err
	}
	panicNilType, err := namedType(runtimePackage, "PanicNilError")
	if err != nil {
		return nil, err
	}
	if _, ok := panicNilType.Underlying().(*types.Struct); !ok {
		return nil, &Error{Reason: "runtime.PanicNilError is not a struct"}
	}
	panicNilPointer := types.NewPointer(panicNilType)
	if err := verifyMethods(
		"*runtime.PanicNilError",
		types.NewMethodSet(panicNilPointer),
	); err != nil {
		return nil, err
	}
	return &Contract{
		runtimePackage:  runtimePackage,
		errorType:       errorType,
		panicNilType:    panicNilType,
		panicNilPointer: panicNilPointer,
	}, nil
}

func (c *Contract) Owns(sourcePackage *types.Package) bool {
	return c != nil &&
		c.runtimePackage != nil &&
		sourcePackage == c.runtimePackage
}

func (c *Contract) Classify(sourceType types.Type) api.GoRuntimeType {
	if sourceType == nil {
		return api.GoRuntimeTypeInvalid
	}
	sourceType = types.Unalias(sourceType)
	if errorObject := types.Universe.Lookup("error"); errorObject != nil &&
		types.Identical(sourceType, errorObject.Type()) {
		return api.GoRuntimeTypeBuiltinError
	}
	if c == nil {
		return api.GoRuntimeTypeInvalid
	}
	switch {
	case c.errorType != nil && types.Identical(sourceType, c.errorType):
		return api.GoRuntimeTypeError
	case c.panicNilType != nil && types.Identical(sourceType, c.panicNilType):
		return api.GoRuntimeTypePanicNilError
	case c.panicNilPointer != nil &&
		types.Identical(sourceType, c.panicNilPointer):
		return api.GoRuntimeTypePanicNilPointer
	default:
		return api.GoRuntimeTypeInvalid
	}
}

func selectedRuntime(program *load.Program) (*types.Package, error) {
	seen := make(map[*types.Package]struct{})
	var runtimePackage *types.Package
	var visit func(*types.Package) error
	visit = func(current *types.Package) error {
		if current == nil {
			return &Error{Reason: "type graph contains a nil package"}
		}
		if _, visited := seen[current]; visited {
			return nil
		}
		seen[current] = struct{}{}
		if current.Path() == "runtime" {
			if runtimePackage != nil && runtimePackage != current {
				return &Error{Reason: "type graph contains multiple runtime packages"}
			}
			runtimePackage = current
		}
		for _, imported := range current.Imports() {
			if err := visit(imported); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range program.Roots() {
		if root == nil || root.Types() == nil {
			return nil, &Error{Reason: "program root has no type package"}
		}
		if err := visit(root.Types()); err != nil {
			return nil, err
		}
	}
	return runtimePackage, nil
}

func namedType(
	sourcePackage *types.Package,
	name string,
) (*types.Named, error) {
	object, ok := sourcePackage.Scope().Lookup(name).(*types.TypeName)
	if !ok {
		return nil, &Error{Reason: fmt.Sprintf("runtime.%s is absent", name)}
	}
	named, ok := types.Unalias(object.Type()).(*types.Named)
	if !ok || named.Obj() != object {
		return nil, &Error{
			Reason: fmt.Sprintf("runtime.%s has no canonical named type", name),
		}
	}
	return named, nil
}

func verifyMethods(name string, set *types.MethodSet) error {
	if set == nil || set.Len() != 2 {
		return &Error{
			Reason: fmt.Sprintf("%s has %d methods, want 2", name, methodCount(set)),
		}
	}
	seen := map[interfacecontract.MethodKind]bool{}
	for index := range set.Len() {
		method, ok := set.At(index).Obj().(*types.Func)
		if !ok {
			return &Error{Reason: name + " contains a non-method"}
		}
		kind := interfacecontract.Method(method)
		if kind == interfacecontract.MethodInvalid || seen[kind] {
			return &Error{Reason: name + " has a foreign or duplicate method contract"}
		}
		seen[kind] = true
	}
	if !seen[interfacecontract.MethodError] ||
		!seen[interfacecontract.MethodRuntimeError] {
		return &Error{Reason: name + " does not satisfy the runtime error contract"}
	}
	return nil
}

func methodCount(set *types.MethodSet) int {
	if set == nil {
		return 0
	}
	return set.Len()
}
