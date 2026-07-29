package api

import "go/types"

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
	*types.Func,
	*types.Var,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementAddressableStorage {
		return nil, nil, false
	}
	source, sourceOK := r.owner.Source()
	owner, ok := source.(*types.Func)
	return owner, r.variable, sourceOK && ok
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

func (r DeclarationRequirement) InterfaceDynamicTypeToken() (
	*GeneratedArtifact,
	bool,
) {
	return r.generatedDefinition(
		DeclarationRequirementInterfaceDynamicTypeToken,
		GeneratedArtifactInterfaceDynamicTypeToken,
	)
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
		DeclarationRequirementInterfaceDynamicTypeToken:
		return r.generated, true
	default:
		return nil, false
	}
}
