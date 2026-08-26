package api

import (
	"go/ast"
	"go/token"
	"go/types"
)

type DeclarationRequirement struct {
	owner       ArtifactOwner
	kind        DeclarationRequirementKind
	operation   NamedStructOperation
	typeName    *types.TypeName
	classMethod *types.Func
	// constant is the untyped constant a local projection materializes. A
	// package projection owns the constant directly (owner is the constant), so
	// this stays nil there; a local projection is owned by the enclosing
	// function, so the constant identity travels here.
	constant *types.Const
	// projection is the exact target basic representation of an untyped
	// constant projection. A basic kind is a canonical, comparable dedup key —
	// unlike a types.Type interface value, whose pointer identity is not a
	// stable projection key.
	projection             types.BasicKind
	generated              *GeneratedArtifact
	interfaceContractType  types.Type
	interfaceContract      *types.Interface
	interfaceContractKey   string
	interfaceCompleteSet   bool
	providerProfileTarget  *GeneratedArtifact
	anonymousDemand        AnonymousStructDemand
	mapDemand              MapSpecializationDemand
	genericOperation       *GenericOperationContract
	genericParameter       *types.TypeParam
	genericFacet           GenericRepresentationFacet
	typeRepresentation     TypeRepresentationFacet
	concretizationDeferred bool
	enclosing              ast.Node
	callable               ast.Node
	control                CallableControlFacet
	controlLabel           *types.Label
	controlPosition        token.Pos
	controlRange           *ast.RangeStmt
	controlDefer           *ast.DeferStmt
}

func NewNamedStructOperationRequirement(
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	switch {
	case typeName == nil:
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation type is nil",
		}
	case !operation.Valid():
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     MustSourceArtifactOwner(typeName),
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
		typeName:  typeName,
	}, nil
}

func NewLexicalNamedStructOperationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	operation NamedStructOperation,
) (DeclarationRequirement, error) {
	if !validLexicalNamedStructOwner(owner, typeName) ||
		!operation.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "lexical named-struct operation is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     owner,
		kind:      DeclarationRequirementNamedStructOperation,
		operation: operation,
		typeName:  typeName,
	}, nil
}

func NewAnonymousStructRequirement(
	artifact *GeneratedArtifact,
	demand AnonymousStructDemand,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactAnonymousStruct ||
		!demand.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "anonymous-struct requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:           artifact.ReconstructionOwner(),
		kind:            DeclarationRequirementAnonymousStruct,
		generated:       artifact,
		anonymousDemand: demand,
	}, nil
}

func NewMapSpecializationRequirement(
	artifact *GeneratedArtifact,
	demand MapSpecializationDemand,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactMapSpecialization ||
		!demand.Valid() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "map-specialization requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      DeclarationRequirementMapSpecialization,
		generated: artifact,
		mapDemand: demand,
	}, nil
}

func NewInterfaceAdapterRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceAdapter,
		DeclarationRequirementInterfaceAdapter,
		"interface-adapter",
	)
}

func NewInterfaceAdapterContractRequirement(
	artifact *GeneratedArtifact,
	contractType types.Type,
	contract *types.Interface,
	contractKey string,
) (DeclarationRequirement, error) {
	selected, selectedOK := interfaceMethodSet(contractType)
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactInterfaceAdapter ||
		!selectedOK ||
		!types.Identical(selected, contract) ||
		!contract.Complete().IsMethodSet() ||
		contractKey == "" {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter contract requirement is invalid",
		}
	}
	sourceType, ok := artifact.InterfaceAdapterType()
	if !ok || !types.Implements(sourceType, contract) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter source does not implement its demanded contract",
		}
	}
	return DeclarationRequirement{
		owner:                 artifact.ReconstructionOwner(),
		kind:                  DeclarationRequirementInterfaceAdapter,
		generated:             artifact,
		interfaceContractType: contractType,
		interfaceContract:     contract,
		interfaceContractKey:  contractKey,
	}, nil
}

func NewInterfaceAdapterCompleteMethodSetRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactInterfaceAdapter {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter complete method-set requirement is invalid",
		}
	}
	if _, ok := artifact.InterfaceAdapterType(); !ok {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "interface-adapter complete method-set source is invalid",
		}
	}
	return DeclarationRequirement{
		owner:                artifact.ReconstructionOwner(),
		kind:                 DeclarationRequirementInterfaceAdapter,
		generated:            artifact,
		interfaceCompleteSet: true,
	}, nil
}

func interfaceMethodSet(sourceType types.Type) (*types.Interface, bool) {
	if sourceType == nil {
		return nil, false
	}
	selected, ok := types.Unalias(sourceType).Underlying().(*types.Interface)
	if !ok {
		return nil, false
	}
	selected = selected.Complete()
	return selected, selected.IsMethodSet()
}

func NewAnonymousInterfaceRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactAnonymousInterface,
		DeclarationRequirementAnonymousInterface,
		"anonymous-interface",
	)
}

func NewInterfaceMethodTokenRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceMethodToken,
		DeclarationRequirementInterfaceMethodToken,
		"interface-method-token",
	)
}

func NewInterfaceMethodCallableRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceMethodCallable,
		DeclarationRequirementInterfaceMethodCallable,
		"interface-method callable",
	)
}

func NewInterfaceDynamicTypeTokenRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactInterfaceDynamicTypeToken,
		DeclarationRequirementInterfaceDynamicTypeToken,
		"interface-dynamic-type-token",
	)
}

func NewReflectionTypeRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactReflectionType,
		DeclarationRequirementReflectionType,
		"reflection type",
	)
}

func NewProviderInterfaceBridgeRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactProviderInterfaceBridge,
		DeclarationRequirementProviderInterfaceBridge,
		"provider-interface bridge",
	)
}

func NewProviderInterfaceCapabilityRequirement(
	artifact *GeneratedArtifact,
	contract *types.Interface,
	contractKey string,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactProviderInterfaceBridge ||
		contract == nil ||
		!contract.Complete().IsMethodSet() ||
		contractKey == "" {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "provider-interface capability requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:                artifact.ReconstructionOwner(),
		kind:                 DeclarationRequirementProviderInterfaceCapability,
		generated:            artifact,
		interfaceContract:    contract,
		interfaceContractKey: contractKey,
	}, nil
}

func NewProviderProfileInterfaceCapabilityRequirement(
	artifact *GeneratedArtifact,
	target *GeneratedArtifact,
) (DeclarationRequirement, error) {
	if !artifact.Valid() ||
		artifact.Kind() != GeneratedArtifactProviderInterfaceBridge ||
		!target.Valid() ||
		target.Kind() != GeneratedArtifactProviderInterfaceBridge ||
		artifact == target {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "provider-profile interface capability requirement is invalid",
		}
	}
	base, baseProfile, baseOK := artifact.ProviderProfileInterfaceBridge()
	targetType, targetProfile, targetOK := target.ProviderProfileInterfaceBridge()
	if !baseOK || len(baseProfile) == 0 || !targetOK || len(targetProfile) == 0 {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "provider-profile interface capability bridge is invalid",
		}
	}
	baseContract, baseInterface := base.Underlying().(*types.Interface)
	targetContract, targetInterface := targetType.Underlying().(*types.Interface)
	if !baseInterface || !targetInterface ||
		!types.Implements(targetType, baseContract.Complete()) ||
		types.Implements(base, targetContract.Complete()) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "provider-profile interface capability is not a strict extension",
		}
	}
	return DeclarationRequirement{
		owner:                 artifact.ReconstructionOwner(),
		kind:                  DeclarationRequirementProviderProfileInterfaceCapability,
		generated:             artifact,
		providerProfileTarget: target,
	}, nil
}

func NewProviderProfileInterfaceCapabilityRequest(
	artifact *GeneratedArtifact,
	target *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewProviderProfileInterfaceCapabilityRequirement(
		artifact,
		target,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewProviderStatefulRepresentationRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactProviderStatefulRepresentation,
		DeclarationRequirementProviderStatefulRepresentation,
		"provider stateful representation",
	)
}

func NewDeferredCallableRegistryRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactDeferredCallableRegistry,
		DeclarationRequirementDeferredCallableRegistry,
		"deferred-callable registry",
	)
}

func NewDeferredCallableRegistryRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewDeferredCallableRegistryRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func newGeneratedDefinitionRequirement(
	artifact *GeneratedArtifact,
	artifactKind GeneratedArtifactKind,
	requirementKind DeclarationRequirementKind,
	name string,
) (DeclarationRequirement, error) {
	if !artifact.Valid() || artifact.Kind() != artifactKind {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: name + " requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:     artifact.ReconstructionOwner(),
		kind:      requirementKind,
		generated: artifact,
	}, nil
}

func NewCallableABIRequirement(
	artifact *GeneratedArtifact,
) (DeclarationRequirement, error) {
	return newGeneratedDefinitionRequirement(
		artifact,
		GeneratedArtifactCallableABI,
		DeclarationRequirementCallableABI,
		"callable ABI",
	)
}

func NewCallableABIRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewCallableABIRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewInterfaceMethodCallableRequest(
	artifact *GeneratedArtifact,
) (RootRequest, error) {
	requirement, err := NewInterfaceMethodCallableRequirement(artifact)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) CallableABI() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementCallableABI,
		GeneratedArtifactCallableABI,
	)
}

func MethodReceiverTypeName(method *types.Func) *types.TypeName {
	if method == nil {
		return nil
	}
	method = method.Origin()
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil
	}
	receiver := types.Unalias(signature.Recv().Type())
	if pointer, ok := receiver.(*types.Pointer); ok {
		receiver = types.Unalias(pointer.Elem())
	}
	named, ok := receiver.(*types.Named)
	if !ok || named.Obj() == nil {
		return nil
	}
	return named.Origin().Obj()
}

func ValueReceiverTypeName(method *types.Func) *types.TypeName {
	if method == nil {
		return nil
	}
	method = method.Origin()
	signature, ok := method.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return nil
	}
	if _, pointer := types.Unalias(signature.Recv().Type()).(*types.Pointer); pointer {
		return nil
	}
	return MethodReceiverTypeName(method)
}

func NewClassMethodRequirement(
	owner *types.TypeName,
	method *types.Func,
) (DeclarationRequirement, error) {
	if owner == nil ||
		method == nil ||
		method.Origin() != method ||
		method.Name() == "_" ||
		MethodReceiverTypeName(method) != owner {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "class-method requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner:       MustSourceArtifactOwner(owner),
		kind:        DeclarationRequirementClassMethod,
		typeName:    owner,
		classMethod: method,
	}, nil
}

func (r DeclarationRequirement) ClassMethod() (
	*types.TypeName,
	*types.Func,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementClassMethod {
		return nil, nil, false
	}
	return r.typeName, r.classMethod, true
}

func NewValueReceiverCopyRequirement(
	method *types.Func,
) (DeclarationRequirement, error) {
	if method == nil ||
		method.Origin() != method ||
		ValueReceiverTypeName(method) == nil {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "value-receiver copy requirement is invalid",
		}
	}
	return DeclarationRequirement{
		owner: MustSourceArtifactOwner(method),
		kind:  DeclarationRequirementValueReceiverCopy,
	}, nil
}

func (r DeclarationRequirement) ValueReceiverCopy() (
	*types.Func,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementValueReceiverCopy {
		return nil, false
	}
	method, ok := r.owner.Source()
	selected, methodOK := method.(*types.Func)
	return selected, ok && methodOK
}
