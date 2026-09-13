package binary

import (
	"go/ast"
	"go/token"
	"go/types"

	gostdlib "github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type unusedNames struct{}

func (unusedNames) Declare(types.Object) (string, error) {
	panic("unused")
}

func (unusedNames) Parameter(*types.Var, int) (string, error) {
	panic("unused")
}

func (unusedNames) Result(*types.Var, int) (string, error) {
	panic("unused")
}

func (unusedNames) Reference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) DefinedValueRepresentation(
	*types.TypeName,
) (api.DefinedValueRepresentation, error) {
	return api.NewDefinedValueRepresentation(
		api.DefinedValueRepresentationGeneratedWrapper,
		api.NameReference{},
	)
}

func (unusedNames) ProviderOwnedDeclaration(types.Object) (bool, error)    { return false, nil }
func (unusedNames) EnvironmentOwnedDeclaration(types.Object) (bool, error) { return false, nil }

func (unusedNames) ProviderNamedStructOperationSelected(*types.TypeName, api.NamedStructOperation) (bool, error) {
	return false, nil
}

func (unusedNames) TypeReference(types.Object) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) AnonymousStruct(
	*types.Struct,
	api.AnonymousStructDemand,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) MapSpecialization(
	types.Type,
	api.MapSpecializationDemand,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceAdapter(
	types.Type,
	types.Type,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceContractDemand(
	types.Type,
	types.Type,
) ([]api.RootRequest, error) {
	panic("unused")
}

func (unusedNames) InterfaceDynamicType(types.Type) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceType(types.Type) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceContract(
	types.Type,
) (api.InterfaceContractReference, error) {
	panic("unused")
}

func (unusedNames) RecoveryCallable(
	*types.Func,
) (api.RecoveryCallableReference, bool, error) {
	panic("unused")
}

func (unusedNames) DeferredCallable(
	*types.Func,
	string,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) DeferredCallableRegistry(
	*types.Signature,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) MethodTarget(*types.Func) (api.MethodTarget, error) {
	panic("unused")
}

func (unusedNames) InterfaceMethodName(*types.Func) (string, error) {
	panic("unused")
}

func (unusedNames) InterfaceMethodCallable(
	*types.Func,
) (api.InterfaceMethodCallableReference, error) {
	panic("unused")
}

func (unusedNames) InterfaceMethodToken(
	*types.Func,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) GenericCapability(
	api.GenericOperationSelection,
	*types.Signature,
) (api.GenericCapabilityReference, error) {
	panic("unused")
}

func (unusedNames) CallableABI(
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (unusedNames) SourceCallableABI(
	types.Object,
	*types.Signature,
) (api.CallableABIReference, error) {
	panic("unused")
}

func (unusedNames) ProviderGenericTypeArguments(
	*types.Func,
) ([]api.GenericTypeArgumentProjection, bool, error) {
	return nil, false, nil
}

func (unusedNames) ProviderInterface(
	types.Type,
) (gostdlib.ProviderInterface, bool, error) {
	return gostdlib.ProviderInterface{}, false, nil
}

func (unusedNames) ProviderInterfaceBridge(
	types.Type,
) (api.NameReference, bool, error) {
	return api.NameReference{}, false, nil
}

func (unusedNames) ProviderProfileInterfaceBridge(
	types.Type,
	[]gostdlib.ProviderCallableProfileInterface,
) (api.ProviderProfileBridgeReference, bool, error) {
	return api.ProviderProfileBridgeReference{}, false, nil
}

func (unusedNames) ProviderInterfaceCapability(
	types.Type,
	types.Type,
	string,
) (api.ProviderInterfaceCapabilityReference, bool, error) {
	return api.ProviderInterfaceCapabilityReference{}, false, nil
}

func (unusedNames) ProviderCallableProfile(
	*types.Func,
	string,
) (api.ProviderCallableProfileReference, bool, error) {
	return api.ProviderCallableProfileReference{}, false, nil
}

func (unusedNames) ProviderCallableProfileCandidates(
	*types.Func,
) ([]api.ProviderCallableProfileCandidate, bool, error) {
	return nil, false, nil
}

func (unusedNames) ProviderCallableParameters(
	*types.Func,
) ([]gostdlib.ProviderCallableParameterDocument, bool, error) {
	return nil, false, nil
}

func (unusedNames) ProviderStatefulProfileCandidates(
	*types.TypeName,
) ([]api.ProviderStatefulProfileCandidate, bool, error) {
	return nil, false, nil
}

func (unusedNames) ProviderStatefulProfileTarget(
	*types.TypeName,
	string,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ProviderRepresentationOwnsMethod(
	types.Type,
	*types.Func,
) (bool, error) {
	return false, nil
}

func (unusedNames) PackageVariable(
	*types.Var,
) (api.PackageVariableReference, error) {
	panic("unused")
}

func (unusedNames) NamedStructConstructor(
	*types.TypeName,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) NamedStructOperation(
	*types.TypeName,
	api.NamedStructOperation,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) NamedStructStorage(
	*types.TypeName,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ProviderStructField(
	*types.TypeName,
	*types.Var,
) (gostdlib.ProviderStructField, bool, error) {
	return gostdlib.ProviderStructField{}, false, nil
}

func (unusedNames) AnonymousStructStorage(
	*types.Struct,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ConstantValue(
	*types.Const,
) (api.NameReference, bool, error) {
	panic("unused")
}

func (unusedNames) ConstantProjection(
	*types.Const,
	types.BasicKind,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Member(*types.Var) (string, error) {
	panic("unused")
}

func (unusedNames) Primitive(api.PrimitiveAlias) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ProviderPrimitive(
	api.PrimitiveAlias,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Runtime(
	api.RuntimeSymbol,
	api.ImportPhase,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) TsonicCore(
	tsoniccore.Symbol,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) ExternalProviderFunction(
	string,
	string,
) (api.NameReference, error) {
	panic("unused")
}

func (unusedNames) Temporary(api.TemporaryKind) (string, error) {
	return "__logical", nil
}

func (unusedNames) ModuleExport(types.Object) (bool, error) {
	panic("unused")
}

type unusedValues struct{}

func (unusedValues) RequiresCustomEquality(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) RequiresExplicitType(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) RequiresStructuralCopy(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) SupportsHash(api.Context, types.Type) bool {
	panic("unused")
}

func (unusedValues) RequiresStorageProjection(api.Context, types.Type) (bool, error) {
	panic("unused")
}

func (unusedValues) StorageType(
	api.Context,
	ast.Node,
	types.Type,
) (api.TypeEmission, error) {
	panic("unused")
}

func (unusedValues) ToStorage(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) StorageZero(
	api.Context,
	ast.Node,
	types.Type,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) FromStorage(
	api.Context,
	ast.Node,
	types.Type,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Zero(
	api.Context,
	ast.Node,
	types.Type,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Transfer(
	api.Context,
	ast.Node,
	types.Type,
	types.Type,
	api.ValueTransferMode,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Assign(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Equal(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
	tsgo.Expression,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) Hash(
	api.Context,
	ast.Node,
	types.Type,
	tsgo.Expression,
) (api.ExpressionEmission, error) {
	panic("unused")
}

func (unusedValues) BinaryUpdate(
	api.Context,
	ast.Node,
	ast.Expr,
	types.Type,
	types.Type,
	token.Token,
	tsgo.Expression,
	api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}

func (unusedValues) Increment(
	api.Context,
	ast.Node,
	types.Type,
	token.Token,
	tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	panic("unused")
}
