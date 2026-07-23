package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/executable"
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
	localOnly bool,
) error {
	if actual.ID() != expected.pkg.ID() ||
		actual.Provenance().String() != expected.loaded.Provenance().String() {
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
	return verifySemanticOperations(expected, actual, localOnly)
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
	records := map[identity.DefinitionID]semantic.DefinitionSemantics{}
	for _, record := range actual.Definitions() {
		records[record.Definition()] = record
	}
	if len(records) != len(expectedDefinitions) {
		return semanticVerificationError(
			"definition",
			fmt.Sprintf(
				"package %s has %d records for %d definitions",
				actual.ID(), len(records), len(expectedDefinitions),
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
	for definition := range expectedDefinitions {
		record, present := records[definition]
		if !present {
			return semanticVerificationError(
				"definition", "missing "+definition.String(),
			)
		}
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
			record.Authority().ArtifactDigest() != provider.Digest() ||
			record.Authority().ShardDigest() != providerContext.ShardDigest ||
			record.Authority().StructuralSource() !=
				provider.StructuralArtifactDigest() {
			return semanticVerificationError(
				"authority", "provider authority differs for "+definition.String(),
			)
		}
	}
	return nil
}

func verifySemanticResolutions(
	expected semanticPackageExpectation,
	actual semantic.Package,
	localOnly bool,
) error {
	records := map[identity.OccurrenceID]semantic.OccurrenceResolution{}
	for _, record := range actual.Resolutions() {
		if localOnly &&
			!expected.localOccurrence(
				record.Occurrence(), record.Owner(),
			) {
			continue
		}
		records[record.Occurrence()] = record
	}
	if len(records) != len(expected.domains) {
		var missing []string
		var extra []string
		for occurrence := range expected.domains {
			if _, present := records[occurrence]; !present {
				missing = append(missing, occurrence.String())
			}
		}
		for occurrence := range records {
			if _, present := expected.domains[occurrence]; !present {
				extra = append(extra, occurrence.String())
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		return semanticVerificationError(
			"resolution",
			fmt.Sprintf(
				"package %s has %d records for %d retained occurrences; missing=%v extra=%v",
				actual.ID(), len(records), len(expected.domains),
				missing, extra,
			),
		)
	}
	for occurrenceID, domain := range expected.domains {
		record, present := records[occurrenceID]
		occurrence := expected.occurrences[occurrenceID]
		if !present {
			return semanticVerificationError(
				"resolution", "missing "+occurrenceID.String(),
			)
		}
		if record.Owner() != expected.owners[occurrenceID] ||
			record.Domain() != domain ||
			record.Syntax() != occurrence.Kind() ||
			record.Role() != occurrence.Role() {
			return semanticVerificationError(
				"resolution", "structural evidence differs for "+occurrenceID.String(),
			)
		}
	}
	return nil
}

func verifySemanticOperations(
	expected semanticPackageExpectation,
	actual semantic.Package,
	localOnly bool,
) error {
	resolutions := map[identity.OccurrenceID]semantic.OccurrenceResolution{}
	for _, record := range actual.Resolutions() {
		if localOnly &&
			!expected.localOccurrence(
				record.Occurrence(), record.Owner(),
			) {
			continue
		}
		resolutions[record.Occurrence()] = record
	}
	sourceOperations := 0
	implicitOperations := 0
	for _, operation := range actual.Operations() {
		if localOnly {
			if operation.ID().Source() {
				if !expected.localOccurrence(
					operation.Occurrence(),
					operation.Definition(),
				) {
					continue
				}
			} else {
				if _, local := expected.definitions[operation.Definition()]; !local {
					continue
				}
			}
		}
		if operation.ID().Source() {
			sourceOperations++
			occurrence, present := expected.occurrences[operation.Occurrence()]
			resolution := resolutions[operation.Occurrence()]
			if !present ||
				expected.domains[operation.Occurrence()] !=
					catalog.ResolutionDomainExecutable ||
				operation.Definition() !=
					expected.owners[operation.Occurrence()] ||
				operation.Syntax() != occurrence.Kind() ||
				operation.Role() != occurrence.Role() ||
				operation.Token() != occurrence.Token() ||
				resolution.Kind() != semantic.ResolutionOperation ||
				resolution.Operation() != operation.ID() {
				return semanticVerificationError(
					"operation", "source origin differs for "+operation.ID().String(),
				)
			}
			continue
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
	}
	resolvedOperations := 0
	for _, resolution := range actual.Resolutions() {
		if localOnly &&
			!expected.localOccurrence(
				resolution.Occurrence(),
				resolution.Owner(),
			) {
			continue
		}
		if resolution.Kind() == semantic.ResolutionOperation {
			resolvedOperations++
		}
	}
	if sourceOperations != resolvedOperations {
		return semanticVerificationError(
			"operation",
			fmt.Sprintf(
				"%d source operations differ from %d operation resolutions",
				sourceOperations, resolvedOperations,
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
) map[identity.PackageID]map[identity.DefinitionID]bool {
	out := map[identity.PackageID]map[identity.DefinitionID]bool{}
	for _, record := range graph.DefinitionCensus() {
		if out[record.Package()] == nil {
			out[record.Package()] =
				map[identity.DefinitionID]bool{}
		}
		out[record.Package()][record.ID()] = true
	}
	return out
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
		return out[left].String() < out[right].String()
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

func semanticAdditionalOccurrences(
	inventory *executable.Inventory,
) map[identity.OccurrenceID]structure.Occurrence {
	out := map[identity.OccurrenceID]structure.Occurrence{}
	for _, occurrence := range inventory.AdditionalOccurrences() {
		out[occurrence.ID()] = occurrence
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
	authority, _ := semantic.NewCheckerAuthority(
		universe.Toolchain().BinaryDigest(),
		universe.Toolchain().BuildConfigurationDigest(),
		loaded.ProviderInputFingerprint(),
		structure.PackageDigest(pkg),
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
	for _, fact := range facts.Facts() {
		if definitions[fact.ID().Definition()] {
			fmt.Fprintf(
				hash, "%s|%t|%s|%s\n",
				fact.ID(), fact.Value(),
				fact.ProducerDigest(), fact.EvidenceDigest(),
			)
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func semanticVerificationError(stage string, reason string) error {
	return &VerificationError{
		Stage:  "typed-frontend/" + stage,
		Reason: reason,
	}
}

func sortedSemanticDefinitionIDs(
	records map[identity.DefinitionID]semantic.DefinitionSemantics,
) []identity.DefinitionID {
	out := make([]identity.DefinitionID, 0, len(records))
	for definition := range records {
		out = append(out, definition)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].String() < out[right].String()
	})
	return out
}
