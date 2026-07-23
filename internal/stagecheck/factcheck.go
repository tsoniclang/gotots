package stagecheck

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/language/selectionfacts"
	"github.com/tsoniclang/gotots/internal/language/structure"
	"github.com/tsoniclang/gotots/internal/scope/contract"
	"github.com/tsoniclang/gotots/internal/scope/sourceplan"
	"github.com/tsoniclang/gotots/internal/source"
)

func verifySelectionFactValues(
	universe *source.Universe,
	plan *sourceplan.Plan,
	graph *structure.Graph,
	facts *selectionfacts.Artifact,
) error {
	packages := map[identity.PackageID]*source.LoadedPackage{}
	definitionPackage := map[identity.DefinitionID]identity.PackageID{}
	cgoFiles := map[identity.FileID]bool{}
	for _, pkg := range universe.Packages() {
		packages[pkg.ID()] = pkg
		for _, file := range pkg.Files() {
			cgoFiles[file.ID()] = file.CgoOriginal()
		}
	}
	for _, pkg := range graph.Packages() {
		for _, definition := range pkg.Definitions() {
			definitionPackage[definition.ID()] = pkg.ID()
		}
	}
	cgoData := map[identity.PackageID]*independentCgoFacts{}
	evidence, err := deriveIndependentFactEvidence(universe, graph)
	if err != nil {
		return err
	}
	problems := newProblemSet()
	for _, fact := range facts.Facts() {
		if fact.ID().Kind() != contract.SelectionFactCDependent {
			problems.add(
				"no independent fact verifier for " + fact.ID().String(),
			)
			continue
		}
		definition := fact.ID().Definition()
		pkgID, present := definitionPackage[definition]
		if !present {
			problems.add(
				"fact names unknown definition " + definition.String(),
			)
			continue
		}
		if !definition.File().IsZero() {
			if plan != nil {
				decision, planned := plan.For(definition.File())
				if !planned {
					problems.add(
						"fact definition lacks source decision " +
							definition.String(),
					)
					continue
				}
				if decision.Kind() == sourceplan.KindCertifiedGraph {
					continue
				}
			}
		} else if definition.SyntheticRole().Valid() {
			if plan != nil {
				decision, planned := plan.SyntheticFor(pkgID)
				if planned &&
					decision.Kind() == sourceplan.KindCertifiedGraph {
					continue
				}
			}
		}
		expected := false
		switch {
		case definition.SyntheticRole().Valid():
			expected = true
		case definition.ImplicitOp().Valid():
			expected = false
		default:
			pkg := packages[pkgID]
			if pkg == nil {
				problems.add("fact package is absent " + pkgID.String())
				continue
			}
			if cgoFiles[definition.File()] {
				data := cgoData[pkgID]
				if data == nil {
					var err error
					data, err = deriveIndependentCgoFacts(
						universe, pkg, plan,
					)
					if err != nil {
						problems.add(err.Error())
						continue
					}
					cgoData[pkgID] = data
				}
				node := data.checked[definition]
				if node == nil {
					problems.add(
						"local cgo fact lacks checked definition " +
							definition.String(),
					)
					continue
				}
				expected = independentUsesSynthetic(
					node, pkg.CheckerView(), data.syntheticObjects,
				)
			}
		}
		if fact.Value() != expected {
			problems.addf(
				"%s value=%t independent=%t",
				fact.ID(), fact.Value(), expected,
			)
		}
		expectedProducer := independentFactDigest(
			fmt.Sprintf(
				"selectionfacts/v%d", selectionfacts.SchemaVersion,
			),
			fact.ID().Kind().String(),
			universe.Toolchain().BinaryDigest(),
			universe.Toolchain().Version(),
			universe.Toolchain().BuildConfigurationDigest(),
			catalog.StructureDigest(),
		)
		expectedEvidence, evidenceErr := evidence.digest(
			definition, fact.ID().Kind(), expected,
		)
		if evidenceErr != nil {
			problems.add(evidenceErr.Error())
			continue
		}
		if fact.ProducerDigest() != expectedProducer ||
			fact.EvidenceDigest() != expectedEvidence {
			problems.add(
				"fact digest mismatch " + fact.ID().String(),
			)
		}
	}
	return problems.verificationError(
		"selection-fact-value",
		"independent fact comparison failed",
	)
}

func independentFactDigest(parts ...string) string {
	hash := sha256.New()
	for _, part := range parts {
		fmt.Fprintf(hash, "%d:%s|", len(part), part)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}

type independentCgoFacts struct {
	checked          map[identity.DefinitionID]ast.Node
	syntheticObjects map[types.Object]bool
}

func deriveIndependentCgoFacts(
	universe *source.Universe,
	pkg *source.LoadedPackage,
	plan *sourceplan.Plan,
) (*independentCgoFacts, error) {
	out := &independentCgoFacts{
		checked:          map[identity.DefinitionID]ast.Node{},
		syntheticObjects: map[types.Object]bool{},
	}
	allOriginal := map[string]bool{}
	localOriginal := map[string]bool{}
	origins := map[independentOrigin][]identity.DefinitionID{}
	for _, file := range pkg.Files() {
		if !file.CgoOriginal() {
			continue
		}
		allOriginal[file.Path()] = true
		if plan != nil {
			decision, present := plan.For(file.ID())
			if !present || decision.Kind() != sourceplan.KindLocalSyntax {
				continue
			}
		}
		localOriginal[file.Path()] = true
		derived, err := deriveFile(file)
		if err != nil {
			return nil, err
		}
		for definition, node := range derived.definitions {
			join, err := independentOriginNode(definition.Kind(), node)
			if err != nil {
				return nil, err
			}
			position := derived.fset.PositionFor(join.Pos(), false)
			key := independentOrigin{
				file:   file.Path(),
				line:   position.Line,
				column: position.Column,
				kind:   definition.Kind(),
			}
			origins[key] = append(origins[key], definition)
		}
	}
	view := pkg.CheckerView()
	for _, declaration := range pkg.CheckedDeclarations() {
		display := universe.Fset().Position(declaration.Pos())
		if !allOriginal[display.Filename] {
			ast.Inspect(declaration, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if !ok || view == nil {
					return true
				}
				if object, present := view.DefOf(identifier); present &&
					object != nil {
					out.syntheticObjects[object] = true
				}
				return true
			})
			continue
		}
		if !localOriginal[display.Filename] {
			continue
		}
		packageContext, contextErr := catalog.NewDefinitionContext(
			catalog.DefinitionScopePackage, catalog.TokenInvalid,
		)
		if contextErr != nil {
			return nil, contextErr
		}
		candidates, err := independentCheckedDefinitions(
			declaration, packageContext,
		)
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			position := universe.Fset().Position(candidate.join.Pos())
			definition, _, err := independentResolveOrigin(
				origins,
				independentOrigin{
					file:   position.Filename,
					line:   position.Line,
					column: position.Column,
					kind:   candidate.kind,
				},
			)
			if err != nil {
				return nil, err
			}
			if out.checked[definition] != nil {
				return nil, fmt.Errorf(
					"independent cgo fact mapping duplicates %s",
					definition,
				)
			}
			out.checked[definition] = candidate.root
		}
	}
	return out, nil
}

func independentUsesSynthetic(
	root ast.Node,
	view *source.TypeInfoView,
	synthetic map[types.Object]bool,
) bool {
	if root == nil || view == nil || len(synthetic) == 0 {
		return false
	}
	found := false
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil || found {
			return false
		}
		if literal, ok := node.(*ast.FuncLit); ok &&
			ast.Node(literal) != root {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if object, present := view.UseOf(identifier); present &&
			synthetic[object] {
			found = true
			return false
		}
		return true
	})
	return found
}
