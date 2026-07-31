package contract

import (
	"go/token"
	"go/types"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
)

type MethodKind uint8

const (
	MethodInvalid MethodKind = iota
	MethodError
	MethodRuntimeError
)

var (
	errorMethod        = canonicalErrorMethod()
	runtimeErrorMethod = types.NewFunc(
		token.NoPos,
		types.NewPackage("runtime", "runtime"),
		"RuntimeError",
		types.NewSignatureType(nil, nil, nil, nil, nil, false),
	)
)

func Method(method *types.Func) MethodKind {
	switch {
	case environmentcontract.EquivalentMethods(method, errorMethod):
		return MethodError
	case environmentcontract.EquivalentMethods(method, runtimeErrorMethod):
		return MethodRuntimeError
	default:
		return MethodInvalid
	}
}

func CanonicalMethod(kind MethodKind) (*types.Func, bool) {
	switch kind {
	case MethodError:
		return errorMethod, true
	case MethodRuntimeError:
		return runtimeErrorMethod, true
	default:
		return nil, false
	}
}

func canonicalErrorMethod() *types.Func {
	errorObject := types.Universe.Lookup("error")
	if errorObject == nil {
		panic("predeclared error contract is absent")
	}
	errorInterface, ok := errorObject.Type().Underlying().(*types.Interface)
	if !ok || errorInterface.Complete().NumMethods() != 1 {
		panic("predeclared error contract is invalid")
	}
	return errorInterface.Complete().Method(0)
}
