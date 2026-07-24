package semantic

import "github.com/tsoniclang/gotots/internal/identity"

type PackageCensus struct {
	Package            identity.PackageID
	Definitions        []identity.DefinitionID
	Declarations       []identity.SemanticDeclarationID
	MemberTargetCount  int
	MemberTargetDigest string
}

func (model *Model) PackageCensus(
	packageID identity.PackageID,
) (PackageCensus, bool, error) {
	if model == nil || packageID.IsZero() {
		return PackageCensus{}, false, nil
	}
	index := searchCanonical(
		model.projections,
		func(projection packageProjection) identity.PackageID {
			return projection.id
		},
		packageID,
	)
	if index == len(model.projections) ||
		model.projections[index].id != packageID {
		return PackageCensus{}, false, nil
	}
	projection := model.projections[index]
	if projection.certified {
		context, present, err := model.provider.PackageContext(packageID)
		if err != nil || !present {
			return PackageCensus{}, present, err
		}
		return PackageCensus{
			Package: packageID,
			Definitions: append(
				[]identity.DefinitionID(nil), context.Definitions...,
			),
			Declarations: append(
				[]identity.SemanticDeclarationID(nil),
				context.Declarations...,
			),
			MemberTargetCount:  context.MemberTargetCount,
			MemberTargetDigest: context.MemberTargetDigest,
		}, true, nil
	}
	context, present, err := model.checker.PackageContext(packageID)
	if err != nil || !present {
		return PackageCensus{}, present, err
	}
	return PackageCensus{
		Package: packageID,
		Definitions: append(
			[]identity.DefinitionID(nil), context.Definitions...,
		),
		Declarations: append(
			[]identity.SemanticDeclarationID(nil),
			context.Declarations...,
		),
		MemberTargetCount:  context.MemberTargetCount,
		MemberTargetDigest: context.MemberTargetDigest,
	}, true, nil
}

// AuthorityProjection is the immutable identity-only view of one logical
// semantic package. It exposes selected authority without opening provider
// detail.
type AuthorityProjection struct {
	id                  identity.PackageID
	provenance          PackageProvenance
	expectedDefinitions []identity.DefinitionID
	local               bool
	certified           bool
}

func (projection AuthorityProjection) ID() identity.PackageID {
	return projection.id
}

func (projection AuthorityProjection) Provenance() PackageProvenance {
	return projection.provenance
}

func (projection AuthorityProjection) ExpectedDefinitions() []identity.DefinitionID {
	return append(
		[]identity.DefinitionID(nil),
		projection.expectedDefinitions...,
	)
}

func (projection AuthorityProjection) HasLocalAuthority() bool {
	return projection.local
}

func (projection AuthorityProjection) HasCertifiedAuthority() bool {
	return projection.certified
}

func (model *Model) AuthorityProjections() []AuthorityProjection {
	if model == nil {
		return nil
	}
	out := make(
		[]AuthorityProjection, 0, len(model.projections),
	)
	for _, projection := range model.projections {
		out = append(out, AuthorityProjection{
			id: projection.id, provenance: projection.provenance,
			expectedDefinitions: append(
				[]identity.DefinitionID(nil),
				projection.expectedDefinitions...,
			),
			local:     projection.local,
			certified: projection.certified,
		})
	}
	return out
}

type ProjectionStats struct {
	Packages          int
	LocalPackages     int
	CertifiedPackages int
	MixedPackages     int
}

func (model *Model) ProjectionStats() ProjectionStats {
	var stats ProjectionStats
	for _, projection := range model.AuthorityProjections() {
		stats.Packages++
		if projection.local {
			stats.LocalPackages++
		}
		if projection.certified {
			stats.CertifiedPackages++
		}
		if projection.local && projection.certified {
			stats.MixedPackages++
		}
	}
	return stats
}
