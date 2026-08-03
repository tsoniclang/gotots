package api

import (
	"fmt"
	"go/types"
	"slices"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type PackageVariableReference struct {
	qualifier string
	stateName string
	fieldName string
	requests  []RootRequest
	provider  bool
}

func NewQualifiedPackageVariableReference(
	qualifier string,
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	if qualifier == "" {
		return PackageVariableReference{},
			&NameError{Reason: "package state qualifier is empty"}
	}
	reference, err := NewPackageVariableReference(
		stateName,
		fieldName,
		requests...,
	)
	if err != nil {
		return PackageVariableReference{}, err
	}
	reference.qualifier = qualifier
	return reference, nil
}

func NewProviderQualifiedPackageVariableReference(
	qualifier string,
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	reference, err := NewQualifiedPackageVariableReference(
		qualifier,
		stateName,
		fieldName,
		requests...,
	)
	if err != nil {
		return PackageVariableReference{}, err
	}
	reference.provider = true
	return reference, nil
}

func NewPackageVariableReference(
	stateName string,
	fieldName string,
	requests ...RootRequest,
) (PackageVariableReference, error) {
	switch {
	case stateName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package state name is empty"}
	case fieldName == "":
		return PackageVariableReference{},
			&NameError{Reason: "package variable field name is empty"}
	}
	return PackageVariableReference{
		stateName: stateName,
		fieldName: fieldName,
		requests:  slices.Clone(requests),
	}, nil
}

func (r PackageVariableReference) StateName() string {
	return r.stateName
}

func (r PackageVariableReference) FieldName() string {
	return r.fieldName
}

func (r PackageVariableReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

func (r PackageVariableReference) ProviderBoundary() bool {
	return r.provider
}

func (r PackageVariableReference) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	var state tsgo.Expression = factory.Identifier(r.stateName)
	if r.qualifier != "" {
		state = factory.PropertyAccessExpression(
			factory.Identifier(r.qualifier),
			nil,
			factory.Identifier(r.stateName),
			tsgo.NodeFlagsNone,
		)
	}
	return factory.PropertyAccessExpression(
		state,
		nil,
		factory.Identifier(r.fieldName),
		tsgo.NodeFlagsNone,
	)
}

type Names interface {
	Declare(types.Object) (string, error)
	Parameter(*types.Var, int) (string, error)
	Result(*types.Var, int) (string, error)
	Reference(types.Object) (NameReference, error)
	ProviderGenericTypeArguments(*types.Func) (
		[]GenericTypeArgumentProjection,
		bool,
		error,
	)
	TypeReference(types.Object) (NameReference, error)
	PackageVariable(*types.Var) (PackageVariableReference, error)
	NamedStructConstructor(*types.TypeName) (NameReference, error)
	NamedStructOperation(*types.TypeName, NamedStructOperation) (NameReference, error)
	NamedStructStorage(*types.TypeName) (NameReference, error)
	DefinedValueRepresentation(*types.TypeName) (DefinedValueRepresentation, error)
	AnonymousStruct(
		*types.Struct,
		AnonymousStructDemand,
		ImportPhase,
	) (NameReference, error)
	AnonymousStructStorage(*types.Struct) (NameReference, error)
	MapSpecialization(
		types.Type,
		MapSpecializationDemand,
	) (NameReference, error)
	InterfaceAdapter(types.Type, types.Type) (NameReference, error)
	InterfaceContractDemand(types.Type, types.Type) ([]RootRequest, error)
	InterfaceDynamicType(types.Type) (NameReference, error)
	ProviderInterface(types.Type) (gostdlib.ProviderInterface, bool, error)
	ProviderInterfaceBridge(types.Type) (NameReference, bool, error)
	ProviderCallableProfile(*types.Func, string) (
		ProviderCallableProfileReference,
		bool,
		error,
	)
	ProviderCallableProfileCandidates(*types.Func) (
		[]ProviderCallableProfileCandidate,
		bool,
		error,
	)
	ProviderCallableParameters(*types.Func) (
		[]gostdlib.ProviderCallableParameterDocument,
		bool,
		error,
	)
	ProviderStatefulProfileCandidates(*types.TypeName) (
		[]ProviderStatefulProfileCandidate,
		bool,
		error,
	)
	ProviderStatefulProfileTarget(
		*types.TypeName,
		string,
		ImportPhase,
	) (NameReference, error)
	ProviderRepresentationOwnsMethod(types.Type, *types.Func) (bool, error)
	InterfaceType(types.Type) (NameReference, error)
	InterfaceContract(types.Type) (InterfaceContractReference, error)
	RecoveryCallable(*types.Func) (RecoveryCallableReference, bool, error)
	DeferredCallable(*types.Func, string) (NameReference, error)
	DeferredCallableRegistry(*types.Signature) (NameReference, error)
	MethodTarget(*types.Func) (MethodTarget, error)
	InterfaceMethodName(*types.Func) (string, error)
	InterfaceMethodCallable(*types.Func) (
		InterfaceMethodCallableReference,
		error,
	)
	InterfaceMethodToken(*types.Func) (NameReference, error)
	GenericCapability(
		GenericOperationSelection,
		*types.Signature,
	) (GenericCapabilityReference, error)
	CallableABI(*types.Signature) (CallableABIReference, error)
	SourceCallableABI(
		types.Object,
		*types.Signature,
	) (CallableABIReference, error)
	ConstantValue(*types.Const) (NameReference, bool, error)
	ConstantProjection(*types.Const, types.BasicKind) (NameReference, error)
	Member(*types.Var) (string, error)
	Primitive(PrimitiveAlias) (NameReference, error)
	Runtime(RuntimeSymbol, ImportPhase) (NameReference, error)
	Temporary(TemporaryKind) (string, error)
	ModuleExport(types.Object) (bool, error)
}

type TypeRepresentationNames interface {
	TypeRepresentation(
		*types.TypeName,
		TypeRepresentationFacet,
	) ([]RootRequest, error)
	AnonymousStructTypeRepresentation(
		*types.Struct,
		TypeRepresentationFacet,
	) ([]RootRequest, error)
}

func TemporaryPrefix(kind TemporaryKind) (string, error) {
	switch kind {
	case TemporaryAssignmentValue:
		return "__gotots_assign_", nil
	case TemporaryMultipleResults:
		return "__gotots_results_", nil
	case TemporaryCompositeField:
		return "__gotots_field_", nil
	case TemporaryStructSource:
		return "__gotots_struct_", nil
	case TemporaryReceiverValue:
		return "__gotots_receiver_", nil
	case TemporaryCallArgument:
		return "__gotots_argument_", nil
	case TemporaryCallCallee:
		return "__gotots_callee_", nil
	case TemporaryArrayReceiver:
		return "__gotots_array_", nil
	case TemporarySliceElement:
		return "__gotots_slice_element_", nil
	case TemporarySliceReceiver:
		return "__gotots_slice_receiver_", nil
	case TemporarySliceOperand:
		return "__gotots_slice_operand_", nil
	case TemporaryStoreOperand:
		return "__gotots_store_", nil
	case TemporaryMapOperand:
		return "__gotots_map_", nil
	case TemporaryAddressOperand:
		return "__gotots_address_", nil
	case TemporaryConversionOperand:
		return "__gotots_conversion_", nil
	case TemporaryArrayComparison:
		return "__gotots_array_equal_", nil
	case TemporaryEqualityOperand:
		return "__gotots_equal_operand_", nil
	case TemporaryBinaryOperand:
		return "__gotots_binary_operand_", nil
	case TemporaryLogicalResult:
		return "__gotots_logical_result_", nil
	case TemporaryArrayHash:
		return "__gotots_array_hash_", nil
	case TemporaryArrayConstruction:
		return "__gotots_array_build_", nil
	case TemporarySliceConstruction:
		return "__gotots_slice_build_", nil
	case TemporaryRangeOperand:
		return "__gotots_range_", nil
	case TemporaryRangeIndex:
		return "__gotots_range_index_", nil
	case TemporaryRangeValue:
		return "__gotots_range_value_", nil
	case TemporaryRangeKeys:
		return "__gotots_range_keys_", nil
	case TemporaryRangeDecode:
		return "__gotots_range_decode_", nil
	case TemporarySwitchTag:
		return "__gotots_switch_tag_", nil
	case TemporarySwitchSelection:
		return "__gotots_switch_selection_", nil
	case TemporarySwitchMatch:
		return "__gotots_switch_match_", nil
	case TemporaryTypeSwitchValue:
		return "__gotots_type_switch_", nil
	case TemporaryForFirstIteration:
		return "__gotots_for_first_", nil
	case TemporaryRangeState:
		return "__gotots_range_state_", nil
	case TemporaryDeferStack:
		return "__gotots_defers_", nil
	case TemporaryDeferredCall:
		return "__gotots_deferred_", nil
	case TemporaryRecoveryAuthority:
		return "__gotots_recovery_", nil
	case TemporaryActivePanic:
		return "__gotots_panic_", nil
	case TemporaryCaughtPanic:
		return "__gotots_caught_", nil
	case TemporaryReturnResult:
		return "__gotots_return_", nil
	case TemporaryReturnLabel:
		return "__gotots_return_block_", nil
	case TemporaryControlTarget:
		return "__gotots_control_target_", nil
	case TemporaryGotoTarget:
		return "__gotots_goto_target_", nil
	case TemporaryGotoState:
		return "__gotots_goto_state_", nil
	case TemporaryGotoDispatch:
		return "__gotots_goto_dispatch_", nil
	case TemporaryChannelOperand:
		return "__gotots_channel_", nil
	case TemporaryChannelResult:
		return "__gotots_receive_", nil
	case TemporarySelectCase:
		return "__gotots_select_", nil
	case TemporaryRangeReturn:
		return "__gotots_range_return_", nil
	default:
		return "", &NameError{
			Reason: fmt.Sprintf("temporary kind %d is invalid", kind),
		}
	}
}

const TargetGlobalAnchorName = "globalThis"

type TargetIntrinsic uint8

const (
	TargetIntrinsicInvalid TargetIntrinsic = iota
	TargetIntrinsicNumber
	TargetIntrinsicString
	TargetIntrinsicBigInt
	TargetIntrinsicMath
	TargetIntrinsicObject
	TargetIntrinsicPromise
	TargetIntrinsicError
)

func (i TargetIntrinsic) Valid() bool {
	return i >= TargetIntrinsicNumber && i <= TargetIntrinsicError
}

func (i TargetIntrinsic) String() string {
	switch i {
	case TargetIntrinsicNumber:
		return "Number"
	case TargetIntrinsicString:
		return "String"
	case TargetIntrinsicBigInt:
		return "BigInt"
	case TargetIntrinsicMath:
		return "Math"
	case TargetIntrinsicObject:
		return "Object"
	case TargetIntrinsicPromise:
		return "Promise"
	case TargetIntrinsicError:
		return "Error"
	default:
		return fmt.Sprintf("target-intrinsic(%d)", i)
	}
}

func (i TargetIntrinsic) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	if !i.Valid() {
		panic("invalid target intrinsic")
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(TargetGlobalAnchorName),
		nil,
		factory.Identifier(i.String()),
		tsgo.NodeFlagsNone,
	)
}

func (i TargetIntrinsic) UnshadowedExpression(
	factory tsgo.Factory,
) tsgo.Identifier {
	if !i.Valid() {
		panic("invalid target intrinsic")
	}
	return factory.Identifier(i.String())
}

func (i TargetIntrinsic) ReservesTypeName() bool {
	return i == TargetIntrinsicObject || i == TargetIntrinsicPromise
}

func IsReservedTargetTypeName(name string) bool {
	return name == TargetIntrinsicObject.String() ||
		name == TargetIntrinsicPromise.String()
}

func (i TargetIntrinsic) TypeName(factory tsgo.Factory) tsgo.Identifier {
	if !i.ReservesTypeName() {
		panic("invalid target intrinsic")
	}
	return factory.Identifier(i.String())
}
