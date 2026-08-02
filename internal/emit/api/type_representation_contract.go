package api

import "go/types"

type TypeRepresentationFacet uint8

const (
	TypeRepresentationInvalid          TypeRepresentationFacet = 0
	TypeRepresentationStorage          TypeRepresentationFacet = 1
	TypeRepresentationContainerStorage TypeRepresentationFacet = 2
	TypeRepresentationPointer          TypeRepresentationFacet = 3
)

func (f TypeRepresentationFacet) Valid() bool {
	return f == TypeRepresentationStorage ||
		f == TypeRepresentationContainerStorage ||
		f == TypeRepresentationPointer
}

func (f TypeRepresentationFacet) String() string {
	switch f {
	case TypeRepresentationStorage:
		return "storage"
	case TypeRepresentationContainerStorage:
		return "container-storage"
	case TypeRepresentationPointer:
		return "pointer"
	default:
		return "invalid"
	}
}

func SupportsTypeRepresentation(typeName *types.TypeName) bool {
	if typeName == nil || typeName.IsAlias() {
		return false
	}
	named, ok := types.Unalias(typeName.Type()).(*types.Named)
	if !ok || named.Obj() != typeName || named.Origin() == nil {
		return false
	}
	switch named.Underlying().(type) {
	case *types.Basic,
		*types.Array,
		*types.Slice,
		*types.Pointer,
		*types.Signature,
		*types.Map,
		*types.Chan,
		*types.Struct:
		return true
	default:
		return false
	}
}

func NewTypeRepresentationRequirement(
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if !SupportsTypeRepresentation(typeName) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "type-representation owner is unsupported",
		}
	}
	return newTypeRepresentationRequirement(
		MustSourceArtifactOwner(typeName),
		typeName,
		nil,
		facet,
	)
}

func NewLexicalTypeRepresentationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if !SupportsTypeRepresentation(typeName) ||
		!validLexicalNamedStructOwner(owner, typeName) {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "lexical type-representation owner is invalid",
		}
	}
	return newTypeRepresentationRequirement(
		owner,
		typeName,
		nil,
		facet,
	)
}

func NewGeneratedTypeRepresentationRequirement(
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	if artifact == nil ||
		artifact.Kind() != GeneratedArtifactAnonymousStruct {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "generated type-representation owner is invalid",
		}
	}
	return newTypeRepresentationRequirement(
		artifact.ReconstructionOwner(),
		nil,
		artifact,
		facet,
	)
}

func newTypeRepresentationRequirement(
	owner ArtifactOwner,
	typeName *types.TypeName,
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (DeclarationRequirement, error) {
	requirement := DeclarationRequirement{
		owner:              owner,
		kind:               DeclarationRequirementTypeRepresentation,
		typeName:           typeName,
		generated:          artifact,
		typeRepresentation: facet,
	}
	if !requirement.validTypeRepresentation() {
		return DeclarationRequirement{}, &RootRequestError{
			Reason: "type-representation requirement is invalid",
		}
	}
	return requirement, nil
}

func NewTypeRepresentationRequest(
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewTypeRepresentationRequirement(typeName, facet)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewLexicalTypeRepresentationRequest(
	owner ArtifactOwner,
	typeName *types.TypeName,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewLexicalTypeRepresentationRequirement(
		owner,
		typeName,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func NewGeneratedTypeRepresentationRequest(
	artifact *GeneratedArtifact,
	facet TypeRepresentationFacet,
) (RootRequest, error) {
	requirement, err := NewGeneratedTypeRepresentationRequirement(
		artifact,
		facet,
	)
	if err != nil {
		return RootRequest{}, err
	}
	return newDeclarationRequirementRequest(requirement), nil
}

func (r DeclarationRequirement) TypeRepresentation() (
	*types.TypeName,
	*GeneratedArtifact,
	TypeRepresentationFacet,
	bool,
) {
	if !r.Valid() ||
		r.kind != DeclarationRequirementTypeRepresentation {
		return nil, nil, TypeRepresentationInvalid, false
	}
	return r.typeName, r.generated, r.typeRepresentation, true
}

func (r DeclarationRequirement) validTypeRepresentation() bool {
	if !r.owner.Valid() ||
		!r.typeRepresentation.Valid() ||
		r.operation != NamedStructOperationInvalid ||
		r.variable != nil ||
		r.constant != nil ||
		r.projection != types.Invalid ||
		r.anonymousDemand != AnonymousStructDemandInvalid ||
		r.mapDemand != MapSpecializationDemandInvalid ||
		r.genericOperation != nil ||
		r.genericParameter != nil ||
		r.genericFacet != GenericRepresentationInvalid {
		return false
	}
	if r.typeName != nil {
		if r.generated != nil || r.typeName.IsAlias() {
			return false
		}
		source, sourceOK := r.owner.Source()
		if sourceOK && source == r.typeName {
			return true
		}
		return validLexicalNamedStructOwner(r.owner, r.typeName)
	}
	return r.generated != nil &&
		r.generated.Kind() == GeneratedArtifactAnonymousStruct &&
		r.owner == r.generated.ReconstructionOwner()
}
