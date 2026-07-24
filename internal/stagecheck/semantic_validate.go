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
	if err := verifySemanticResolutions(
		expected, actual, localOnly,
	); err != nil {
		return err
	}
	if err := verifySemanticOperations(
		expected, actual, localOnly,
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
	checkerAuthority := expectedCheckerAuthority(
		universe, expected.pkg, expected.loaded, facts,
	)
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

func verifySemanticResolutions(
	expected semanticPackageExpectation,
	actual semantic.Package,
	localOnly bool,
) error {
	actualCount := 0
	if err := actual.VisitResolutions(func(
		record semantic.OccurrenceResolution,
	) error {
		if localOnly &&
			!expected.localOccurrence(
				record.Occurrence(), record.Owner(),
			) {
			return nil
		}
		actualCount++
		occurrence, present := expected.occurrences.get(record.Occurrence())
		if !present {
			return semanticVerificationError(
				"resolution",
				"unexpected "+record.Occurrence().String(),
			)
		}
		if record.Owner() !=
			expected.definitionID(occurrence.owner) ||
			record.Domain() != occurrence.domain ||
			record.Syntax() != occurrence.Kind() ||
			record.Role() != occurrence.Role() {
			return semanticVerificationError(
				"resolution",
				"structural evidence differs for "+
					record.Occurrence().String(),
			)
		}
		return nil
	}); err != nil {
		return err
	}
	if actualCount != expected.domainCount {
		return semanticVerificationError(
			"resolution",
			fmt.Sprintf(
				"package %s has %d records for %d retained occurrences",
				actual.ID(), actualCount, expected.domainCount,
			),
		)
	}
	return nil
}

func verifySemanticOperations(
	expected semanticPackageExpectation,
	actual semantic.Package,
	localOnly bool,
) error {
	sourceOperations := 0
	implicitOperations := 0
	resolvedOperations := map[identity.OperationID]bool{}
	if err := actual.VisitResolutions(func(
		resolution semantic.OccurrenceResolution,
	) error {
		if localOnly &&
			!expected.localOccurrence(
				resolution.Occurrence(),
				resolution.Owner(),
			) {
			return nil
		}
		if resolution.Kind() != semantic.ResolutionOperation {
			return nil
		}
		if resolvedOperations[resolution.Operation()] {
			return semanticVerificationError(
				"operation",
				"operation resolution is duplicated "+
					resolution.Operation().String(),
			)
		}
		resolvedOperations[resolution.Operation()] = true
		return nil
	}); err != nil {
		return err
	}
	if err := actual.VisitOperations(func(
		operation semantic.Operation,
	) error {
		if localOnly {
			if operation.ID().Source() {
				if !expected.localOccurrence(
					operation.Occurrence(),
					operation.Definition(),
				) {
					return nil
				}
			} else {
				if _, local := expected.definitions[operation.Definition()]; !local {
					return nil
				}
			}
		}
		if operation.ID().Source() {
			sourceOperations++
			occurrence, present := expected.occurrences.get(
				operation.Occurrence(),
			)
			if !present ||
				occurrence.domain !=
					catalog.ResolutionDomainExecutable ||
				operation.Definition() !=
					expected.definitionID(occurrence.owner) ||
				operation.Syntax() != occurrence.Kind() ||
				operation.Role() != occurrence.Role() ||
				operation.Token() != occurrence.Token() ||
				!resolvedOperations[operation.ID()] {
				return semanticVerificationError(
					"operation", "source origin differs for "+operation.ID().String(),
				)
			}
			delete(resolvedOperations, operation.ID())
			return nil
		}
		implicitOperations++
		if operation.Kind() !=
			semantic.OperationPackageInitialization ||
			operation.ID().ImplicitOp() !=
				identity.ImplicitDefinitionPackageInit {
			return semanticVerificationError(
				"operation", "invalid implicit origin "+operation.ID().String(),
			)
		}
		return nil
	}); err != nil {
		return err
	}
	if len(resolvedOperations) != 0 {
		return semanticVerificationError(
			"operation",
			fmt.Sprintf(
				"%d source operations leave %d unresolved operation references",
				sourceOperations, len(resolvedOperations),
			),
		)
	}
	implicitDefinitions := 0
	for definition := range expected.definitions {
		if definition.ImplicitOp().Valid() &&
			expected.executable[definition] {
			implicitDefinitions++
		}
	}
	if implicitOperations != implicitDefinitions {
		return semanticVerificationError(
			"operation",
			fmt.Sprintf(
				"%d implicit operations differ from %d implicit definitions",
				implicitOperations, implicitDefinitions,
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
) semantic.Authority {
	structureDigest := structure.PackageDigest(pkg)
	if loaded.Disposition() == source.DispositionBuiltinUniverse {
		structureDigest = catalog.StructureDigest()
	}
	authority, _ := semantic.NewCheckerAuthority(
		universe.Toolchain().BinaryDigest(),
		universe.Toolchain().BuildConfigurationDigest(),
		loaded.ProviderInputFingerprint(),
		structureDigest,
		semanticSelectionDigest(pkg, facts),
	)
	return authority
}

func semanticSelectionDigest(
	pkg structure.PackageGraph,
	facts *selectionfacts.Artifact,
) string {
	definitions := map[identity.DefinitionID]bool{}
	for _, definition := range pkg.Definitions() {
		definitions[definition.ID()] = true
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
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func semanticVerificationError(stage string, reason string) error {
	return &VerificationError{
		Stage:  "typed-frontend/" + stage,
		Reason: reason,
	}
}
