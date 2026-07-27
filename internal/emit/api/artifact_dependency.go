package api

import (
	"fmt"
	"go/types"
)

type ArtifactFacet uint8

const (
	ArtifactFacetInvalid             ArtifactFacet = 0
	ArtifactFacetCallableSignature   ArtifactFacet = 1
	ArtifactFacetConstructorSurface  ArtifactFacet = 2
	ArtifactFacetInstanceTypeSurface ArtifactFacet = 3
	ArtifactFacetStaticSurface       ArtifactFacet = 4
	ArtifactFacetValueSurface        ArtifactFacet = 5
)

func (f ArtifactFacet) Valid() bool {
	return f >= ArtifactFacetCallableSignature &&
		f <= ArtifactFacetValueSurface
}

func (f ArtifactFacet) String() string {
	switch f {
	case ArtifactFacetCallableSignature:
		return "callable-signature"
	case ArtifactFacetConstructorSurface:
		return "constructor-surface"
	case ArtifactFacetInstanceTypeSurface:
		return "instance-type-surface"
	case ArtifactFacetStaticSurface:
		return "static-surface"
	case ArtifactFacetValueSurface:
		return "value-surface"
	default:
		return fmt.Sprintf("artifact-facet(%d)", f)
	}
}

type ArtifactDependency struct {
	provider types.Object
	facet    ArtifactFacet
}

func NewArtifactDependency(
	provider types.Object,
	facet ArtifactFacet,
) (ArtifactDependency, error) {
	if provider == nil {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency provider is nil"}
	}
	if !facet.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency facet is invalid"}
	}
	return ArtifactDependency{provider: provider, facet: facet}, nil
}

func (d ArtifactDependency) Valid() bool {
	return d.provider != nil && d.facet.Valid()
}

func (d ArtifactDependency) Provider() types.Object {
	return d.provider
}

func (d ArtifactDependency) Facet() ArtifactFacet {
	return d.facet
}
