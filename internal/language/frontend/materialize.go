package frontend

import (
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

type Result struct {
	model   *semantic.Model
	work    Work
	metrics semantic.Metrics
}

func (result *Result) Model() *semantic.Model { return result.model }
func (result *Result) Work() Work             { return result.work }
func (result *Result) Metrics() semantic.Metrics {
	return result.metrics
}

func Materialize(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	plan *sourceplan.Plan,
	provider *semantic.ProviderArtifact,
) (*Result, error) {
	stage, err := newStageInput(
		universe,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		plan,
		false,
	)
	if err != nil {
		return nil, err
	}
	packages := map[identity.PackageID]semantic.Package{}
	work := Work{}
	for _, input := range stage.packageList {
		pkg, packageWork, err := materializePackage(stage, input)
		if err != nil {
			return nil, fmt.Errorf(
				"materialize semantic package %s: %w",
				input.id, err,
			)
		}
		packages[pkg.ID()] = pkg
		work.merge(packageWork)
	}
	localPackages := make([]semantic.Package, 0, len(packages))
	for _, pkg := range packages {
		localPackages = append(localPackages, pkg)
	}
	metrics, err := semantic.MeasurePackages(localPackages)
	if err != nil {
		return nil, err
	}
	projections, err := buildPackageProjections(
		stage, packages,
	)
	if err != nil {
		return nil, err
	}
	model, err := semantic.NewProjectedModel(projections, provider)
	if err != nil {
		return nil, err
	}
	return &Result{
		model: model, work: work, metrics: metrics,
	}, nil
}

func MaterializeProviderPackage(
	universe *source.Universe,
	graph *structure.Graph,
	index *structure.TransientIndex,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	plan *sourceplan.Plan,
) (semantic.Package, Work, error) {
	stage, err := newStageInput(
		universe,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		plan,
		true,
	)
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	if len(stage.packageList) != 1 {
		return semantic.Package{}, Work{}, fmt.Errorf(
			"semantic provider derivation has %d packages",
			len(stage.packageList),
		)
	}
	return materializePackage(stage, stage.packageList[0])
}

func buildPackageProjections(
	stage *stageInput,
	local map[identity.PackageID]semantic.Package,
) ([]semantic.PackageProjectionInput, error) {
	expected := map[identity.PackageID][]identity.DefinitionID{}
	for _, record := range stage.graph.DefinitionCensus() {
		expected[record.Package()] = append(
			expected[record.Package()], record.ID(),
		)
	}
	loaded := map[identity.PackageID]*source.LoadedPackage{}
	for _, pkg := range stage.universe.Packages() {
		loaded[pkg.ID()] = pkg
	}
	for packageID := range local {
		if _, present := expected[packageID]; present {
			continue
		}
		pkg := loaded[packageID]
		if pkg == nil ||
			pkg.Disposition() != source.DispositionBuiltinUniverse {
			return nil, fmt.Errorf(
				"semantic package %s has no structural census",
				packageID,
			)
		}
		expected[packageID] = nil
	}
	packageIDs := make(
		[]identity.PackageID, 0, len(expected),
	)
	for packageID := range expected {
		packageIDs = append(packageIDs, packageID)
	}
	sort.Slice(packageIDs, func(left, right int) bool {
		return packageIDs[left].String() <
			packageIDs[right].String()
	})
	out := make(
		[]semantic.PackageProjectionInput, 0, len(packageIDs),
	)
	for _, packageID := range packageIDs {
		loadedPackage := loaded[packageID]
		if loadedPackage == nil {
			return nil, fmt.Errorf(
				"semantic projection package %s is absent from source",
				packageID,
			)
		}
		certified := packageUsesCertifiedSemantics(
			stage.plan, loadedPackage,
		)
		input := semantic.PackageProjectionInput{
			ID: packageID,
			Provenance: semanticProvenance(
				loadedPackage.Provenance(),
			),
			ExpectedDefinitions: expected[packageID],
			Certified:           certified,
		}
		if pkg, present := local[packageID]; present {
			input.Local = &pkg
			localFiles, declarations, err :=
				localPackageProjection(
					stage.plan, loadedPackage, pkg,
				)
			if err != nil {
				return nil, err
			}
			input.LocalFiles = localFiles
			input.LocalDeclarations = declarations
			if synthetic, present := stage.plan.SyntheticFor(
				packageID,
			); present {
				input.LocalSynthetic =
					synthetic.Kind() == sourceplan.KindLocalSyntax
			}
		}
		out = append(out, input)
	}
	return out, nil
}

func localPackageProjection(
	plan *sourceplan.Plan,
	loaded *source.LoadedPackage,
	pkg semantic.Package,
) (
	[]identity.FileID,
	[]identity.SemanticDeclarationID,
	error,
) {
	var files []identity.FileID
	for _, file := range loaded.Files() {
		decision, present := plan.For(file.ID())
		if !present {
			return nil, nil, fmt.Errorf(
				"semantic source plan omits %s", file.ID(),
			)
		}
		if decision.Kind() == sourceplan.KindLocalSyntax {
			files = append(files, file.ID())
		}
	}
	var declarations []identity.SemanticDeclarationID
	if !packageUsesCertifiedSemantics(plan, loaded) {
		for _, declaration := range pkg.Declarations() {
			declarations = append(declarations, declaration.ID())
		}
		return files, declarations, nil
	}
	selectedDeclarations := map[identity.SemanticDeclarationID]bool{}
	syntheticDeclarations := map[identity.SemanticDeclarationID]bool{}
	for _, definition := range pkg.Definitions() {
		for _, declaration := range definition.Spec().Declarations {
			selectedDeclarations[declaration] = true
			if definition.Form() == semantic.DefinitionFormSynthetic {
				syntheticDeclarations[declaration] = true
			}
		}
	}
	for _, resolution := range pkg.Resolutions() {
		var declaration identity.SemanticDeclarationID
		switch resolution.Kind() {
		case semantic.ResolutionDeclaration:
			declaration = resolution.Declaration()
		case semantic.ResolutionStructuralOnly:
			if resolution.Structural().Disposition() ==
				semantic.StructuralCompileTimeExpression {
				continue
			}
			declaration = resolution.Structural().Declaration()
		}
		if declaration.IsZero() ||
			resolution.Role() != catalog.RoleDeclarationName {
			continue
		}
		selectedDeclarations[declaration] = true
	}
	for _, declaration := range pkg.Declarations() {
		if declaration.ID().Form() ==
			identity.SemanticDeclarationPredeclared {
			selectedDeclarations[declaration.ID()] = true
			continue
		}
		if syntheticDeclarations[declaration.ID()] {
			decision, present := plan.SyntheticFor(loaded.ID())
			if !present {
				return nil, nil, fmt.Errorf(
					"semantic synthetic declaration %s has no source plan",
					declaration.ID(),
				)
			}
			if decision.Kind() != sourceplan.KindLocalSyntax {
				delete(selectedDeclarations, declaration.ID())
			}
			continue
		}
		if declaration.ID().Form() ==
			identity.SemanticDeclarationOccurrence {
			source := declaration.ID().Occurrence()
			decision, present := plan.For(source.Span().File())
			if !present ||
				decision.Kind() != sourceplan.KindLocalSyntax {
				delete(selectedDeclarations, declaration.ID())
			}
		}
	}
	for _, declaration := range pkg.Declarations() {
		if selectedDeclarations[declaration.ID()] {
			declarations = append(declarations, declaration.ID())
		}
	}
	return files, declarations, nil
}

type packageBuilder struct {
	input                 *packageInput
	contexts              *contextIndex
	objects               *objectIndex
	types                 *typeBuilder
	draft                 *semantic.PackageDraft
	operationByOccurrence map[identity.OccurrenceID]identity.OperationID
	variantByOccurrence   map[identity.OccurrenceID]catalog.Variant
	resolvedOccurrences   map[identity.OccurrenceID]struct{}
	definitionByRoot      map[identity.OccurrenceID]identity.DefinitionID
}

func materializePackage(
	stage *stageInput,
	input *packageInput,
) (semantic.Package, Work, error) {
	work := input.work
	work.Packages = 1
	contexts, err := buildContexts(input, &work)
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	objects, err := buildObjectIndex(
		stage, input, contexts, &work,
	)
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	draft, err := semantic.NewPackageDraft(
		input.id,
		input.provenance,
		packageDraftCapacity(input, objects),
	)
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	builder := &packageBuilder{
		input: input, contexts: contexts, objects: objects,
		types:                 objects.typeBuilder,
		draft:                 draft,
		operationByOccurrence: map[identity.OccurrenceID]identity.OperationID{},
		variantByOccurrence:   map[identity.OccurrenceID]catalog.Variant{},
		resolvedOccurrences:   map[identity.OccurrenceID]struct{}{},
		definitionByRoot:      map[identity.OccurrenceID]identity.DefinitionID{},
	}
	for definition := range input.definitions {
		if !definition.Root().IsZero() {
			builder.definitionByRoot[definition.Root()] = definition
		}
	}
	if err := builder.resolveOccurrences(); err != nil {
		return semantic.Package{}, Work{}, err
	}
	declarations, err := objects.declarationRecords()
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	bindings, err := objects.bindingRecords()
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	if err := builder.buildDefinitions(); err != nil {
		return semantic.Package{}, Work{}, err
	}
	if err := builder.types.finish(); err != nil {
		return semantic.Package{}, Work{}, err
	}
	types := builder.types.recordsSorted()
	for _, record := range types {
		witness, err := semantic.NewTypeWitness(
			input.id, record.ID(), input.authority,
		)
		if err != nil {
			return semantic.Package{}, Work{}, err
		}
		if err := draft.AddType(record, witness); err != nil {
			return semantic.Package{}, Work{}, err
		}
	}
	for _, declaration := range declarations {
		if err := draft.AddDeclaration(declaration); err != nil {
			return semantic.Package{}, Work{}, err
		}
	}
	for _, binding := range bindings {
		if err := draft.AddBinding(binding); err != nil {
			return semantic.Package{}, Work{}, err
		}
	}
	resolutionCount := draft.ResolutionCount()
	operationCount := draft.OperationCount()
	unsupportedCount := draft.UnsupportedCount()
	pkg, err := draft.SealProducer()
	if err != nil {
		return semantic.Package{}, Work{}, fmt.Errorf(
			"materialize semantic package %s: %w", input.id, err,
		)
	}
	if work.ContextAssignments != len(contexts.byOccurrence) {
		return semantic.Package{}, Work{}, fmt.Errorf(
			"semantic context work=%d, contexts=%d",
			work.ContextAssignments,
			len(contexts.byOccurrence),
		)
	}
	work.OccurrenceResolutions = resolutionCount
	work.TypeConstructions = builder.types.work
	work.ObjectConstructions =
		len(declarations) + len(bindings)
	work.OperationConstructions =
		operationCount + unsupportedCount
	work.CanonicalSortInputs +=
		len(types) + len(declarations) + len(bindings) +
			operationCount + resolutionCount
	return pkg, work, nil
}

func packageDraftCapacity(
	input *packageInput,
	objects *objectIndex,
) semantic.PackageCapacity {
	capacity := semantic.PackageCapacity{
		Definitions: len(input.definitions),
		Resolutions: len(input.order),
		Bindings:    len(objects.bindingIDs),
	}
	for _, occurrence := range input.occurrences {
		if occurrence.domain ==
			catalog.ResolutionDomainExecutable {
			capacity.Operations++
		}
		if occurrence.checkedUnmapped ||
			occurrence.occurrence.Kind().Disposition() !=
				catalog.DispositionActive {
			capacity.Unsupported++
		}
	}
	return capacity
}

func sortedDefinitions(
	definitions map[identity.DefinitionID]structure.ImplementationDefinition,
) []structure.ImplementationDefinition {
	out := make(
		[]structure.ImplementationDefinition, 0, len(definitions),
	)
	for _, definition := range definitions {
		out = append(out, definition)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].ID().String() <
			out[right].ID().String()
	})
	return out
}
