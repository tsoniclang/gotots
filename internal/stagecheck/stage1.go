package stagecheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/executable"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerifyStage1 blocks source finalization until every Stage-1 artifact has
// passed independent structural, selection, and executable exact joins.
func VerifyStage1(
	req source.Request,
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	certified *structure.ProviderArtifact,
) error {
	return verifyStage1(
		req,
		universe,
		plan,
		graph,
		facts,
		selections,
		executableInventory,
		certified,
		nil,
	)
}

// VerifyExtractedStage1 independently certifies the all-local graph used only
// to produce or audit provider artifacts. It accepts no structural-source
// plan, so extraction authority cannot masquerade as an ordinary compilation
// decision.
func VerifyExtractedStage1(
	req source.Request,
	universe *source.Universe,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
) error {
	return verifyStage1(
		req,
		universe,
		nil,
		graph,
		facts,
		selections,
		executableInventory,
		nil,
		nil,
	)
}

// VerifyExtractedPackageStage1 independently certifies one bounded all-local
// provider package without widening hydration or graph derivation to the
// surrounding metadata closure.
func VerifyExtractedPackageStage1(
	req source.Request,
	universe *source.Universe,
	packageID identity.PackageID,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
) error {
	if packageID.IsZero() {
		return &VerificationError{
			Stage:  "provider-package",
			Reason: "provider package identity is zero",
		}
	}
	return verifyStage1(
		req,
		universe,
		nil,
		graph,
		facts,
		selections,
		executableInventory,
		nil,
		map[identity.PackageID]bool{packageID: true},
	)
}

func verifyStage1(
	req source.Request,
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
	certified *structure.ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
) error {
	selected, err := contract.Resolve(
		req.ProviderContract,
		req.ProviderContractDigest,
		req.ProviderContractArtifact,
	)
	if err != nil {
		return err
	}
	if plan != nil {
		if err := verifySourcePlan(
			req, universe, plan, selected, certified,
		); err != nil {
			return err
		}
	}
	var hydrationErr error
	if selectedPackages == nil {
		hydrationErr = verifyHydration(universe, plan)
	} else {
		hydrationErr = verifyHydrationPackages(
			universe, selectedPackages,
		)
	}
	if hydrationErr != nil {
		return hydrationErr
	}
	if err := verifyDefinitionGraphPackages(
		universe, plan, graph, certified, selectedPackages,
	); err != nil {
		return err
	}
	if err := verifyCertifiedSelectionFacts(
		plan, graph, selected, certified,
	); err != nil {
		return err
	}
	if err := verifySelections(
		universe,
		graph,
		facts,
		selections,
		selected,
		selectedPackages,
	); err != nil {
		return err
	}
	if err := verifySelectionFactValues(
		universe, plan, graph, facts,
	); err != nil {
		return err
	}
	if err := verifyExecutableRegions(
		universe,
		plan,
		graph,
		selections,
		executableInventory,
	); err != nil {
		return err
	}
	return nil
}

func verifyDefinitionGraphPackages(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	certified *structure.ProviderArtifact,
	selectedPackages map[identity.PackageID]bool,
) error {
	expectedPackages := map[identity.PackageID]bool{}
	for _, pkg := range universe.Packages() {
		if selectedPackages != nil && !selectedPackages[pkg.ID()] {
			continue
		}
		if pkg.Disposition() == source.DispositionOrdinarySource ||
			pkg.Disposition() == source.DispositionUnsafeIntrinsic {
			expectedPackages[pkg.ID()] = true
		}
	}
	actualPackages := map[identity.PackageID]bool{}
	censusByPackage := map[identity.PackageID][]structure.DefinitionCensusRecord{}
	for _, record := range graph.DefinitionCensus() {
		censusByPackage[record.Package()] = append(
			censusByPackage[record.Package()],
			record,
		)
	}
	problems := newProblemSet()
	if err := graph.VisitPackages(func(
		pkg structure.PackageGraph,
	) error {
		if actualPackages[pkg.ID()] {
			problems.add("duplicate package|" + pkg.ID().String())
		}
		actualPackages[pkg.ID()] = true
		if !expectedPackages[pkg.ID()] {
			problems.add("unexpected package|" + pkg.ID().String())
		}
		if err := compareDefinitionCensus(
			pkg, censusByPackage[pkg.ID()],
		); err != nil {
			return err
		}
		delete(censusByPackage, pkg.ID())
		expected, err := deriveExpectedGraph(
			universe,
			plan,
			certified,
			map[identity.PackageID]bool{pkg.ID(): true},
		)
		if err != nil {
			return &VerificationError{
				Stage:  "definition-graph-independent",
				Reason: pkg.ID().String() + ": " + err.Error(),
			}
		}
		return compareLedgers(
			"definition-graph/"+pkg.ID().String(),
			ledgerForPackage(pkg),
			expected,
		)
	}); err != nil {
		return err
	}
	for packageID := range expectedPackages {
		if !actualPackages[packageID] {
			problems.add("missing package|" + packageID.String())
		}
	}
	for packageID := range censusByPackage {
		problems.add(
			"indexed package without graph|" + packageID.String(),
		)
	}
	if !problems.empty() {
		return problems.verificationError(
			"definition-graph",
			"package-set exact join failed",
		)
	}
	return nil
}

// VerifyFinalizedStage1 proves the final artifact graph has no reachable raw
// AST, token.FileSet, types.Info, types.Package, or mutable checker facade.
func VerifyFinalizedStage1(
	universe *source.Universe,
	workspace *source.Workspace,
	graph *structure.Graph,
	selections *scope.DefinitionSelections,
	executableInventory *executable.Inventory,
) error {
	if universe == nil ||
		workspace == nil ||
		graph == nil ||
		selections == nil ||
		executableInventory == nil {
		return &VerificationError{
			Stage:  "stage1-finalization",
			Reason: "one finalized Stage-1 artifact is nil",
		}
	}
	if !universe.Finalized() || universe.Hydrated() ||
		universe.Fset() != nil {
		return &VerificationError{
			Stage:  "stage1-finalization",
			Reason: "transient universe was not actively severed",
		}
	}
	for _, pkg := range universe.Packages() {
		if pkg.Types() != nil ||
			pkg.CheckerView() != nil ||
			len(pkg.CheckedDeclarations()) != 0 {
			return &VerificationError{
				Stage: "stage1-finalization",
				Reason: "transient checker remains reachable from " +
					pkg.ID().String(),
			}
		}
		for _, file := range pkg.Files() {
			if file.PhysicalSyntax() != nil ||
				file.PhysicalFileSet() != nil ||
				file.CheckedSyntax() != nil ||
				len(file.SelectedBytes()) != 0 {
				return &VerificationError{
					Stage: "stage1-finalization",
					Reason: "transient syntax remains reachable from " +
						file.ID().String(),
				}
			}
		}
	}
	for _, value := range []any{
		workspace, graph, selections, executableInventory,
	} {
		if path := rawFacadePath(
			reflect.TypeOf(value), map[reflect.Type]bool{},
		); path != "" {
			return &VerificationError{
				Stage:  "stage1-finalization",
				Reason: "raw transient facade remains reachable at " + path,
			}
		}
	}
	return nil
}

func rawFacadePath(
	typ reflect.Type,
	seen map[reflect.Type]bool,
) string {
	if typ == nil || seen[typ] {
		return ""
	}
	seen[typ] = true
	astNode := reflect.TypeOf((*ast.Node)(nil)).Elem()
	typeNode := reflect.TypeOf((*types.Type)(nil)).Elem()
	objectNode := reflect.TypeOf((*types.Object)(nil)).Elem()
	if typ == astNode ||
		typ.Implements(astNode) ||
		typ == typeNode ||
		typ.Implements(typeNode) ||
		typ == objectNode ||
		typ.Implements(objectNode) ||
		typ == reflect.TypeOf((*token.FileSet)(nil)) ||
		typ == reflect.TypeOf((*token.File)(nil)) ||
		typ == reflect.TypeOf((*types.Info)(nil)) ||
		typ == reflect.TypeOf((*types.Package)(nil)) ||
		typ == reflect.TypeOf((*types.Scope)(nil)) ||
		typ == reflect.TypeOf((*types.Selection)(nil)) ||
		typ == reflect.TypeOf(types.Instance{}) {
		return typ.String()
	}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return rawFacadePath(typ.Elem(), seen)
	case reflect.Map:
		if path := rawFacadePath(typ.Key(), seen); path != "" {
			return path
		}
		return rawFacadePath(typ.Elem(), seen)
	case reflect.Struct:
		for index := 0; index < typ.NumField(); index++ {
			field := typ.Field(index)
			if path := rawFacadePath(field.Type, seen); path != "" {
				return typ.String() + "." + field.Name + "->" + path
			}
		}
	}
	return ""
}
