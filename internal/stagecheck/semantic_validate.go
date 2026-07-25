package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifySemanticPackage(
	expected semanticPackageExpectation,
	expectedDefinitions map[identity.DefinitionID]bool,
	actual semantic.Package,
	universe *source.Universe,
	plan *sourceplan.Plan,
	facts *selectionfacts.Artifact,
	provider *semantic.ProviderArtifact,
	index *structure.TransientIndex,
	localOnly bool,
) error {
	if actual.ID() != expected.id ||
		actual.Provenance() != verifiedSemanticProvenance(
			expected.loaded.Provenance(),
		) {
		return semanticVerificationError(
			"package", "semantic package identity or provenance differs",
		)
	}
	if err := verifySemanticDefinitions(
		expected,
		expectedDefinitions,
		actual,
		universe,
		plan,
		facts,
		provider,
	); err != nil {
		return err
	}
	if err := verifyCheckerSemanticPackage(
		expected, actual, universe, index, localOnly,
	); err != nil {
		return err
	}
	return verifyIntrinsicSemanticPackage(
		actual, expected, universe, facts,
	)
}

func verifySemanticDefinitions(
	expected semanticPackageExpectation,
	expectedDefinitions map[identity.DefinitionID]bool,
	actual semantic.Package,
	universe *source.Universe,
	plan *sourceplan.Plan,
	facts *selectionfacts.Artifact,
	provider *semantic.ProviderArtifact,
) error {
	if actual.DefinitionCount() != len(expectedDefinitions) {
		return semanticVerificationError(
			"definition",
			fmt.Sprintf(
				"package %s has %d records for %d definitions",
				actual.ID(),
				actual.DefinitionCount(),
				len(expectedDefinitions),
			),
		)
	}
	checkerAuthority, err := expectedCheckerAuthority(
		universe, expected.pkg, expected.loaded, facts,
	)
	if err != nil {
		return err
	}
	var providerContext semantic.ProviderPackageContext
	if provider != nil {
		providerContext, _, _ = provider.PackageContext(actual.ID())
	}
	seen := 0
	if err := actual.VisitDefinitions(func(
		record semantic.DefinitionSemantics,
	) error {
		definition := record.Definition()
		if !expectedDefinitions[definition] {
			return semanticVerificationError(
				"definition", "unexpected "+definition.String(),
			)
		}
		seen++
		if record.Package() != actual.ID() ||
			record.Form() != expectedDefinitionForm(definition) {
			return semanticVerificationError(
				"definition", "form or package differs for "+definition.String(),
			)
		}
		local := semanticDefinitionUsesLocal(
			plan, expected.loaded, definition,
		)
		if local {
			if record.Authority() != checkerAuthority {
				return semanticVerificationError(
					"authority", "checker authority differs for "+definition.String(),
				)
			}
		} else if record.Authority().Kind() !=
			semantic.AuthorityCertifiedProvider ||
			provider == nil ||
			record.Authority().ArtifactDigest() != provider.Digest() ||
			record.Authority().ShardDigest() != providerContext.ShardDigest ||
			record.Authority().StructuralSource() !=
				provider.StructuralArtifactDigest() {
			return semanticVerificationError(
				"authority", "provider authority differs for "+definition.String(),
			)
		}
		return nil
	}); err != nil {
		return err
	}
	if seen != len(expectedDefinitions) {
		return semanticVerificationError(
			"definition",
			fmt.Sprintf(
				"package %s visited %d records for %d definitions",
				actual.ID(), seen, len(expectedDefinitions),
			),
		)
	}
	return nil
}

func (expected semanticPackageExpectation) localOccurrence(
	occurrence identity.OccurrenceID,
	owner identity.DefinitionID,
) bool {
	if expected.localFiles[occurrence.Span().File()] {
		return true
	}
	_, local := expected.definitions[owner]
	return !owner.IsZero() && local
}

func loadedSemanticPackages(
	universe *source.Universe,
) map[identity.PackageID]*source.LoadedPackage {
	out := map[identity.PackageID]*source.LoadedPackage{}
	for _, pkg := range universe.Packages() {
		out[pkg.ID()] = pkg
	}
	return out
}

func semanticDefinitionCensus(
	graph *structure.Graph,
	loaded map[identity.PackageID]*source.LoadedPackage,
) (
	map[identity.PackageID]map[identity.DefinitionID]bool,
	error,
) {
	out := map[identity.PackageID]map[identity.DefinitionID]bool{}
	if err := graph.VisitResidentPackages(func(
		pkg structure.PackageGraph,
	) error {
		out[pkg.ID()] = map[identity.DefinitionID]bool{}
		return nil
	}); err != nil {
		return nil, semanticVerificationError(
			"package", err.Error(),
		)
	}
	for _, record := range graph.DefinitionCensus() {
		if out[record.Package()] == nil {
			return nil, semanticVerificationError(
				"package",
				"definition census has no package "+
					record.Package().String(),
			)
		}
		out[record.Package()][record.ID()] = true
	}
	for packageID, pkg := range loaded {
		if pkg.Disposition() != source.DispositionBuiltinUniverse {
			continue
		}
		if _, duplicate := out[packageID]; duplicate {
			return nil, semanticVerificationError(
				"package",
				"builtin package also has structural ownership "+
					packageID.String(),
			)
		}
		out[packageID] = map[identity.DefinitionID]bool{}
	}
	return out, nil
}

func semanticDefinitionSet(
	records map[identity.DefinitionID]structure.ImplementationDefinition,
) map[identity.DefinitionID]bool {
	out := make(map[identity.DefinitionID]bool, len(records))
	for definition := range records {
		out[definition] = true
	}
	return out
}

func sortedSemanticPackages(
	census map[identity.PackageID]map[identity.DefinitionID]bool,
) []identity.PackageID {
	out := make([]identity.PackageID, 0, len(census))
	for packageID := range census {
		out = append(out, packageID)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].Compare(out[right]) < 0
	})
	return out
}

func semanticSelections(
	selections *scope.DefinitionSelections,
) map[identity.DefinitionID]scope.DefinitionSelection {
	out := map[identity.DefinitionID]scope.DefinitionSelection{}
	for _, selection := range selections.Records() {
		out[selection.Definition()] = selection
	}
	return out
}

func expectedDefinitionForm(
	definition identity.DefinitionID,
) semantic.DefinitionForm {
	if definition.SyntheticRole().Valid() {
		return semantic.DefinitionFormSynthetic
	}
	switch definition.Kind() {
	case identity.DefinitionFuncDecl,
		identity.DefinitionFuncLit:
		return semantic.DefinitionFormCallable
	case identity.DefinitionPackageInitializer:
		return semantic.DefinitionFormInitializer
	case identity.DefinitionBodylessDecl:
		return semantic.DefinitionFormBodyless
	case identity.DefinitionImplicit:
		return semantic.DefinitionFormImplicit
	default:
		return semantic.DefinitionFormInvalid
	}
}

func semanticDefinitionUsesLocal(
	plan *sourceplan.Plan,
	pkg *source.LoadedPackage,
	definition identity.DefinitionID,
) bool {
	if plan.Purpose() == sourceplan.PurposeProviderProduction {
		return false
	}
	if file := definition.File(); !file.IsZero() {
		decision, present := plan.For(file)
		return present && decision.Kind() == sourceplan.KindLocalSyntax
	}
	if definition.SyntheticRole().Valid() {
		decision, present := plan.SyntheticFor(pkg.ID())
		return present && decision.Kind() == sourceplan.KindLocalSyntax
	}
	if definition.ImplicitOp().Valid() {
		for _, file := range pkg.Files() {
			decision, present := plan.For(file.ID())
			if present &&
				decision.Kind() == sourceplan.KindCertifiedGraph {
				return false
			}
		}
	}
	return true
}

func expectedCheckerAuthority(
	universe *source.Universe,
	pkg structure.PackageGraph,
	loaded *source.LoadedPackage,
	facts *selectionfacts.Artifact,
) (semantic.Authority, error) {
	structureDigest := structure.PackageDigest(pkg)
	if loaded.Disposition() == source.DispositionBuiltinUniverse {
		structureDigest = catalog.StructureDigest()
	}
	selectionDigest, err := semanticSelectionDigest(pkg, facts)
	if err != nil {
		return semantic.Authority{}, err
	}
	return semantic.NewCheckerAuthority(
		universe.Toolchain().BinaryDigest(),
		universe.Toolchain().BuildConfigurationDigest(),
		loaded.ProviderInputFingerprint(),
		structureDigest,
		selectionDigest,
	)
}

func semanticSelectionDigest(
	pkg structure.PackageGraph,
	facts *selectionfacts.Artifact,
) (string, error) {
	definitions := map[identity.DefinitionID]bool{}
	if err := pkg.VisitDefinitions(func(
		definition structure.ImplementationDefinition,
	) error {
		definitions[definition.ID()] = true
		return nil
	}); err != nil {
		return "", err
	}
	hash := sha256.New()
	fmt.Fprintln(hash, "gotots-semantic-selection/v1")
	_ = facts.VisitFacts(func(fact selectionfacts.Fact) error {
		if definitions[fact.ID().Definition()] {
			fmt.Fprintf(
				hash, "%s|%t|%s|%s\n",
				fact.ID(), fact.Value(),
				fact.ProducerDigest(), fact.EvidenceDigest(),
			)
		}
		return nil
	})
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func semanticVerificationError(stage string, reason string) error {
	return &VerificationError{
		Stage:  "typed-frontend/" + stage,
		Reason: reason,
	}
}
