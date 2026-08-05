package api

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"strings"
)

func (r DeclarationRequirement) Owner() ArtifactOwner {
	return r.owner
}

func (r DeclarationRequirement) Kind() DeclarationRequirementKind {
	return r.kind
}

func (r DeclarationRequirement) NamedStructOperation() (
	*types.TypeName,
	NamedStructOperation,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementNamedStructOperation {
		return nil, NamedStructOperationInvalid, false
	}
	return r.typeName, r.operation, true
}

func (r DeclarationRequirement) AddressableStorage() (
	ArtifactOwner,
	*types.Var,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAddressableStorage {
		return ArtifactOwner{}, nil, false
	}
	return r.owner, r.variable, true
}

func (r DeclarationRequirement) ConstantProjection() (
	*types.Const,
	types.BasicKind,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementConstantProjection {
		return nil, types.Invalid, false
	}
	source, sourceOK := r.owner.Source()
	constant, ok := source.(*types.Const)
	return constant, r.projection, sourceOK && ok
}

func (r DeclarationRequirement) LocalConstantProjection() (
	*types.Func,
	*types.Const,
	types.BasicKind,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementLocalConstantProjection {
		return nil, nil, types.Invalid, false
	}
	source, sourceOK := r.owner.Source()
	owner, ok := source.(*types.Func)
	return owner, r.constant, r.projection, sourceOK && ok
}

func (r DeclarationRequirement) GenericOperation() (
	types.Object,
	*GenericOperationContract,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericOperation {
		return nil, nil, false
	}
	source, sourceOK := r.owner.Source()
	return source,
		r.genericOperation,
		sourceOK &&
			GenericDeclarationOrigin(source) == source
}

func (r DeclarationRequirement) AnonymousStruct() (
	*GeneratedArtifact,
	AnonymousStructDemand,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAnonymousStruct {
		return nil, AnonymousStructDemandInvalid, false
	}
	return r.generated, r.anonymousDemand, true
}

func (r DeclarationRequirement) MapSpecialization() (
	*GeneratedArtifact,
	MapSpecializationDemand,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementMapSpecialization {
		return nil, MapSpecializationDemandInvalid, false
	}
	return r.generated, r.mapDemand, true
}

func (r DeclarationRequirement) InterfaceAdapter() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceAdapter,
		GeneratedArtifactInterfaceAdapter,
	)
}

func (r DeclarationRequirement) InterfaceAdapterContract() (
	*GeneratedArtifact,
	*types.Interface,
	string,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementInterfaceAdapter ||
		r.interfaceContract == nil ||
		r.interfaceContractKey == "" {
		return nil, nil, "", false
	}
	return r.generated,
		r.interfaceContract,
		r.interfaceContractKey,
		true
}

func (r DeclarationRequirement) AnonymousInterface() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementAnonymousInterface,
		GeneratedArtifactAnonymousInterface,
	)
}

func (r DeclarationRequirement) InterfaceMethodToken() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceMethodToken,
		GeneratedArtifactInterfaceMethodToken,
	)
}

func (r DeclarationRequirement) InterfaceMethodCallable() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceMethodCallable,
		GeneratedArtifactInterfaceMethodCallable,
	)
}

func (r DeclarationRequirement) InterfaceDynamicTypeToken() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceDynamicTypeToken,
		GeneratedArtifactInterfaceDynamicTypeToken,
	)
}

func (r DeclarationRequirement) ReflectionType() (*GeneratedArtifact, bool) {
	return r.generatedDefinition(
		DeclarationRequirementReflectionType,
		GeneratedArtifactReflectionType,
	)
}

func (r DeclarationRequirement) UnsafeCodec() (*GeneratedArtifact, bool) {
	return r.generatedDefinition(
		DeclarationRequirementUnsafeCodec,
		GeneratedArtifactUnsafeCodec,
	)
}

func (r DeclarationRequirement) ProviderInterfaceBridge() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementProviderInterfaceBridge,
		GeneratedArtifactProviderInterfaceBridge,
	)
}

func (r DeclarationRequirement) ProviderInterfaceCapability() (
	*GeneratedArtifact,
	*types.Interface,
	string,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementProviderInterfaceCapability {
		return nil, nil, "", false
	}
	return r.generated,
		r.interfaceContract,
		r.interfaceContractKey,
		true
}

func (r DeclarationRequirement) ProviderProfileInterfaceCapability() (
	*GeneratedArtifact,
	*GeneratedArtifact,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementProviderProfileInterfaceCapability {
		return nil, nil, false
	}
	return r.generated,
		r.providerProfileTarget,
		true
}

func (r DeclarationRequirement) ProviderStatefulRepresentation() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementProviderStatefulRepresentation,
		GeneratedArtifactProviderStatefulRepresentation,
	)
}

func (r DeclarationRequirement) GenericCapability() (
	*GeneratedArtifact,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementGenericCapability ||
		r.generated.Kind() != GeneratedArtifactGenericCapability {
		return nil, false
	}
	return r.generated, true
}

func (r DeclarationRequirement) DeferredCallableRegistry() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementDeferredCallableRegistry,
		GeneratedArtifactDeferredCallableRegistry,
	)
}

func (r DeclarationRequirement) CallableControl() (
	ArtifactOwner,
	ast.Node,
	ast.Node,
	CallableControlFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCallableControl {
		return ArtifactOwner{}, nil, nil, CallableControlInvalid, false
	}
	return r.owner, r.enclosing, r.callable, r.control, true
}

func (r DeclarationRequirement) GotoControl() (
	*types.Label,
	token.Pos,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCallableControl ||
		r.control != CallableControlGoto {
		return nil, token.NoPos, false
	}
	return r.controlLabel, r.controlPosition, true
}

func (r DeclarationRequirement) IteratorReturnControl() (
	*ast.RangeStmt,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementCallableControl ||
		r.control != CallableControlIteratorReturn {
		return nil, false
	}
	return r.controlRange, true
}

func (r DeclarationRequirement) generatedDefinition(
	requirementKind DeclarationRequirementKind,
	artifactKind GeneratedArtifactKind,
) (*GeneratedArtifact, bool) {
	if !r.Valid() ||
		r.kind != requirementKind ||
		r.generated.Kind() != artifactKind {
		return nil, false
	}
	return r.generated, true
}

func (r DeclarationRequirement) GeneratedArtifact() (
	*GeneratedArtifact,
	bool,
) {
	if !r.Valid() {
		return nil, false
	}
	switch r.kind {
	case DeclarationRequirementAnonymousStruct,
		DeclarationRequirementMapSpecialization,
		DeclarationRequirementInterfaceAdapter,
		DeclarationRequirementAnonymousInterface,
		DeclarationRequirementInterfaceMethodToken,
		DeclarationRequirementInterfaceMethodCallable,
		DeclarationRequirementInterfaceDynamicTypeToken,
		DeclarationRequirementGenericCapability,
		DeclarationRequirementCallableABI,
		DeclarationRequirementPointerRepresentation,
		DeclarationRequirementProviderInterfaceBridge,
		DeclarationRequirementProviderInterfaceCapability,
		DeclarationRequirementProviderProfileInterfaceCapability,
		DeclarationRequirementProviderStatefulRepresentation,
		DeclarationRequirementDeferredCallableRegistry,
		DeclarationRequirementGenericConcretization,
		DeclarationRequirementReflectionType,
		DeclarationRequirementUnsafeCodec:
		return r.generated, true
	case DeclarationRequirementTypeRepresentation:
		if r.generated != nil {
			return r.generated, true
		}
		return nil, false
	default:
		return nil, false
	}
}

func (r DeclarationRequirement) LexicalGeneratedArtifact() (
	*GeneratedArtifact,
	bool,
) {
	if artifact, ok := r.GeneratedArtifact(); ok {
		return artifact,
			artifact.Placement() == GeneratedArtifactPlacementLexical &&
				r.Owner() == artifact.ReconstructionOwner()
	}
	facet, ok := r.CooperativeCallable()
	if !ok {
		return nil, false
	}
	artifact, ok := facet.GenericCapability()
	return artifact,
		ok &&
			artifact.Placement() == GeneratedArtifactPlacementLexical &&
			r.Owner() == artifact.ReconstructionOwner()
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
