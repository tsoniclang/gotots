package api

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const (
	StructMakeMember         = "$make"
	StructStorageOfMember    = "$storageOf"
	StructFromStorageMember  = "$fromStorage"
	StructStorageTypeSuffix  = "$Storage"
	ProviderBridgeFromMember = "$from"
	ProviderBridgeToMember   = "$to"
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

type MethodTargetKind uint8

const (
	MethodTargetInvalid MethodTargetKind = iota
	MethodTargetClassMember
	MethodTargetEnvironmentFunction
)

type MethodTarget struct {
	kind        MethodTargetKind
	name        string
	receiverABI MethodReceiverABI
	requests    []RootRequest
}

type RecoveryCallableReference struct {
	reference   NameReference
	cooperative bool
}

type ProviderCallableProfileReference struct {
	reference NameReference
	profile   gostdlib.ProviderCallableProfile
	guards    []types.Type
}

type ProviderCallableProfileCandidate struct {
	profile gostdlib.ProviderCallableProfile
	guards  []types.Type
}

func NewProviderCallableProfileCandidate(
	profile gostdlib.ProviderCallableProfile,
	guards []types.Type,
) (ProviderCallableProfileCandidate, error) {
	if !profile.Valid() || len(guards) != len(profile.GuardInterfaces()) {
		return ProviderCallableProfileCandidate{}, &NameError{
			Reason: "provider callable-profile candidate is invalid",
		}
	}
	for _, guard := range guards {
		if guard == nil {
			return ProviderCallableProfileCandidate{}, &NameError{
				Reason: "provider callable-profile candidate guard is nil",
			}
		}
	}
	return ProviderCallableProfileCandidate{
		profile: profile,
		guards:  slices.Clone(guards),
	}, nil
}

func (c ProviderCallableProfileCandidate) Profile() gostdlib.ProviderCallableProfile {
	return c.profile
}

func (c ProviderCallableProfileCandidate) Guards() []types.Type {
	return slices.Clone(c.guards)
}

func NewProviderCallableProfileReference(
	reference NameReference,
	profile gostdlib.ProviderCallableProfile,
	guards []types.Type,
) (ProviderCallableProfileReference, error) {
	if reference.Name() == "" || !profile.Valid() ||
		len(guards) != len(profile.GuardInterfaces()) {
		return ProviderCallableProfileReference{}, &NameError{
			Reason: "provider callable-profile reference is invalid",
		}
	}
	for _, guard := range guards {
		if guard == nil {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile guard is nil",
			}
		}
	}
	return ProviderCallableProfileReference{
		reference: reference,
		profile:   profile,
		guards:    slices.Clone(guards),
	}, nil
}

func (r ProviderCallableProfileReference) Expression(
	factory tsgo.Factory,
) tsgo.Expression {
	return r.reference.Expression(factory)
}

func (r ProviderCallableProfileReference) Requests() []RootRequest {
	return r.reference.Requests()
}

func (r ProviderCallableProfileReference) Profile() gostdlib.ProviderCallableProfile {
	return r.profile
}

func (r ProviderCallableProfileReference) Guards() []types.Type {
	return slices.Clone(r.guards)
}

func NewRecoveryCallableReference(
	reference NameReference,
	cooperative bool,
) (RecoveryCallableReference, error) {
	if reference.Name() == "" {
		return RecoveryCallableReference{}, &NameError{
			Reason: "recovery-callable reference is empty",
		}
	}
	return RecoveryCallableReference{
		reference:   reference,
		cooperative: cooperative,
	}, nil
}

func (r RecoveryCallableReference) Expression(
	factory tsgo.Factory,
) tsgo.Expression {
	return r.reference.Expression(factory)
}

func (r RecoveryCallableReference) Requests() []RootRequest {
	return r.reference.Requests()
}

func (r RecoveryCallableReference) Cooperative() bool {
	return r.cooperative
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

func NewMethodTarget(
	kind MethodTargetKind,
	name string,
	receiverABI MethodReceiverABI,
	requests ...RootRequest,
) (MethodTarget, error) {
	if (kind != MethodTargetClassMember &&
		kind != MethodTargetEnvironmentFunction) ||
		name == "" ||
		!receiverABI.Valid() {
		return MethodTarget{}, &NameError{
			Name:   name,
			Reason: "method target is invalid",
		}
	}
	return MethodTarget{
		kind:        kind,
		name:        name,
		receiverABI: receiverABI,
		requests:    slices.Clone(requests),
	}, nil
}

func (t MethodTarget) Kind() MethodTargetKind {
	return t.kind
}

func (t MethodTarget) Name() string {
	return t.name
}

func (t MethodTarget) ReceiverABI() MethodReceiverABI {
	return t.receiverABI
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
	GenericCallableProfile(*GenericCallableProfile) (NameReference, error)
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
	ProviderRepresentationOwnsMethod(types.Type, *types.Func) (bool, error)
	InterfaceType(types.Type) (NameReference, error)
	InterfaceContract(types.Type) (InterfaceContractReference, error)
	RecoveryCallable(*types.Func) (RecoveryCallableReference, bool, error)
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
