package api

import "go/types"

type GoRuntimeType uint8

const (
	GoRuntimeTypeInvalid GoRuntimeType = iota
	GoRuntimeTypeBuiltinError
	GoRuntimeTypeError
	GoRuntimeTypePanicNilError
	GoRuntimeTypePanicNilPointer
)

func (k GoRuntimeType) Valid() bool {
	return k == GoRuntimeTypeBuiltinError ||
		k == GoRuntimeTypeError ||
		k == GoRuntimeTypePanicNilError ||
		k == GoRuntimeTypePanicNilPointer
}

type GoRuntimeContract interface {
	Owns(*types.Package) bool
	Classify(types.Type) GoRuntimeType
}

func (c Context) WithGoRuntimeContract(contract GoRuntimeContract) Context {
	if contract == nil {
		panic("Go runtime contract is nil")
	}
	c.goRuntime = contract
	return c
}

func (c Context) GoRuntimeType(sourceType types.Type) GoRuntimeType {
	if c.goRuntime == nil {
		return GoRuntimeTypeInvalid
	}
	return c.goRuntime.Classify(sourceType)
}
