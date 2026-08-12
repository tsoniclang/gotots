package api

import (
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/types"
	"slices"
)

const (
	StructMakeMember              = "$make"
	StructStorageOfMember         = "$storageOf"
	StructFromStorageMember       = "$fromStorage"
	StructStorageTypeSuffix       = "$Storage"
	InterfaceContractSuffix       = "$contract"
	InterfaceGuardSuffix          = "$is"
	ProviderBridgeFromMember      = "$from"
	ProviderBridgeToMember        = "$to"
	ProviderProfileContractSuffix = "$ProviderContract"
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
	temporaryKindLimit
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
	reference       NameReference
	profile         gostdlib.ProviderCallableProfile
	capabilityViews []NameReference
	guards          []types.Type
	contracts       []types.Type
	fromProvider    []NameReference
	values          []types.Object
	typeArguments   []types.Type
}

type ProviderCallableProfileCandidate struct {
	profile         gostdlib.ProviderCallableProfile
	capabilityViews []ProviderCallableProfileCapability
	guards          []types.Type
}

type ProviderCallableProfileCapability struct {
	base   *types.Named
	target *types.Named
}

func NewProviderCallableProfileCapability(
	base *types.Named,
	target *types.Named,
) (ProviderCallableProfileCapability, error) {
	if base == nil || target == nil || base.Obj() == nil || target.Obj() == nil ||
		base.Origin() != base || target.Origin() != target ||
		types.Identical(base, target) {
		return ProviderCallableProfileCapability{}, &NameError{
			Reason: "provider callable-profile capability is invalid",
		}
	}
	baseContract, baseOK := base.Underlying().(*types.Interface)
	targetContract, targetOK := target.Underlying().(*types.Interface)
	if !baseOK || !targetOK ||
		!types.Implements(target, baseContract.Complete()) ||
		types.Implements(base, targetContract.Complete()) {
		return ProviderCallableProfileCapability{}, &NameError{
			Reason: "provider callable-profile capability relation is invalid",
		}
	}
	return ProviderCallableProfileCapability{base: base, target: target}, nil
}

func (c ProviderCallableProfileCapability) Base() *types.Named {
	return c.base
}

func (c ProviderCallableProfileCapability) Target() *types.Named {
	return c.target
}

type ProviderStatefulProfileCandidate struct {
	profile       gostdlib.ProviderStatefulProfile
	typeArguments []types.Type
}

type ReflectionNames interface {
	ProviderOwnershipNames
	ReflectionMethodIdentity(*types.Func) (string, error)
	ReflectionType(types.Type, *types.TypeName) (NameReference, error)
	ReflectionOperations(*types.TypeName) (NameReference, error)
	ReflectionTypeOf(types.Type, *types.TypeName) (NameReference, error)
	// ReflectionValueOf demands the canonical descriptor plus the generated
	// value-operation facet for one reflected operand type and returns the
	// metadata operation reference carrying those requests.
	ReflectionValueOf(types.Type, *types.TypeName) (NameReference, error)
	// ReflectionValueOperationsDemanded reports whether the value-operation
	// facet was demanded for one canonical reflection artifact.
	ReflectionValueOperationsDemanded(string) bool
	// ReflectionValueType returns the canonical descriptor reference for
	// one type while joining its value-operation facet demand.
	ReflectionValueType(types.Type, *types.TypeName) (NameReference, error)
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
	capabilityViews []ProviderCallableProfileCapability,
	guards []types.Type,
) (ProviderCallableProfileCandidate, error) {
	if !profile.Valid() ||
		len(capabilityViews) != len(profile.CapabilityViews()) ||
		len(guards) != len(profile.GuardInterfaces()) {
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
	for _, capability := range capabilityViews {
		if capability.base == nil || capability.target == nil {
			return ProviderCallableProfileCandidate{}, &NameError{
				Reason: "provider callable-profile candidate capability is invalid",
			}
		}
	}
	return ProviderCallableProfileCandidate{
		profile:         profile,
		capabilityViews: slices.Clone(capabilityViews),
		guards:          slices.Clone(guards),
	}, nil
}

func (c ProviderCallableProfileCandidate) Profile() gostdlib.ProviderCallableProfile {
	return c.profile
}

func (c ProviderCallableProfileCandidate) Guards() []types.Type {
	return slices.Clone(c.guards)
}

func (c ProviderCallableProfileCandidate) CapabilityViews() []ProviderCallableProfileCapability {
	return slices.Clone(c.capabilityViews)
}

func NewProviderCallableProfileReference(
	reference NameReference,
	profile gostdlib.ProviderCallableProfile,
	capabilityViews []NameReference,
	guards []types.Type,
	contracts []types.Type,
	fromProvider []NameReference,
	values []types.Object,
	typeArguments []types.Type,
) (ProviderCallableProfileReference, error) {
	if reference.Name() == "" || !profile.Valid() ||
		len(capabilityViews) != len(profile.CapabilityViews()) ||
		len(guards) != len(profile.GuardInterfaces()) ||
		len(contracts) != len(profile.ContractInterfaces()) ||
		len(fromProvider) != len(profile.FromProviderInterfaces()) ||
		len(values) != len(profile.CanonicalValues()) ||
		len(typeArguments) != len(profile.CanonicalTypeArguments()) {
		return ProviderCallableProfileReference{}, &NameError{
			Reason: "provider callable-profile reference is invalid",
		}
	}
	for _, view := range capabilityViews {
		if view.Name() == "" {
			return ProviderCallableProfileReference{}, &NameError{
				Reason: "provider callable-profile capability view is empty",
			}
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
		reference:       reference,
		profile:         profile,
		capabilityViews: slices.Clone(capabilityViews),
		guards:          slices.Clone(guards),
		contracts:       slices.Clone(contracts),
		fromProvider:    slices.Clone(fromProvider),
		values:          slices.Clone(values),
		typeArguments:   slices.Clone(typeArguments),
	}, nil
}

func (r ProviderCallableProfileReference) CapabilityViews() []NameReference {
	return slices.Clone(r.capabilityViews)
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

func (r RecoveryCallableReference) ProviderBoundary() bool {
	return r.reference.ProviderBoundary()
}

func (o *GeneratedArtifact) ArtifactKey() string {
	if o == nil {
		return ""
	}
	return o.artifact
}

func (o *GeneratedArtifact) TargetName() string {
	if o == nil {
		return ""
	}
	return o.targetName
}

func (o *GeneratedArtifact) Placement() GeneratedArtifactPlacement {
	if o == nil {
		return GeneratedArtifactPlacementInvalid
	}
	return o.placement
}

func (o *GeneratedArtifact) OutputPath() string {
	if o == nil {
		return ""
	}
	return o.outputPath
}

func (o *GeneratedArtifact) LexicalOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	return o.lexicalOwner
}

func (o *GeneratedArtifact) LexicalAnchor() *types.TypeName {
	if o == nil {
		return nil
	}
	return o.anchor
}

func (o *GeneratedArtifact) ReconstructionOwner() ArtifactOwner {
	if o == nil {
		return ArtifactOwner{}
	}
	if o.placement == GeneratedArtifactPlacementLexical {
		return o.lexicalOwner
	}
	return MustGeneratedArtifactOwner(o)
}

func (o *GeneratedArtifact) Valid() bool {
	if o == nil ||
		!validGeneratedArtifactType(o.kind, o.sourceType) ||
		o.artifact == "" ||
		o.targetName == "" ||
		!o.placement.Valid() ||
		(o.kind == GeneratedArtifactGenericCapability) != o.generic.Valid() ||
		(o.kind == GeneratedArtifactInterfaceMethodToken &&
			!validInterfaceMethodRuntime(o.runtime)) ||
		(o.kind != GeneratedArtifactInterfaceMethodToken &&
			o.runtime != RuntimeInvalid) ||
		(o.kind == GeneratedArtifactGenericConcretization) !=
			(o.concretization != nil) ||
		(o.kind == GeneratedArtifactReflectionType) !=
			(o.reflectionType != nil) ||
		(len(o.providerProfile) != 0 &&
			o.kind != GeneratedArtifactProviderInterfaceBridge) {
		return false
	}
	switch o.placement {
	case GeneratedArtifactPlacementCompilation:
		return o.outputPath != "" &&
			!o.lexicalOwner.Valid() &&
			o.anchor == nil
	case GeneratedArtifactPlacementLexical:
		sourcePackage := o.lexicalOwner.Package()
		_, sourceOwned := o.lexicalOwner.Source()
		_, _, initializerOwned := o.lexicalOwner.PackageInitializer()
		return o.outputPath == "" &&
			(sourceOwned || initializerOwned) &&
			o.anchor != nil &&
			sourcePackage != nil &&
			o.anchor.Pkg() == sourcePackage &&
			o.anchor.Parent() != nil &&
			o.anchor.Parent() != o.anchor.Pkg().Scope()
	case GeneratedArtifactPlacementContract:
		return (o.kind == GeneratedArtifactCallableABI ||
			o.kind == GeneratedArtifactInterfaceMethodCallable) &&
			o.outputPath == "" &&
			!o.lexicalOwner.Valid() &&
			o.anchor == nil
	default:
		return false
	}
}

func validGeneratedArtifactType(
	kind GeneratedArtifactKind,
	sourceType types.Type,
) bool {
	if sourceType == nil || !kind.Valid() {
		return false
	}
	switch kind {
	case GeneratedArtifactAnonymousStruct:
		_, ok := types.Unalias(sourceType).(*types.Struct)
		return ok
	case GeneratedArtifactMapSpecialization:
		source, ok := types.Unalias(sourceType).(*types.Map)
		return ok && types.Comparable(source.Key())
	case GeneratedArtifactInterfaceAdapter:
		return interfaceAdapterType(sourceType)
	case GeneratedArtifactAnonymousInterface:
		source, ok := types.Unalias(sourceType).Underlying().(*types.Interface)
		return ok && source.IsMethodSet()
	case GeneratedArtifactInterfaceMethodToken:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok &&
			source.Recv() == nil &&
			!ContainsGenericTypeParameter(source)
	case GeneratedArtifactInterfaceMethodCallable:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source.Recv() == nil
	case GeneratedArtifactInterfaceDynamicTypeToken:
		return interfaceAdapterType(sourceType)
	case GeneratedArtifactGenericCapability:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && validGenericOperationSignature(source)
	case GeneratedArtifactCallableABI:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source.Recv() == nil
	case GeneratedArtifactDeferredCallableRegistry:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source.Recv() == nil &&
			!ContainsGenericTypeParameter(source)
	case GeneratedArtifactGenericConcretization:
		source, ok := types.Unalias(sourceType).(*types.Signature)
		return ok && source != nil
	case GeneratedArtifactProviderInterfaceBridge:
		source, ok := types.Unalias(sourceType).(*types.Named)
		if !ok || source.Obj() == nil {
			return false
		}
		contract, ok := source.Underlying().(*types.Interface)
		return ok && contract.Complete().IsMethodSet()
	case GeneratedArtifactProviderStatefulRepresentation:
		source, ok := types.Unalias(sourceType).(*types.Named)
		if !ok || source.Obj() == nil {
			return false
		}
		_, interfaceType := source.Underlying().(*types.Interface)
		return !interfaceType
	case GeneratedArtifactReflectionType:
		return sourceType != nil && !ContainsGenericTypeParameter(sourceType)
	default:
		return false
	}
}

func interfaceAdapterType(sourceType types.Type) bool {
	if sourceType == nil {
		return false
	}
	switch types.Unalias(sourceType).Underlying().(type) {
	case *types.Interface, *types.Tuple, *types.TypeParam, *types.Union:
		return false
	default:
		return true
	}
}

func validInterfaceMethodRuntime(symbol RuntimeSymbol) bool {
	return symbol == RuntimeInvalid ||
		symbol == RuntimeErrorMethodToken ||
		symbol == RuntimeRuntimeErrorToken
}
