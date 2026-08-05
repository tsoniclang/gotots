package verify

import (
	"errors"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestWaveTenCommandImportsCommentsAndScopesCompile(t *testing.T) {
	program, sourcePackage := loadClosurePackage(t, ".")
	if sourcePackage.Name() != "main" || len(program.Packages()) != 4 {
		t.Fatalf(
			"command universe = package %q with %d packages, want main with 4",
			sourcePackage.Name(),
			len(program.Packages()),
		)
	}
	roots := []emit.Root{
		closureRoot(t, sourcePackage, "Audit"),
		closureRoot(t, sourcePackage, "main"),
	}
	emission, err := emit.Compile(program, roots)
	if err != nil {
		t.Fatal(err)
	}
	var blankImportPackage bool
	for _, file := range emission.Files() {
		if file.PackageName() == "sideeffect" {
			blankImportPackage = true
			break
		}
	}
	if !blankImportPackage {
		t.Fatal("blank-import package initialization was not reached")
	}
	artifacts := materializeClosure(t, emission)
	for _, required := range []string{
		"export function Audit",
		"export class Box",
		"function scoped",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf("command artifact lacks %q", required)
		}
	}
	for _, sourceComment := range []string{
		"declaration comments are source metadata",
		"trailing parameter comment",
		"localVariable keeps",
	} {
		if strings.Contains(artifacts.printed, sourceComment) {
			t.Fatalf("source-only comment leaked into target: %q", sourceComment)
		}
	}
	if artifacts.files < 5 {
		t.Fatalf("command emission has %d files, want command and dependency artifacts", artifacts.files)
	}
	t.Logf(
		"Wave 10 command: files=%d bytes=%d nodes=%d largest=%d",
		artifacts.files,
		artifacts.bytes,
		artifacts.nodes,
		artifacts.largest,
	)
}

func TestWaveTenSupportedBuiltinsCloseSelectedUniverse(t *testing.T) {
	program, sourcePackage := loadClosurePackage(t, "builtins")
	seen := selectedBuiltins(sourcePackage)
	var missing []string
	for _, name := range types.Universe.Names() {
		builtin, ok := types.Universe.Lookup(name).(*types.Builtin)
		if !ok || name == "print" || name == "println" {
			continue
		}
		if seen[builtin] == 0 {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(seen) != 16 || len(missing) != 0 {
		t.Fatalf("supported builtin fixture has %d identities, missing %v", len(seen), missing)
	}
	roots, err := emit.ExportedAPIRoots(sourcePackage)
	if err != nil {
		t.Fatal(err)
	}
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	emission, err := emit.CompileWithOptions(program, roots, options)
	if err != nil {
		t.Fatal(err)
	}
	artifacts := materializeClosure(t, emission)
	t.Logf(
		"Wave 10 builtins: files=%d bytes=%d nodes=%d largest=%d",
		artifacts.files,
		artifacts.bytes,
		artifacts.nodes,
		artifacts.largest,
	)
}

func TestWaveTenImplementationDefinedBuiltinsHaveTypedBoundaries(t *testing.T) {
	for _, testCase := range []struct {
		function string
		builtin  string
	}{
		{function: "Print", builtin: "print"},
		{function: "DeferredPrint", builtin: "print"},
		{function: "Println", builtin: "println"},
	} {
		t.Run(testCase.function, func(t *testing.T) {
			program, sourcePackage := loadClosurePackage(t, "boundary")
			_, err := emit.Compile(
				program,
				[]emit.Root{closureRoot(t, sourcePackage, testCase.function)},
			)
			var boundary *api.BuiltinBoundaryError
			if !errors.As(err, &boundary) ||
				boundary.Builtin != types.Universe.Lookup(testCase.builtin) ||
				boundary.Reason == "" {
				t.Fatalf("%s boundary = %#v", testCase.function, err)
			}
		})
	}
}

func TestWaveTenBuiltinBoundaryUsesCheckerIdentityNotSpelling(t *testing.T) {
	program, sourcePackage := loadClosurePackage(t, "boundary")
	var mutated *ast.Ident
	ast.Inspect(sourcePackage.Files()[0].Syntax(), func(node ast.Node) bool {
		if mutated != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		identifier, ok := call.Fun.(*ast.Ident)
		if ok && identifier.Name == "print" {
			mutated = identifier
			return false
		}
		return true
	})
	if mutated == nil {
		t.Fatal("print call mutation target is absent")
	}
	mutated.Name = "len"
	_, err := emit.Compile(
		program,
		[]emit.Root{closureRoot(t, sourcePackage, "Print")},
	)
	var boundary *api.BuiltinBoundaryError
	if !errors.As(err, &boundary) ||
		boundary.Builtin != types.Universe.Lookup("print") {
		t.Fatalf("spelling mutation boundary = %#v", err)
	}
}

func TestWaveTenBodylessFunctionIsExactExternalObligation(t *testing.T) {
	program, sourcePackage := loadClosurePackage(t, "external")
	selected, ok := sourcePackage.Types().Scope().Lookup("Read").(*types.Func)
	if !ok {
		t.Fatal("external Read function is absent")
	}
	emission, err := emit.Compile(
		program,
		[]emit.Root{closureRoot(t, sourcePackage, "Read")},
	)
	if err != nil {
		t.Fatal(err)
	}
	obligations := emission.ExternalFunctionObligations()
	if len(obligations) != 1 {
		t.Fatalf("external obligations = %d, want 1", len(obligations))
	}
	obligation := obligations[0]
	signature, _ := selected.Type().(*types.Signature)
	position := obligation.Position()
	if obligation.Function() != selected ||
		obligation.Signature() != signature ||
		obligation.Identity() == "" ||
		!position.IsValid() ||
		!obligation.BuildProfile().Valid() {
		t.Fatalf("external obligation = %#v", obligation)
	}
	obligations[0] = emit.ExternalFunctionObligation{}
	if emission.ExternalFunctionObligations()[0].Identity() == "" {
		t.Fatal("external-obligation accessor exposes backing storage")
	}
	artifacts := materializeClosure(t, emission)
	for _, required := range []string{
		"export function Read",
		"buffer: RuntimeSlice<uint8>",
		"int,",
		"$goInterface_",
		"GoPanic.raiseRuntime(\"unresolved external Go function ",
	} {
		if !strings.Contains(artifacts.printed, required) {
			t.Fatalf(
				"external artifact lacks %q:\n%s",
				required,
				artifacts.printed,
			)
		}
	}
	if strings.Contains(artifacts.printed, "declare function Read") {
		t.Fatal("bodyless source function remained an ambient declaration")
	}
	executeExternalClosure(t, emission, artifacts, obligation.Identity())
}

func TestConstructFixturesCoverSelectedASTUniverse(t *testing.T) {
	universe := selectedASTUniverse(t)
	observed := make(map[string][]string)
	root := filepath.Join(repositoryRoot(t), "testdata", "constructs")
	err := filepath.Walk(root, func(
		sourcePath string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(sourcePath) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(
			token.NewFileSet(),
			sourcePath,
			nil,
			parser.ParseComments,
		)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, sourcePath)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				return true
			}
			target := reflect.TypeOf(node)
			if target.Kind() == reflect.Pointer {
				target = target.Elem()
			}
			name := target.Name()
			if _, selected := universe[name]; selected {
				observed[name] = append(observed[name], filepath.ToSlash(relative))
			}
			if comment, ok := node.(*ast.Comment); ok {
				if _, valid := ast.ParseDirective(comment.Slash, comment.Text); valid {
					observed["Directive"] = append(
						observed["Directive"],
						filepath.ToSlash(relative),
					)
				}
			}
			return true
		})
		for _, group := range file.Comments {
			for _, comment := range group.List {
				if _, valid := ast.ParseDirective(comment.Slash, comment.Text); valid {
					observed["Directive"] = append(
						observed["Directive"],
						filepath.ToSlash(relative),
					)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	packages, err := parser.ParseDir(
		token.NewFileSet(),
		closureDirectory("."),
		nil,
		parser.ParseComments,
	)
	if err != nil {
		t.Fatal(err)
	}
	for name := range packages {
		observed["Package"] = append(observed["Package"], name)
	}
	var missing []string
	for name := range universe {
		if name == "BadDecl" || name == "BadExpr" || name == "BadStmt" {
			continue
		}
		if len(observed[name]) == 0 {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("selected AST forms lack valid construct fixtures: %v", missing)
	}
}

func TestConstructFixturesCoverEverySemanticOperator(t *testing.T) {
	expected := semanticOperatorTokens()
	observed := make(map[token.Token]int)
	root := filepath.Join(repositoryRoot(t), "testdata", "constructs")
	err := filepath.Walk(root, func(
		sourcePath string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(sourcePath) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.AssignStmt:
				observed[node.Tok]++
			case *ast.BinaryExpr:
				observed[node.Op]++
			case *ast.ChanType, *ast.SendStmt:
				observed[token.ARROW]++
			case *ast.Ellipsis:
				observed[token.ELLIPSIS]++
			case *ast.IncDecStmt:
				observed[node.Tok]++
			case *ast.RangeStmt:
				if node.Tok.IsOperator() {
					observed[node.Tok]++
				}
			case *ast.StarExpr:
				observed[token.MUL]++
			case *ast.TypeSpec:
				if node.Assign.IsValid() {
					observed[token.ASSIGN]++
				}
			case *ast.UnaryExpr:
				observed[node.Op]++
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for selected := range expected {
		if observed[selected] == 0 {
			missing = append(missing, selected.String())
		}
	}
	sort.Strings(missing)
	if len(missing) != 0 {
		t.Fatalf("semantic operators lack valid construct fixtures: %v", missing)
	}
	for _, selected := range selectedToolchainTokenValues(t) {
		if tokenDisposition(selected, expected) == "" {
			t.Fatalf("selected token %d (%s) has no closed disposition", selected, selected)
		}
	}
}

func semanticOperatorTokens() map[token.Token]struct{} {
	return map[token.Token]struct{}{
		token.ADD: {}, token.SUB: {}, token.MUL: {}, token.QUO: {},
		token.REM: {}, token.AND: {}, token.OR: {}, token.XOR: {},
		token.SHL: {}, token.SHR: {}, token.AND_NOT: {},
		token.ADD_ASSIGN: {}, token.SUB_ASSIGN: {}, token.MUL_ASSIGN: {},
		token.QUO_ASSIGN: {}, token.REM_ASSIGN: {}, token.AND_ASSIGN: {},
		token.OR_ASSIGN: {}, token.XOR_ASSIGN: {}, token.SHL_ASSIGN: {},
		token.SHR_ASSIGN: {}, token.AND_NOT_ASSIGN: {},
		token.LAND: {}, token.LOR: {}, token.ARROW: {}, token.INC: {},
		token.DEC: {}, token.EQL: {}, token.LSS: {}, token.GTR: {},
		token.ASSIGN: {}, token.NOT: {}, token.NEQ: {}, token.LEQ: {},
		token.GEQ: {}, token.DEFINE: {}, token.ELLIPSIS: {}, token.TILDE: {},
	}
}

func selectedToolchainTokenValues(t *testing.T) []token.Token {
	t.Helper()
	sourcePackage, err := importer.Default().Import("go/token")
	if err != nil {
		t.Fatal(err)
	}
	tokenObject := sourcePackage.Scope().Lookup("Token")
	var result []token.Token
	for _, name := range sourcePackage.Scope().Names() {
		object, ok := sourcePackage.Scope().Lookup(name).(*types.Const)
		if !ok || !types.Identical(object.Type(), tokenObject.Type()) {
			continue
		}
		value, exact := constant.Int64Val(object.Val())
		if !exact {
			t.Fatalf("go/token.%s has non-integral value %s", name, object.Val())
		}
		result = append(result, token.Token(value))
	}
	return result
}

func tokenDisposition(
	selected token.Token,
	semantic map[token.Token]struct{},
) string {
	if _, ok := semantic[selected]; ok {
		return "semantic"
	}
	if selected.IsLiteral() {
		return "literal"
	}
	if selected.IsKeyword() {
		return "parent-owned"
	}
	switch selected {
	case token.ILLEGAL:
		return "recovery"
	case token.EOF, token.COMMENT,
		token.LPAREN, token.LBRACK, token.LBRACE,
		token.COMMA, token.PERIOD, token.SEMICOLON, token.COLON,
		token.RPAREN, token.RBRACK, token.RBRACE:
		return "parent-owned"
	default:
		return ""
	}
}

func selectedBuiltins(sourcePackage *load.Package) map[*types.Builtin]int {
	result := make(map[*types.Builtin]int)
	for _, object := range sourcePackage.TypesInfo().Uses {
		if builtin, ok := object.(*types.Builtin); ok {
			result[builtin]++
		}
	}
	return result
}
