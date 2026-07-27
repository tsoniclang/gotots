package constant_test

import (
	"context"
	"errors"
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestContextualConstantExpressionsUseOwningFacts(t *testing.T) {
	loaded := loadConstantSource(t, `package contextual

const Scale = 100

func Direct() int32 { return Scale }
func Double() int { return Scale + Scale }
func F32() float32 { return 0.1 + 0.2 }
func F64() float64 { return 0.1 + 0.2 }
`)
	emission := compileExportedPackage(t, loaded)
	typecheckProgram(t, emission)
	source := onlySourceFile(t, emission, "contextual")

	direct := targetFunction(t, source, "Direct")
	directValue := returnValue(direct)
	directReference, ok := directValue.(tsgo.Identifier)
	if !ok || directReference.Text() != "Scale$int32" {
		t.Fatalf(
			"Direct returns %T, want Scale$int32 projection reference",
			directValue,
		)
	}
	double := returnValue(targetFunction(t, source, "Double"))
	doubleLiteral, ok := double.(tsgo.NumericLiteral)
	if !ok || doubleLiteral.Text() != "200" {
		t.Fatalf("Double returns %T %v, want folded literal 200", double, double)
	}
	float32Value := returnValue(targetFunction(t, source, "F32"))
	float32Literal, ok := float32Value.(tsgo.NumericLiteral)
	if !ok || float32Literal.Text() != "0.30000001192092896" {
		t.Fatalf(
			"F32 returns %T %v, want checker-folded binary32 value",
			float32Value,
			float32Value,
		)
	}
	float64Value := returnValue(targetFunction(t, source, "F64"))
	float64Literal, ok := float64Value.(tsgo.NumericLiteral)
	if !ok || float64Literal.Text() != "0.3" {
		t.Fatalf(
			"F64 returns %T %v, want checker-folded binary64 value",
			float64Value,
			float64Value,
		)
	}
}

func TestLocalConstantProjectionsStayInLexicalBlocks(t *testing.T) {
	loaded := loadConstantSource(t, `package lexical

func Select(flag bool) int {
	result := 0
	if flag {
		const Width = 1
		result = Width
	}
	if !flag {
		const Width = 2
		result = result + Width
	}
	return result
}
`)
	emission := compileExportedPackage(t, loaded)
	typecheckProgram(t, emission)
	source := onlySourceFile(t, emission, "lexical")
	body := targetFunction(t, source, "Select").Body().(tsgo.Block)

	var topLevelProjections int
	var branchProjections int
	for _, statement := range body.Statements() {
		if variable, ok := statement.(tsgo.VariableStatement); ok {
			for _, declaration := range variable.DeclarationList().Declarations() {
				if strings.HasPrefix(
					declaration.Name().(tsgo.Identifier).Text(),
					"Width$",
				) {
					topLevelProjections++
				}
			}
		}
		conditional, ok := statement.(tsgo.IfStatement)
		if !ok {
			continue
		}
		block := conditional.ThenStatement().(tsgo.Block)
		first := block.Statements()[0].(tsgo.VariableStatement)
		name := first.DeclarationList().Declarations()[0].
			Name().(tsgo.Identifier).Text()
		if name != "Width$int" {
			t.Fatalf("branch projection = %q, want Width$int", name)
		}
		branchProjections++
	}
	if topLevelProjections != 0 || branchProjections != 2 {
		t.Fatalf(
			"projection placement = top-level %d, branches %d; want 0 and 2",
			topLevelProjections,
			branchProjections,
		)
	}
}

func TestCrossPackageProjectionImportsUseFullIdentity(t *testing.T) {
	directory := writeConstantModule(t, map[string]string{
		"a/a.go": "package a\n\nconst Width = 1\n",
		"b/b.go": "package b\n\nconst Width = 2\n",
		"api/api.go": `package api

import (
	"example.com/context/a"
	"example.com/context/b"
)

func FromA() int { return a.Width }
func FromB() int { return b.Width }
`,
	})
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   "./api",
	})
	if err != nil {
		t.Fatal(err)
	}
	emission := compileExportedPackage(t, program.Roots()[0])
	typecheckProgram(t, emission)
	printed := printConstantFamily(t, emission)
	for _, expected := range []string{
		"Width$int as Width$int__from_a",
		"Width$int as Width$int__from_b",
		"return Width$int__from_a",
		"return Width$int__from_b",
	} {
		if !strings.Contains(printed, expected) {
			t.Fatalf("cross-package artifact lacks %q:\n%s", expected, printed)
		}
	}
}

func TestUntypedConstantRootIntentIsExplicit(t *testing.T) {
	loaded := loadConstantSource(t, "package roots\n\nconst Width = 64\n")
	selected := loaded.Types().Scope().Lookup("Width").(*types.Const)

	if _, err := emit.NewRoot(selected); err == nil {
		t.Fatal("ambiguous untyped-constant representation root was accepted")
	}

	apiEmission := compileExportedPackage(t, loaded)
	if printed := printConstantFamily(t, apiEmission); strings.Contains(
		printed,
		"Width",
	) {
		t.Fatalf(
			"compile-time-only exported Go constant gained runtime output:\n%s",
			printed,
		)
	}

	root, err := emit.NewConstantProjectionRoot(selected, types.Int32)
	if err != nil {
		t.Fatal(err)
	}
	emission, err := emit.Compile(loaded.Program(), []emit.Root{root})
	if err != nil {
		t.Fatal(err)
	}
	typecheckProgram(t, emission)
	printed := printConstantFamily(t, emission)
	if strings.Count(printed, "export const Width$int32") != 1 {
		t.Fatalf("explicit int32 projection is not singular:\n%s", printed)
	}
	if strings.Contains(printed, "Width$int64") {
		t.Fatalf("explicit int32 root widened to another projection:\n%s", printed)
	}
}

func TestExplicitConstantProjectionRootRejectsUnrepresentableValue(t *testing.T) {
	loaded := loadConstantSource(t, "package roots\n\nconst Width = 300\n")
	selected := loaded.Types().Scope().Lookup("Width").(*types.Const)
	root, err := emit.NewConstantProjectionRoot(selected, types.Uint8)
	if err != nil {
		t.Fatal(err)
	}
	_, err = emit.Compile(loaded.Program(), []emit.Root{root})
	var unsupported *api.UnsupportedError
	if !errors.As(err, &unsupported) ||
		unsupported.Role != api.RolePackageConstantValue {
		t.Fatalf(
			"unrepresentable explicit root error = %#v, want package-constant-value rejection",
			err,
		)
	}
}

func TestMultipleExplicitConstantProjectionRootsExactJoinTheirDeclarations(
	t *testing.T,
) {
	loaded := loadConstantSource(t, "package roots\n\nconst Width = 64\n")
	selected := loaded.Types().Scope().Lookup("Width").(*types.Const)
	var roots []emit.Root
	for _, kind := range []types.BasicKind{types.Int32, types.Uint64} {
		root, err := emit.NewConstantProjectionRoot(selected, kind)
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, root)
	}
	emission, err := emit.Compile(loaded.Program(), roots)
	if err != nil {
		t.Fatal(err)
	}
	typecheckProgram(t, emission)
	printed := printConstantFamily(t, emission)
	for _, declaration := range []string{
		"export const Width$int32",
		"export const Width$uint64",
	} {
		if strings.Count(printed, declaration) != 1 {
			t.Fatalf("%q is not singular:\n%s", declaration, printed)
		}
	}
	if strings.Contains(printed, "Width$int64") {
		t.Fatalf("explicit roots widened to an unrequested projection:\n%s", printed)
	}
}

func TestConstantProjectionRootRejectsInvalidKinds(t *testing.T) {
	loaded := loadConstantSource(t, "package roots\n\nconst Width = 64\n")
	selected := loaded.Types().Scope().Lookup("Width").(*types.Const)
	for _, kind := range []types.BasicKind{
		types.Invalid,
		types.UnsafePointer,
		types.UntypedInt,
		types.BasicKind(-1),
		types.BasicKind(len(types.Typ)),
	} {
		if _, err := emit.NewConstantProjectionRoot(selected, kind); err == nil {
			t.Fatalf("projection root accepted invalid kind %d", kind)
		}
	}
}

func TestConstantContextMutationsFailAtOwners(t *testing.T) {
	t.Run("missing enclosing folded value", func(t *testing.T) {
		loaded := loadConstantSource(t, `package mutation

const Scale = 100
func Double() int { return Scale + Scale }
`)
		function := sourceFunction(t, loaded.Files()[0].Syntax(), "Double")
		binary := function.Body.List[0].(*ast.ReturnStmt).
			Results[0].(*ast.BinaryExpr)
		facts := loaded.TypesInfo().Types[binary]
		facts.Value = nil
		loaded.TypesInfo().Types[binary] = facts
		_, err := compileExportedPackageError(loaded)
		var unsupported *api.UnsupportedError
		if !errors.As(err, &unsupported) ||
			unsupported.Construct != "*ast.BinaryExpr" {
			t.Fatalf(
				"missing folded value error = %#v, want binary owner rejection",
				err,
			)
		}
	})

	t.Run("wrong child semantic type", func(t *testing.T) {
		loaded := loadConstantSource(t, `package mutation

const Scale = 100
func Direct() int32 { return Scale }
`)
		use := findConstUse(t, loaded, "Direct", "Scale")
		facts := loaded.TypesInfo().Types[use]
		facts.Type = types.Typ[types.UntypedString]
		loaded.TypesInfo().Types[use] = facts
		_, err := compileExportedPackageError(loaded)
		var unsupported *api.UnsupportedError
		if !errors.As(err, &unsupported) ||
			unsupported.Construct != "*ast.Ident" {
			t.Fatalf(
				"wrong child type error = %#v, want identifier owner rejection",
				err,
			)
		}
	})
}

func loadConstantSource(t *testing.T, source string) *load.Package {
	t.Helper()
	directory := writeConstantModule(t, map[string]string{"source.go": source})
	loaded, err := load.One(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func writeConstantModule(t *testing.T, files map[string]string) string {
	t.Helper()
	directory := t.TempDir()
	writeFile(
		t,
		filepath.Join(directory, "go.mod"),
		"module example.com/context\n\ngo 1.26.4\n",
	)
	for path, content := range files {
		writeFile(t, filepath.Join(directory, path), content)
	}
	return directory
}

func compileExportedPackage(
	t *testing.T,
	loaded *load.Package,
) emit.ProgramEmission {
	t.Helper()
	emission, err := compileExportedPackageError(loaded)
	if err != nil {
		t.Fatal(err)
	}
	return emission
}

func compileExportedPackageError(
	loaded *load.Package,
) (emit.ProgramEmission, error) {
	roots, err := emit.ExportedAPIRoots(loaded)
	if err != nil {
		return emit.ProgramEmission{}, err
	}
	return emit.Compile(loaded.Program(), roots)
}

func onlySourceFile(
	t *testing.T,
	emission emit.ProgramEmission,
	packageName string,
) tsgo.SourceFile {
	t.Helper()
	var selected tsgo.SourceFile
	for _, file := range emission.Files() {
		if file.Kind() != emit.TargetFileSource ||
			file.PackageName() != packageName {
			continue
		}
		if selected != nil {
			t.Fatalf("package %s has multiple source artifacts", packageName)
		}
		selected = file.SourceFile()
	}
	if selected == nil {
		t.Fatalf("package %s source artifact is absent", packageName)
	}
	return selected
}

func returnValue(function tsgo.FunctionDeclaration) tsgo.Expression {
	return function.Body().(tsgo.Block).Statements()[0].(tsgo.ReturnStatement).Expression()
}

func sourceFunction(
	t *testing.T,
	source *ast.File,
	name string,
) *ast.FuncDecl {
	t.Helper()
	for _, declaration := range source.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	t.Fatalf("source function %s is absent", name)
	return nil
}
