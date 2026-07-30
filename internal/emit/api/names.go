package api

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	StructMakeMember        = "$make"
	StructStorageOfMember   = "$storageOf"
	StructFromStorageMember = "$fromStorage"
	StructStorageTypeSuffix = "$Storage"
)

type TemporaryKind uint8

const (
	TemporaryInvalid TemporaryKind = iota
	TemporaryAssignmentValue
	TemporaryMultipleResults
	TemporaryCompositeField
	TemporaryStructSource
	TemporaryReceiverValue
	TemporaryCallArgument
	TemporaryCallCallee
	TemporaryArrayReceiver
	TemporarySliceElement
	TemporarySliceReceiver
	TemporarySliceOperand
	TemporaryStoreOperand
	TemporaryMapOperand
	TemporaryAddressOperand
	TemporaryConversionOperand
	TemporaryArrayComparison
	TemporaryEqualityOperand
	TemporaryBinaryOperand
	TemporaryLogicalResult
	TemporaryArrayHash
	TemporaryArrayConstruction
	TemporarySliceConstruction
	TemporaryRangeOperand
	TemporaryRangeIndex
	TemporaryRangeValue
	TemporaryRangeKeys
	TemporaryRangeDecode
	TemporarySwitchTag
	TemporarySwitchSelection
	TemporarySwitchMatch
	TemporaryTypeSwitchValue
	TemporaryForFirstIteration
	TemporaryRangeState
	TemporaryDeferStack
	TemporaryDeferredCall
	TemporaryRecoveryAuthority
	TemporaryActivePanic
	TemporaryCaughtPanic
	TemporaryReturnResult
	TemporaryReturnLabel
	TemporaryControlTarget
	TemporaryGotoTarget
	TemporaryGotoState
	TemporaryGotoDispatch
	TemporaryChannelOperand
	TemporaryChannelResult
	TemporarySelectCase
	TemporaryRangeReturn
)

type NameReference struct {
	name     string
	requests []RootRequest
}

func NewNameReference(name string, requests ...RootRequest) (NameReference, error) {
	if name == "" {
		return NameReference{}, &NameError{Reason: "reference name is empty"}
	}
	return NameReference{name: name, requests: slices.Clone(requests)}, nil
}

func (r NameReference) Name() string {
	return r.name
}

func (r NameReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type MethodTargetKind uint8

const (
	MethodTargetInvalid MethodTargetKind = iota
	MethodTargetClassMember
	MethodTargetEnvironmentFunction
)

type MethodTarget struct {
	kind     MethodTargetKind
	name     string
	requests []RootRequest
}

func NewMethodTarget(
	kind MethodTargetKind,
	name string,
	requests ...RootRequest,
) (MethodTarget, error) {
	if (kind != MethodTargetClassMember &&
		kind != MethodTargetEnvironmentFunction) ||
		name == "" {
		return MethodTarget{}, &NameError{
			Name:   name,
			Reason: "method target is invalid",
		}
	}
	return MethodTarget{
		kind:     kind,
		name:     name,
		requests: slices.Clone(requests),
	}, nil
}

func (t MethodTarget) Kind() MethodTargetKind {
	return t.kind
}

func (t MethodTarget) Name() string {
	return t.name
}

func (t MethodTarget) Requests() []RootRequest {
	return slices.Clone(t.requests)
}

type InterfaceContractReference struct {
	typeName     string
	contractName string
	guardName    string
	requests     []RootRequest
}

func NewInterfaceContractReference(
	typeName string,
	contractName string,
	guardName string,
	requests ...RootRequest,
) (InterfaceContractReference, error) {
	if typeName == "" || contractName == "" || guardName == "" {
		return InterfaceContractReference{}, &NameError{
			Reason: "interface-contract reference name is empty",
		}
	}
	return InterfaceContractReference{
		typeName:     typeName,
		contractName: contractName,
		guardName:    guardName,
		requests:     slices.Clone(requests),
	}, nil
}

func (r InterfaceContractReference) TypeName() string {
	return r.typeName
}

func (r InterfaceContractReference) ContractName() string {
	return r.contractName
}

func (r InterfaceContractReference) GuardName() string {
	return r.guardName
}

func (r InterfaceContractReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type PackageVariableReference struct {
	stateName string
	fieldName string
	requests  []RootRequest
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

func (r PackageVariableReference) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	return factory.PropertyAccessExpression(
		factory.Identifier(r.stateName),
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
	GenericCallableProfile(*GenericCallableProfile) (NameReference, error)
	TypeReference(types.Object) (NameReference, error)
	PackageVariable(*types.Var) (PackageVariableReference, error)
	NamedStructOperation(*types.TypeName, NamedStructOperation) (NameReference, error)
	NamedStructStorage(*types.TypeName) (NameReference, error)
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
	InterfaceType(types.Type) (NameReference, error)
	InterfaceContract(types.Type) (InterfaceContractReference, error)
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
	ConstantProjection(*types.Const, types.BasicKind) (NameReference, error)
	Member(*types.Var) (string, error)
	Primitive(PrimitiveAlias) (NameReference, error)
	Runtime(RuntimeSymbol, ImportPhase) (NameReference, error)
	Temporary(TemporaryKind) (string, error)
	ModuleExport(types.Object) (bool, error)
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
)

func (i TargetIntrinsic) String() string {
	switch i {
	case TargetIntrinsicNumber:
		return "Number"
	default:
		return fmt.Sprintf("target-intrinsic(%d)", i)
	}
}

func (i TargetIntrinsic) Expression(
	factory tsgo.Factory,
) tsgo.PropertyAccessExpression {
	if i != TargetIntrinsicNumber {
		panic("invalid target intrinsic")
	}
	return factory.PropertyAccessExpression(
		factory.Identifier(TargetGlobalAnchorName),
		nil,
		factory.Identifier(i.String()),
		tsgo.NodeFlagsNone,
	)
}

type CallableABIReference struct {
	artifact    *GeneratedArtifact
	sourceOwner types.Object
	requests    []RootRequest
}

func NewCallableABIReference(
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	_, ok := artifact.CallableABI()
	if !ok {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference is invalid",
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "callable ABI reference request is invalid",
		}
	}
	return CallableABIReference{
		artifact: artifact,
		requests: slices.Clone(requests),
	}, nil
}

func NewSourceCallableABIReference(
	sourceOwner types.Object,
	artifact *GeneratedArtifact,
	requests ...RootRequest,
) (CallableABIReference, error) {
	sourceOwner = GenericDeclarationOrigin(sourceOwner)
	reference, err := NewCallableABIReference(artifact, requests...)
	if err != nil {
		return CallableABIReference{}, err
	}
	if sourceOwner == nil || sourceOwner.Pkg() == nil {
		return CallableABIReference{}, &RootRequestError{
			Reason: "source callable ABI reference owner is invalid",
		}
	}
	reference.sourceOwner = sourceOwner
	return reference, nil
}

func (r CallableABIReference) Artifact() *GeneratedArtifact {
	return r.artifact
}

func (r CallableABIReference) SourceOwner() (types.Object, bool) {
	return r.sourceOwner, r.sourceOwner != nil
}

func (r CallableABIReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}

type InterfaceMethodCallableCorrespondence struct {
	owner        *types.TypeName
	declaration  *types.Signature
	instantiated *types.Signature
}

func NewInterfaceMethodCallableCorrespondence(
	owner *types.TypeName,
	declaration *types.Signature,
	instantiated *types.Signature,
) (InterfaceMethodCallableCorrespondence, error) {
	origin, _ := GenericDeclarationOrigin(owner).(*types.TypeName)
	validSignatures := declaration != nil &&
		instantiated != nil &&
		declaration.Recv() == nil &&
		instantiated.Recv() == nil &&
		declaration.Params().Len() == instantiated.Params().Len() &&
		declaration.Results().Len() == instantiated.Results().Len() &&
		declaration.Variadic() == instantiated.Variadic() &&
		!types.Identical(declaration, instantiated)
	if origin == nil ||
		len(GenericDeclarationParameters(origin)) == 0 ||
		!validSignatures {
		return InterfaceMethodCallableCorrespondence{}, &NameError{
			Reason: "interface-method callable correspondence is invalid",
		}
	}
	return InterfaceMethodCallableCorrespondence{
		owner:        origin,
		declaration:  declaration,
		instantiated: instantiated,
	}, nil
}

func (c InterfaceMethodCallableCorrespondence) Parts() (
	*types.TypeName,
	*types.Signature,
	*types.Signature,
) {
	return c.owner, c.declaration, c.instantiated
}

type InterfaceMethodCallableReference struct {
	artifacts      []*GeneratedArtifact
	correspondence []InterfaceMethodCallableCorrespondence
	requests       []RootRequest
}

func NewInterfaceMethodCallableReference(
	artifacts []*GeneratedArtifact,
	correspondence []InterfaceMethodCallableCorrespondence,
	requests ...RootRequest,
) (InterfaceMethodCallableReference, error) {
	if len(artifacts) == 0 {
		return InterfaceMethodCallableReference{}, &NameError{
			Reason: "interface-method callable identities are absent",
		}
	}
	artifacts = slices.Clone(artifacts)
	slices.SortFunc(
		artifacts,
		func(left *GeneratedArtifact, right *GeneratedArtifact) int {
			if left == nil || right == nil {
				switch {
				case left == right:
					return 0
				case left == nil:
					return -1
				default:
					return 1
				}
			}
			return strings.Compare(left.ArtifactKey(), right.ArtifactKey())
		},
	)
	var previous *GeneratedArtifact
	for _, callable := range artifacts {
		if callable == nil ||
			callable.Kind() != GeneratedArtifactInterfaceMethodCallable ||
			!callable.Valid() ||
			callable == previous {
			return InterfaceMethodCallableReference{}, &NameError{
				Reason: "interface-method callable identities are invalid",
			}
		}
		previous = callable
	}
	correspondence = slices.Clone(correspondence)
	for _, selected := range correspondence {
		owner, declaration, instantiated := selected.Parts()
		if owner == nil || declaration == nil || instantiated == nil {
			return InterfaceMethodCallableReference{}, &NameError{
				Reason: "interface-method callable correspondences are invalid",
			}
		}
	}
	if err := validateReferenceRequests(requests); err != nil {
		return InterfaceMethodCallableReference{}, &RootRequestError{
			Reason: "interface-method reference request is invalid",
		}
	}
	return InterfaceMethodCallableReference{
		artifacts:      artifacts,
		correspondence: correspondence,
		requests:       slices.Clone(requests),
	}, nil
}

func (r InterfaceMethodCallableReference) Artifacts() []*GeneratedArtifact {
	return slices.Clone(r.artifacts)
}

func (r InterfaceMethodCallableReference) Correspondences() []InterfaceMethodCallableCorrespondence {
	return slices.Clone(r.correspondence)
}

func (r InterfaceMethodCallableReference) Requests() []RootRequest {
	return slices.Clone(r.requests)
}
