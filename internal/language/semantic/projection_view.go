package semantic

import "github.com/tsoniclang/gotots/internal/identity"

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
	if len(model.projections) == 0 {
		out := make([]AuthorityProjection, 0, len(model.packages))
		for _, pkg := range model.packages {
			definitions := make(
				[]identity.DefinitionID, 0, len(pkg.definitions),
			)
			for _, definition := range pkg.definitions {
				definitions = append(
					definitions, definition.Definition(),
				)
			}
			out = append(out, AuthorityProjection{
				id: pkg.id, provenance: pkg.provenance,
				expectedDefinitions: definitions,
				local:               true,
			})
		}
		return out
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
			local:     !projection.local.ID.IsZero(),
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
