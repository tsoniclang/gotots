package maprepresentation

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
)

type staticSpecializationNames struct{}

func (staticSpecializationNames) Declare(types.Object) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Parameter(*types.Var, int) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Reference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) TypeReference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) PackageVariable(
	*types.Var,
) (api.PackageVariableReference, error) {
	panic("unused")
}

func (staticSpecializationNames) NamedStructOperation(
	*types.TypeName,
	api.NamedStructOperation,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) NamedStructStorage(
	*types.TypeName,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) AnonymousStructStorage(
	*types.Struct,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) AnonymousStruct(
	*types.Struct,
	api.AnonymousStructDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) ConstantProjection(
	*types.Const,
	types.BasicKind,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) Member(*types.Var) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Primitive(
	api.PrimitiveAlias,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) Runtime(
	symbol api.RuntimeSymbol,
	_ api.ImportPhase,
) (api.NameReference, error) {
	if symbol != api.RuntimePanic {
		panic("unexpected runtime symbol")
	}
	return api.NewNameReference("GoPanic")
}

func (staticSpecializationNames) Temporary(api.TemporaryKind) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) ModuleExport(types.Object) (bool, error) {
	panic("unused")
}
