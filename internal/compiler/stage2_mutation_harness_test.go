package compiler

import (
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/frontend"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/semantic"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
	"github.com/tsoniclang/gotots/internal/stagecheck"
)

type stage2MutationHarness struct {
	universe   *source.Universe
	plan       *sourceplan.Plan
	graph      *structure.Graph
	index      *structure.TransientIndex
	facts      *selectionfacts.Artifact
	selections *scope.DefinitionSelections
	executable *executable.Inventory
	packages   []semantic.Package
}

func newStage2MutationHarness(t *testing.T) *stage2MutationHarness {
	t.Helper()
	request := source.Request{
		Dir:              writeStage2Project(t),
		Patterns:         []string{"."},
		ProviderContract: contract.DefaultID,
	}
	selected, err := contract.Resolve(
		request.ProviderContract, "", "",
	)
	if err != nil {
		t.Fatal(err)
	}
	universe, err := source.ResolveUniverse(request)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := sourceplan.Build(
		universe, selected, sourceplan.CertifiedInput{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stagecheck.VerifyProviderSelection(
		universe, plan, nil,
	); err != nil {
		t.Fatal(err)
	}
	hydration, err := source.NewHydrationRequest(
		plan.LocalFileIDs(), plan.LocalSyntheticPackages(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.HydrateUniverse(universe, hydration); err != nil {
		t.Fatal(err)
	}
	graph, index, err := structure.BuildPlanned(
		universe, plan, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := selectionfacts.Materialize(
		universe, graph, index, plan, selected, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	selections, err := scope.SelectDefinitions(
		universe, graph, facts, selected,
	)
	if err != nil {
		t.Fatal(err)
	}
	executableInventory, err := executable.Build(
		graph, index, selections,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stagecheck.VerifyStage1(
		request,
		universe,
		plan,
		graph,
		facts,
		selections,
		executableInventory,
		nil,
	); err != nil {
		t.Fatal(err)
	}
	result, err := frontend.Materialize(
		universe,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		plan,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := stagecheck.VerifyStage2(
		universe,
		plan,
		graph,
		index,
		facts,
		selections,
		executableInventory,
		result.Model(),
		nil,
	); err != nil {
		t.Fatal(err)
	}
	var packages []semantic.Package
	if err := result.Model().VisitPackages(
		func(pkg semantic.Package) error {
			packages = append(packages, pkg)
			return nil
		},
	); err != nil {
		t.Fatal(err)
	}
	return &stage2MutationHarness{
		universe: universe, plan: plan, graph: graph, index: index,
		facts: facts, selections: selections,
		executable: executableInventory, packages: packages,
	}
}

type stage2PackageMutation func(
	*semantic.PackageInput,
) ([]string, error)

func (h *stage2MutationHarness) requireRejected(
	t *testing.T,
	importPath string,
	wantIDs []string,
	mutate stage2PackageMutation,
) {
	t.Helper()
	input, packageIndex := h.packageInput(t, importPath)
	mutationIDs, err := mutate(&input)
	if err == nil {
		var mutated semantic.Package
		mutated, err = semantic.NewPackage(input)
		if err == nil {
			packages := append([]semantic.Package(nil), h.packages...)
			packages[packageIndex] = mutated
			var model *semantic.Model
			model, err = semantic.NewModel(packages)
			if err == nil {
				err = stagecheck.VerifyStage2(
					h.universe,
					h.plan,
					h.graph,
					h.index,
					h.facts,
					h.selections,
					h.executable,
					model,
					nil,
				)
			}
		}
	}
	if err == nil {
		t.Fatal("semantic mutation passed every production boundary")
	}
	for _, want := range append(wantIDs, mutationIDs...) {
		if want != "" && !strings.Contains(err.Error(), want) {
			t.Fatalf(
				"mutation error %q lacks exact identity %q",
				err, want,
			)
		}
	}
}

func (h *stage2MutationHarness) packageInput(
	t *testing.T,
	importPath string,
) (semantic.PackageInput, int) {
	t.Helper()
	for index, pkg := range h.packages {
		if pkg.ID().ImportPath() != importPath {
			continue
		}
		return semantic.PackageInput{
			ID: pkg.ID(), Provenance: pkg.Provenance(),
			Definitions:   pkg.Definitions(),
			Resolutions:   pkg.Resolutions(),
			Declarations:  pkg.Declarations(),
			Bindings:      pkg.Bindings(),
			Types:         pkg.Types(),
			TypeWitnesses: pkg.TypeWitnesses(),
			Operations:    pkg.Operations(),
			Unsupported:   pkg.Unsupported(),
		}, index
	}
	t.Fatalf("semantic package %s is absent", importPath)
	return semantic.PackageInput{}, -1
}

func definitionByName(
	t *testing.T,
	input semantic.PackageInput,
	name string,
) (int, semantic.DefinitionSemantics) {
	t.Helper()
	for index, record := range input.Definitions {
		if record.Spec().Name == name {
			return index, record
		}
	}
	t.Fatalf("semantic definition %s is absent", name)
	return -1, semantic.DefinitionSemantics{}
}

func operationByKind(
	t *testing.T,
	input semantic.PackageInput,
	kind semantic.OperationKind,
) (int, semantic.Operation) {
	t.Helper()
	for index, record := range input.Operations {
		if record.Kind() == kind {
			return index, record
		}
	}
	t.Fatalf("semantic operation %s is absent", kind)
	return -1, semantic.Operation{}
}

func sourceOperationBy(
	t *testing.T,
	input semantic.PackageInput,
	match func(semantic.Operation) bool,
) (int, semantic.Operation) {
	t.Helper()
	for index, record := range input.Operations {
		if record.ID().Source() && match(record) {
			return index, record
		}
	}
	t.Fatal("matching source semantic operation is absent")
	return -1, semantic.Operation{}
}

func resolutionIndex(
	t *testing.T,
	input semantic.PackageInput,
	occurrence identity.OccurrenceID,
) int {
	t.Helper()
	for index, record := range input.Resolutions {
		if record.Occurrence() == occurrence {
			return index
		}
	}
	t.Fatalf("resolution %s is absent", occurrence)
	return -1
}
