package maprepresentation

import (
	"go/types"

	gostdlib "github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/emit/api"
)

type staticSpecializationNames struct{}

func (staticSpecializationNames) Declare(types.Object) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Parameter(*types.Var, int) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Result(*types.Var, int) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) Reference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) DefinedValueRepresentation(
	*types.TypeName,
) (api.DefinedValueRepresentation, error) {
	return api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedWrapper,
		api.NameReference{},
	)
}

func (staticSpecializationNames) TypeReference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) PackageVariable(
	*types.Var,
) (api.PackageVariableReference, error) {
	panic("unused")
}

func (staticSpecializationNames) NamedStructConstructor(
	*types.TypeName,
) (api.NameReference, error) {
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
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceAdapter(
	types.Type,
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceContractDemand(
	types.Type,
	types.Type,
) ([]api.RootRequest, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceDynamicType(
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceType(
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceContract(
	types.Type,
) (api.InterfaceContractReference, error) {
	panic("unused")
}

func (staticSpecializationNames) RecoveryCallable(
	*types.Func,
) (api.RecoveryCallableReference, bool, error) {
	panic("unused")
}

func (staticSpecializationNames) DeferredCallable(
	*types.Func,
	string,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) DeferredCallableRegistry(
	*types.Signature,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) MethodTarget(
	*types.Func,
) (api.MethodTarget, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceMethodName(
	*types.Func,
) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceMethodCallable(
	*types.Func,
) (api.InterfaceMethodCallableReference, error) {
	panic("unused")
}

func (staticSpecializationNames) InterfaceMethodToken(
	*types.Func,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) GenericCapability(
	api.GenericOperationSelection,
	*types.Signature,
) (api.GenericCapabilityReference, error) {
	panic("unused")
}

func (staticSpecializationNames) CallableABI(
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (staticSpecializationNames) SourceCallableABI(
	types.Object,
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (staticSpecializationNames) ProviderGenericTypeArguments(
	*types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	return nil, false, nil
}

func (staticSpecializationNames) ProviderInterface(
	types.Type,
) (gostdlib.ProviderInterface, bool, error) {
	return gostdlib.ProviderInterface{}, false, nil
}

func (staticSpecializationNames) ProviderInterfaceBridge(
	types.Type,
) (api.NameReference, bool, error) {
	return api.NameReference{}, false, nil
}

func (staticSpecializationNames) ProviderCallableProfile(
	*types.Func,
	string,
) (api.ProviderCallableProfileReference, bool, error) {
	return api.ProviderCallableProfileReference{}, false, nil
}

func (staticSpecializationNames) ProviderCallableProfileCandidates(
	*types.Func,
) ([]api.ProviderCallableProfileCandidate, bool, error) {
	return nil, false, nil
}

func (staticSpecializationNames) ProviderCallableParameters(
	*types.Func,
) ([]gostdlib.ProviderCallableParameterDocument, bool, error) {
	return nil, false, nil
}

func (staticSpecializationNames) ProviderStatefulProfileCandidates(
	*types.TypeName,
) ([]api.ProviderStatefulProfileCandidate, bool, error) {
	return nil, false, nil
}

func (staticSpecializationNames) ProviderStatefulProfileTarget(
	*types.TypeName,
	string,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (staticSpecializationNames) ProviderRepresentationOwnsMethod(
	types.Type,
	*types.Func,
) (bool, error) {
	return false, nil
}

func (staticSpecializationNames) ConstantValue(
	*types.Const,
) (api.NameReference, bool, error) {
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
	switch symbol {
	case api.RuntimePanic:
		return api.NewNameReference("GoPanic")
	case api.RuntimeDenseIndex:
		return api.NewNameReference("GoDenseIndex")
	default:
		panic("unexpected runtime symbol")
	}
}

func (staticSpecializationNames) Temporary(api.TemporaryKind) (string, error) {
	panic("unused")
}

func (staticSpecializationNames) ModuleExport(types.Object) (bool, error) {
	panic("unused")
}
