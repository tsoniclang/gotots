package api

import (
	"go/ast"
	"go/token"
	"go/types"
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

func (r DeclarationRequirement) ProviderInterfaceBridge() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementProviderInterfaceBridge,
		GeneratedArtifactProviderInterfaceBridge,
	)
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
		DeclarationRequirementProviderStatefulRepresentation,
		DeclarationRequirementDeferredCallableRegistry,
		DeclarationRequirementGenericConcretization,
		DeclarationRequirementReflectionType:
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
