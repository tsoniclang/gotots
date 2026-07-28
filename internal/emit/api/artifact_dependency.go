package api

import (
	"fmt"
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
	provider ArtifactOwner
	facet    ArtifactFacet
}

func NewArtifactDependency(
	provider ArtifactOwner,
	facet ArtifactFacet,
) (ArtifactDependency, error) {
	if !provider.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency provider is invalid"}
	}
	if !facet.Valid() {
		return ArtifactDependency{},
			&RootRequestError{Reason: "artifact dependency facet is invalid"}
	}
	return ArtifactDependency{provider: provider, facet: facet}, nil
}

func (d ArtifactDependency) Valid() bool {
	return d.provider.Valid() && d.facet.Valid()
}

func (d ArtifactDependency) Provider() ArtifactOwner {
	return d.provider
}

func (d ArtifactDependency) Facet() ArtifactFacet {
	return d.facet
}
