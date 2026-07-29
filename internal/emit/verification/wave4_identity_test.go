package emit_test

import (
	"context"
	"errors"
	"go/ast"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestWaveFourLabelsUseCheckerIdentityNotSourceSpelling(t *testing.T) {
	program, sourcePackage := loadWaveFourProgram(t)
	function := waveFourFunction(t, sourcePackage, "integerRange")
	labeled, branches := waveFourLabeledRange(t, function)
	labeled.Label.Name = "forgedDeclarationSpelling"
	for _, branch := range branches {
		branch.Label.Name = "forgedUseSpelling"
	}

	emission := compileWaveFourProgram(t, program, sourcePackage)
	target := waveFourTargetFunction(t, emission, "integerRange")
	body := target.Body().(tsgo.Block).Statements()
	var targetLabel tsgo.LabeledStatement
	for _, statement := range body {
		if candidate, ok := statement.(tsgo.LabeledStatement); ok {
			targetLabel = candidate
		}
	}
	if targetLabel == nil || targetLabel.Label().Text() != "outer" {
		t.Fatalf("target label is absent or not exact checker-owned outer")
	}
	loop := targetLabel.Statement().(tsgo.ForStatement)
	loopBody := loop.Statement().(tsgo.Block).Statements()
	continueStatement := loopBody[2].(tsgo.IfStatement).
		ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.ContinueStatement)
	breakStatement := loopBody[4].(tsgo.IfStatement).
		ThenStatement().(tsgo.Block).
		Statements()[0].(tsgo.BreakStatement)
	if continueStatement.Label().Text() != "outer" ||
		breakStatement.Label().Text() != "outer" {
		t.Fatalf(
			"branch labels = %q/%q, want outer",
			continueStatement.Label().Text(),
			breakStatement.Label().Text(),
		)
	}
}

func TestWaveFourMissingLabelUseEvidenceFailsAtBranchOwner(t *testing.T) {
	program, sourcePackage := loadWaveFourProgram(t)
	function := waveFourFunction(t, sourcePackage, "integerRange")
	_, branches := waveFourLabeledRange(t, function)
	delete(sourcePackage.TypesInfo().Uses, branches[0].Label)

	_, err := emit.Compile(
		program,
		[]emit.Root{mustWaveFourRoot(t, sourcePackage, "integerRange")},
	)
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Construct != "*ast.BranchStmt" ||
		unsupported.Category != api.CategoryStatement {
		t.Fatalf("error = %#v, want branch UnsupportedError", err)
	}
}

func loadWaveFourProgram(t *testing.T) (*load.Program, *load.Package) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory: waveFourStatementDirectory(),
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.Roots()[0]
}

func compileWaveFourProgram(
	t *testing.T,
	program *load.Program,
	sourcePackage *load.Package,
) emit.ProgramEmission {
	t.Helper()
	roots, err := emit.ExportedAPIRoots(sourcePackage)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func mustWaveFourRoot(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) emit.Root {
	t.Helper()
	object := sourcePackage.Types().Scope().Lookup(name)
	root, err := emit.NewRoot(object)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func waveFourFunction(
	t *testing.T,
	sourcePackage *load.Package,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, file := range sourcePackage.Files() {
		for _, declaration := range file.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if ok && function.Name.Name == name {
				return function
			}
		}
	}
	t.Fatalf("Go function %s is absent", name)
	return nil
}

func waveFourLabeledRange(
	t *testing.T,
	function *ast.FuncDecl,
) (*ast.LabeledStmt, []*ast.BranchStmt) {
	t.Helper()
	var labeled *ast.LabeledStmt
	var branches []*ast.BranchStmt
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.LabeledStmt:
			labeled = node
		case *ast.BranchStmt:
			if node.Label != nil {
				branches = append(branches, node)
			}
		}
		return true
	})
	if labeled == nil || len(branches) != 2 {
		t.Fatalf("labeled range = %v with %d branches", labeled != nil, len(branches))
	}
	return labeled, branches
}

func waveFourTargetFunction(
	t *testing.T,
	emission emit.ProgramEmission,
	name string,
) tsgo.FunctionDeclaration {
	t.Helper()
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource {
			continue
		}
		for _, statement := range file.SourceFile().Statements() {
			function, ok := statement.(tsgo.FunctionDeclaration)
			if ok && function.Name().Text() == name {
				return function
			}
		}
	}
	t.Fatalf("target function %s is absent", name)
	return nil
}
