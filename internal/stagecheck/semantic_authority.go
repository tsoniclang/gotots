package stagecheck

import (
	"fmt"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifySemanticAuthorityProjections(
	model *semantic.Model,
	expected map[identity.PackageID]map[identity.DefinitionID]bool,
	loaded map[identity.PackageID]*source.LoadedPackage,
	plan *sourceplan.Plan,
) (
	map[identity.PackageID]semantic.AuthorityProjection,
	int,
	error,
) {
	projections := model.AuthorityProjections()
	if len(projections) != len(expected) {
		return nil, 0, semanticVerificationError(
			"authority",
			fmt.Sprintf(
				"semantic model has %d authority projections, expected %d",
				len(projections), len(expected),
			),
		)
	}
	out := make(
		map[identity.PackageID]semantic.AuthorityProjection,
		len(projections),
	)
	mixed := 0
	for _, projection := range projections {
		packageID := projection.ID()
		definitions, present := expected[packageID]
		if !present {
			return nil, 0, semanticVerificationError(
				"authority",
				"semantic model has unexpected package "+packageID.String(),
			)
		}
		if _, duplicate := out[packageID]; duplicate {
			return nil, 0, semanticVerificationError(
				"authority",
				"semantic model repeats package "+packageID.String(),
			)
		}
		pkg := loaded[packageID]
		if pkg == nil {
			return nil, 0, semanticVerificationError(
				"authority",
				"semantic package is absent from source "+packageID.String(),
			)
		}
		local, certified, err := independentSemanticAuthority(plan, pkg)
		if err != nil {
			return nil, 0, err
		}
		if projection.Provenance() !=
			verifiedSemanticProvenance(pkg.Provenance()) ||
			projection.HasLocalAuthority() != local ||
			projection.HasCertifiedAuthority() != certified {
			return nil, 0, semanticVerificationError(
				"authority",
				"semantic authority differs for "+packageID.String(),
			)
		}
		if err := verifyAuthorityDefinitionCensus(
			projection, definitions,
		); err != nil {
			return nil, 0, err
		}
		if local && certified {
			mixed++
		}
		out[packageID] = projection
	}
	return out, mixed, nil
}

func independentSemanticAuthority(
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
) (bool, bool, error) {
	local := false
	certified := false
	for _, file := range pkg.Files() {
		decision, present := plan.For(file.ID())
		if !present {
			return false, false, semanticVerificationError(
				"authority",
				"source plan omits semantic file "+file.ID().String(),
			)
		}
		switch decision.Kind() {
		case sourceplan.KindLocalSyntax:
			local = true
		case sourceplan.KindCertifiedGraph:
			certified = true
		default:
			return false, false, semanticVerificationError(
				"authority",
				"source plan has invalid semantic authority for "+
					file.ID().String(),
			)
		}
	}
	if synthetic, present := plan.SyntheticFor(pkg.ID()); present {
		switch synthetic.Kind() {
		case sourceplan.KindLocalSyntax:
			local = true
		case sourceplan.KindCertifiedGraph:
			certified = true
		default:
			return false, false, semanticVerificationError(
				"authority",
				"source plan has invalid synthetic semantic authority for "+
					pkg.ID().String(),
			)
		}
	}
	if !local && !certified {
		switch pkg.Disposition() {
		case source.DispositionBuiltinUniverse,
			source.DispositionUnsafeIntrinsic:
			local = true
		default:
			return false, false, semanticVerificationError(
				"authority",
				"semantic package has no selected authority "+pkg.ID().String(),
			)
		}
	}
	return local, certified, nil
}

func verifyAuthorityDefinitionCensus(
	projection semantic.AuthorityProjection,
	expected map[identity.DefinitionID]bool,
) error {
	actual := projection.ExpectedDefinitions()
	if len(actual) != len(expected) {
		return semanticVerificationError(
			"authority",
			fmt.Sprintf(
				"semantic package %s has %d definitions, expected %d",
				projection.ID(), len(actual), len(expected),
			),
		)
	}
	for _, definition := range actual {
		if !expected[definition] {
			return semanticVerificationError(
				"authority",
				"semantic package has unexpected definition "+
					definition.String(),
			)
		}
	}
	return nil
}

func verifiedSemanticProvenance(
	provenance source.Provenance,
) semantic.PackageProvenance {
	switch provenance {
	case source.ProvenanceWorkspaceModule:
		return semantic.ProvenanceWorkspaceModule
	case source.ProvenanceModuleDependency:
		return semantic.ProvenanceModuleDependency
	case source.ProvenanceStandardLibrary:
		return semantic.ProvenanceStandardLibrary
	case source.ProvenanceToolchainPackage:
		return semantic.ProvenanceToolchainPackage
	case source.ProvenanceLanguagePseudo:
		return semantic.ProvenanceLanguagePseudo
	default:
		return semantic.ProvenanceInvalid
	}
}
