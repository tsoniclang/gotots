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

type localProjectionSelection struct {
	files        []identity.FileID
	declarations []identity.SemanticDeclarationID
	synthetic    bool
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
	writer, err := semantic.NewCheckerStoreWriter()
	if err != nil {
		return nil, err
	}
	sealed := false
	defer func() {
		if !sealed {
			writer.Abort()
		}
	}()
	local := map[identity.PackageID]localProjectionSelection{}
	work := Work{}
	_, err = stage.visitPackageInputs(func(input *packageInput) error {
		pkg, declarations, packageWork, err :=
			materializePackage(stage, input)
		if err != nil {
			return fmt.Errorf(
				"materialize semantic package %s: %w",
				input.id, err,
			)
		}
		files, err := localPackageFiles(
			stage.plan, input.loaded,
		)
		if err != nil {
			return err
		}
		if err := writer.Append(pkg); err != nil {
			return err
		}
		selection := localProjectionSelection{
			files: files, declarations: declarations,
		}
		if synthetic, present := stage.plan.SyntheticFor(input.id); present {
			selection.synthetic =
				synthetic.Kind() == sourceplan.KindLocalSyntax
		}
		local[input.id] = selection
		work.merge(packageWork)
		return nil
	})
	if err != nil {
		return nil, err
	}
	checker, metrics, err := writer.Seal()
	if err != nil {
		return nil, err
	}
	sealed = true
	projections, err := buildPackageProjections(
		stage, local,
	)
	if err != nil {
		_ = checker.Close()
		return nil, err
	}
	model, err := semantic.NewProjectedModel(
		projections, checker, provider,
	)
	if err != nil {
		_ = checker.Close()
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
	var semanticPackage semantic.Package
	var work Work
	count, err := stage.visitPackageInputs(func(input *packageInput) error {
		if !semanticPackage.ID().IsZero() {
			return fmt.Errorf(
				"semantic provider derivation has more than one package",
			)
		}
		var materializeErr error
		semanticPackage, _, work, materializeErr =
			materializePackage(stage, input)
		return materializeErr
	})
	if err != nil {
		return semantic.Package{}, Work{}, err
	}
	if count != 1 {
		return semantic.Package{}, Work{}, fmt.Errorf(
			"semantic provider derivation has %d packages",
			count,
		)
	}
	return semanticPackage, work, nil
}

func buildPackageProjections(
	stage *stageInput,
	local map[identity.PackageID]localProjectionSelection,
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
		return packageIDs[left].Compare(packageIDs[right]) < 0
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
		if selection, present := local[packageID]; present {
			input.Local = true
			input.LocalFiles = selection.files
			input.LocalDeclarations = selection.declarations
			input.LocalSynthetic = selection.synthetic
		}
		out = append(out, input)
	}
	return out, nil
}

func localPackageFiles(
	plan *sourceplan.Plan,
	loaded *source.LoadedPackage,
) (
	[]identity.FileID,
	error,
) {
	var files []identity.FileID
	for _, file := range loaded.Files() {
		decision, present := plan.For(file.ID())
		if !present {
			return nil, fmt.Errorf(
				"semantic source plan omits %s", file.ID(),
			)
		}
		if decision.Kind() == sourceplan.KindLocalSyntax {
			files = append(files, file.ID())
		}
	}
	return files, nil
}

type packageBuilder struct {
	stage                 *stageInput
	input                 *packageInput
	contexts              *contextIndex
	objects               *objectIndex
	types                 *typeBuilder
	draft                 *semantic.PackageDraft
	operationByOccurrence []identity.OperationID
	variantByOccurrence   []catalog.Variant
	resolvedOccurrences   []bool
	definitionByRoot      map[packageOccurrenceRef]identity.DefinitionID
}

func materializePackage(
	stage *stageInput,
	input *packageInput,
) (
	semantic.Package,
	[]identity.SemanticDeclarationID,
	Work,
	error,
) {
	work := input.work
	work.Packages = 1
	contexts, err := buildContexts(input, &work)
	if err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	objects, err := buildObjectIndex(
		stage, input, contexts, &work,
	)
	if err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	draft, err := semantic.NewPackageDraft(
		input.id,
		input.provenance,
		packageDraftCapacity(input, objects),
	)
	if err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	builder := &packageBuilder{
		stage: stage, input: input, contexts: contexts, objects: objects,
		types: objects.typeBuilder,
		draft: draft,
		operationByOccurrence: make(
			[]identity.OperationID,
			input.occurrences.referenceCount()+1,
		),
		variantByOccurrence: make(
			[]catalog.Variant,
			input.occurrences.referenceCount()+1,
		),
		resolvedOccurrences: make(
			[]bool,
			input.occurrences.referenceCount()+1,
		),
		definitionByRoot: map[packageOccurrenceRef]identity.DefinitionID{},
	}
	if err := input.definitions.visit(func(
		_ packageDefinitionRef,
		record *definitionInput,
	) error {
		definition := record.definition.ID()
		if !definition.Root().IsZero() {
			reference := input.occurrenceReference(definition.Root())
			if reference.valid() {
				builder.definitionByRoot[reference] = definition
			}
		}
		return nil
	}); err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	if err := builder.resolveOccurrences(); err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	if err := builder.buildDefinitions(); err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	bindingCount, err := objects.visitBindingRecords(
		draft.AddBinding,
	)
	if err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	declarations, typeCount, err :=
		builder.materializeSemanticClosure()
	if err != nil {
		return semantic.Package{}, nil, Work{}, err
	}
	declarationCount := len(declarations)
	resolutionCount := draft.ResolutionCount()
	operationCount := draft.OperationCount()
	unsupportedCount := draft.UnsupportedCount()
	pkg, err := draft.SealProducer()
	if err != nil {
		return semantic.Package{}, nil, Work{}, fmt.Errorf(
			"materialize semantic package %s: %w", input.id, err,
		)
	}
	if work.ContextAssignments != contexts.count {
		return semantic.Package{}, nil, Work{}, fmt.Errorf(
			"semantic context work=%d, contexts=%d",
			work.ContextAssignments,
			contexts.count,
		)
	}
	work.OccurrenceResolutions = resolutionCount
	work.TypeConstructions = builder.types.work
	work.ObjectConstructions =
		declarationCount + bindingCount
	work.OperationConstructions =
		operationCount + unsupportedCount
	work.CanonicalSortInputs +=
		typeCount + declarationCount + bindingCount +
			operationCount + resolutionCount
	return pkg, declarations, work, nil
}

func packageDraftCapacity(
	input *packageInput,
	objects *objectIndex,
) semantic.PackageCapacity {
	capacity := semantic.PackageCapacity{
		Definitions: input.definitions.count(),
		Resolutions: len(input.order),
		Bindings:    len(objects.bindingIDs),
	}
	_ = input.occurrences.visit(func(
		_ packageOccurrenceRef,
		occurrence *occurrenceInput,
	) error {
		if occurrence.domain ==
			catalog.ResolutionDomainExecutable {
			capacity.Operations++
		}
		if occurrence.checkedUnmapped ||
			occurrence.occurrence.Kind().Disposition() !=
				catalog.DispositionActive {
			capacity.Unsupported++
		}
		return nil
	})
	return capacity
}

func sortedDefinitions(
	definitions *definitionStore,
) []structure.ImplementationDefinition {
	out := make(
		[]structure.ImplementationDefinition, 0, definitions.count(),
	)
	_ = definitions.visit(func(
		_ packageDefinitionRef,
		definition *definitionInput,
	) error {
		out = append(out, definition.definition)
		return nil
	})
	sort.Slice(out, func(left, right int) bool {
		return out[left].ID().Compare(out[right].ID()) < 0
	})
	return out
}
