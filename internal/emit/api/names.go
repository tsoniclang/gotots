package api

import (
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
	provider    bool
	requests    []RootRequest
}

type RecoveryCallableReference struct {
	reference   NameReference
	cooperative bool
}

type ProviderCallableProfileReference struct {
	reference     NameReference
	profile       gostdlib.ProviderCallableProfile
	guards        []types.Type
	contracts     []types.Type
	fromProvider  []NameReference
	values        []types.Object
	typeArguments []types.Type
}

type ProviderCallableProfileCandidate struct {
	profile gostdlib.ProviderCallableProfile
	guards  []types.Type
}

type ProviderStatefulProfileCandidate struct {
	profile       gostdlib.ProviderStatefulProfile
	typeArguments []types.Type
}

type ReflectionNames interface {
	ReflectionType(types.Type, *types.TypeName) (NameReference, error)
	ReflectionOperations(*types.TypeName) (NameReference, error)
	ReflectionTypeOf(types.Type, *types.TypeName) (NameReference, error)
}

func NewProviderStatefulProfileCandidate(
	profile gostdlib.ProviderStatefulProfile,
	typeArguments []types.Type,
) (ProviderStatefulProfileCandidate, error) {
	if !profile.Valid() || len(typeArguments) != len(profile.TypeArguments()) {
		return ProviderStatefulProfileCandidate{}, &NameError{
			Reason: "provider stateful-profile candidate is invalid",
		}
	}
	for _, selected := range typeArguments {
		if selected == nil {
			return ProviderStatefulProfileCandidate{}, &NameError{
				Reason: "provider stateful-profile interface is nil",
			}
		}
	}
	return ProviderStatefulProfileCandidate{
		profile:       profile,
		typeArguments: slices.Clone(typeArguments),
	}, nil
}

func (c ProviderStatefulProfileCandidate) Profile() gostdlib.ProviderStatefulProfile {
	return c.profile
}

func (c ProviderStatefulProfileCandidate) TypeArguments() []types.Type {
	return slices.Clone(c.typeArguments)
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
	contracts []types.Type,
	fromProvider []NameReference,
	values []types.Object,
	typeArguments []types.Type,
) (ProviderCallableProfileReference, error) {
	if reference.Name() == "" || !profile.Valid() ||
		len(guards) != len(profile.GuardInterfaces()) ||
		len(contracts) != len(profile.ContractInterfaces()) ||
		len(fromProvider) != len(profile.FromProviderInterfaces()) ||
		len(values) != len(profile.CanonicalValues()) ||
		len(typeArguments) != len(profile.CanonicalTypeArguments()) {
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
	for _, contract := range contracts {
		if contract == nil {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile contract is nil",
			}
		}
	}
	for _, bridge := range fromProvider {
		if bridge.Name() == "" {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile from-provider bridge is empty",
			}
		}
	}
	for _, value := range values {
		variable, ok := value.(*types.Var)
		if !ok || variable.IsField() || variable.Pkg() == nil ||
			variable.Parent() != variable.Pkg().Scope() {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile canonical value is invalid",
			}
		}
	}
	for _, argument := range typeArguments {
		if argument == nil {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile type argument is nil",
			}
		}
	}
	return ProviderCallableProfileReference{
		reference:     reference,
		profile:       profile,
		guards:        slices.Clone(guards),
		contracts:     slices.Clone(contracts),
		fromProvider:  slices.Clone(fromProvider),
		values:        slices.Clone(values),
		typeArguments: slices.Clone(typeArguments),
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

func (r ProviderCallableProfileReference) Contracts() []types.Type {
	return slices.Clone(r.contracts)
}

func (r ProviderCallableProfileReference) FromProviderBridges() []NameReference {
	return slices.Clone(r.fromProvider)
}

func (r ProviderCallableProfileReference) CanonicalValues() []types.Object {
	return slices.Clone(r.values)
}

func (r ProviderCallableProfileReference) CanonicalTypeArguments() []types.Type {
	return slices.Clone(r.typeArguments)
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
	providerBoundary bool,
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
		provider:    providerBoundary,
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

func (t MethodTarget) ProviderBoundary() bool {
	return t.provider
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
