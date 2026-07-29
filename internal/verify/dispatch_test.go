package verify

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

type nodeCategory string

const (
	categoryDeclaration nodeCategory = "declaration"
	categoryExpression  nodeCategory = "expression"
	categoryParentOwned nodeCategory = "parent-owned"
	categoryStatement   nodeCategory = "statement"
)

func TestSelectedToolchainASTUniverseHasTotalTypedDispatch(t *testing.T) {
	universe := selectedASTUniverse(t)
	counts := make(map[nodeCategory]int)
	var astContract []string
	for name, form := range universe {
		counts[form.category]++
		astContract = append(astContract, string(form.category)+":"+name+":"+form.shape)
	}
	wantCounts := map[nodeCategory]int{
		categoryDeclaration: 3,
		categoryExpression:  15,
		categoryParentOwned: 20,
		categoryStatement:   19,
	}
	for category, expected := range wantCounts {
		if counts[category] != expected {
			t.Errorf("%s forms = %d, replace pinned expected %d", category, counts[category], expected)
		}
	}
	tokenContract := selectedTokenContract(t)
	universeContract, builtinCount := selectedUniverseContract()
	if len(tokenContract) != 82 || len(universeContract) != 44 || builtinCount != 18 {
		t.Errorf(
			"replace pinned selected-toolchain counts: tokens=%d universe=%d builtins=%d",
			len(tokenContract),
			len(universeContract),
			builtinCount,
		)
	}
	for name, contract := range map[string]struct {
		actual   []string
		expected string
	}{
		"go/ast": {
			actual:   astContract,
			expected: "3bc8fb3e649a7c8f059ae19d60c68471503f52b6adaf9325fd1cff529987f7e7",
		},
		"go/token": {
			actual:   tokenContract,
			expected: "d75894a3fe752821d081ad0183d0bf3153af632871d49137d24882ee67f882a4",
		},
		"types.Universe": {
			actual:   universeContract,
			expected: "50daaf97c83b52de1b6cd2714c3aa827c64bea5637044177914bf2230caae5d3",
		},
	} {
		if actual := fingerprint(contract.actual); actual != contract.expected {
			t.Errorf(
				"%s contract fingerprint = %s, replace pinned %s",
				name,
				actual,
				contract.expected,
			)
		}
	}

	root := repositoryRoot(t)
	dispatchPath := filepath.Join(root, "internal", "emit", "dispatch.go")
	for category, functionName := range map[nodeCategory]string{
		categoryDeclaration: "declarationObject",
		categoryExpression:  "Expression",
		categoryStatement:   "Statement",
	} {
		cases := dispatchCases(t, dispatchPath, functionName)
		if err := verifyDispatchTotality(universe, category, cases); err != nil {
			t.Errorf("%s: %v", functionName, err)
		}
	}
}

func verifyDispatchTotality(
	universe map[string]selectedASTForm,
	category nodeCategory,
	cases []string,
) error {
	expected := make(map[string]struct{})
	for name, form := range universe {
		if form.category == category &&
			name != "BadDecl" &&
			name != "BadExpr" &&
			name != "BadStmt" {
			expected[name] = struct{}{}
		}
	}
	actual := make(map[string]struct{}, len(cases))
	var wrong []string
	for _, name := range cases {
		form, exists := universe[name]
		if !exists || form.category != category {
			wrong = append(wrong, name)
			continue
		}
		actual[name] = struct{}{}
	}
	var missing []string
	var extra []string
	for name := range expected {
		if _, exists := actual[name]; !exists {
			missing = append(missing, name)
		}
	}
	for name := range actual {
		if _, exists := expected[name]; !exists {
			extra = append(extra, name)
		}
	}
	sort.Strings(wrong)
	sort.Strings(missing)
	sort.Strings(extra)
	if len(wrong) != 0 || len(missing) != 0 || len(extra) != 0 {
		return fmt.Errorf(
			"selected dispatch mismatch: wrong=%v missing=%v extra=%v",
			wrong,
			missing,
			extra,
		)
	}
	return nil
}

func selectedTokenContract(t *testing.T) []string {
	t.Helper()
	sourcePackage, err := importer.Default().Import("go/token")
	if err != nil {
		t.Fatal(err)
	}
	tokenObject := sourcePackage.Scope().Lookup("Token")
	if tokenObject == nil {
		t.Fatal("go/token.Token is absent")
	}
	var result []string
	for _, name := range sourcePackage.Scope().Names() {
		object, ok := sourcePackage.Scope().Lookup(name).(*types.Const)
		if ok && types.Identical(object.Type(), tokenObject.Type()) {
			result = append(result, name+":"+object.Val().ExactString())
		}
	}
	sort.Strings(result)
	return result
}

func selectedUniverseContract() ([]string, int) {
	names := types.Universe.Names()
	builtins := 0
	result := make([]string, 0, len(names))
	for _, name := range names {
		object := types.Universe.Lookup(name)
		if _, ok := object.(*types.Builtin); ok {
			builtins++
		}
		result = append(
			result,
			name+":"+types.ObjectString(object, packagePath),
		)
	}
	sort.Strings(result)
	return result, builtins
}

type selectedASTForm struct {
	category nodeCategory
	shape    string
}

func selectedASTUniverse(t *testing.T) map[string]selectedASTForm {
	t.Helper()
	sourcePackage, err := importer.Default().Import("go/ast")
	if err != nil {
		t.Fatal(err)
	}
	node := interfaceType(t, sourcePackage, "Node")
	expression := interfaceType(t, sourcePackage, "Expr")
	statement := interfaceType(t, sourcePackage, "Stmt")
	declaration := interfaceType(t, sourcePackage, "Decl")
	result := make(map[string]selectedASTForm)
	for _, name := range sourcePackage.Scope().Names() {
		object, ok := sourcePackage.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		named, ok := object.Type().(*types.Named)
		if !ok {
			continue
		}
		if _, ok := named.Underlying().(*types.Struct); !ok {
			continue
		}
		pointer := types.NewPointer(named)
		if !types.Implements(pointer, node) {
			continue
		}
		category := categoryParentOwned
		switch {
		case types.Implements(pointer, declaration):
			category = categoryDeclaration
		case types.Implements(pointer, expression):
			category = categoryExpression
		case types.Implements(pointer, statement):
			category = categoryStatement
		}
		if contextOwnedASTForm(name) {
			category = categoryParentOwned
		}
		result[name] = selectedASTForm{
			category: category,
			shape:    types.TypeString(named.Underlying(), packagePath),
		}
	}
	return result
}

func contextOwnedASTForm(name string) bool {
	switch name {
	case "ArrayType",
		"CaseClause",
		"ChanType",
		"CommClause",
		"Ellipsis",
		"FuncType",
		"InterfaceType",
		"KeyValueExpr",
		"MapType",
		"StructType":
		return true
	default:
		return false
	}
}

func packagePath(sourcePackage *types.Package) string {
	if sourcePackage == nil {
		return ""
	}
	return sourcePackage.Path()
}

func fingerprint(values []string) string {
	values = append([]string(nil), values...)
	sort.Strings(values)
	digest := sha256.New()
	for _, value := range values {
		digest.Write([]byte(value))
		digest.Write([]byte{0})
	}
	return fmt.Sprintf("%x", digest.Sum(nil))
}

func interfaceType(
	t *testing.T,
	sourcePackage *types.Package,
	name string,
) *types.Interface {
	t.Helper()
	object := sourcePackage.Scope().Lookup(name)
	if object == nil {
		t.Fatalf("go/ast.%s is absent", name)
	}
	named, ok := object.Type().(*types.Named)
	if !ok {
		t.Fatalf("go/ast.%s is %T, want named interface", name, object.Type())
	}
	value, ok := named.Underlying().(*types.Interface)
	if !ok {
		t.Fatalf("go/ast.%s is %T, want interface", name, named.Underlying())
	}
	return value.Complete()
}

func dispatchCases(
	t *testing.T,
	sourcePath string,
	functionName string,
) []string {
	t.Helper()
	result, err := inspectDispatch(sourcePath, functionName)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func inspectDispatch(
	sourcePath string,
	functionName string,
) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
	if err != nil {
		return nil, err
	}
	imports, err := importAliases(file)
	if err != nil {
		return nil, err
	}
	astAlias := imports["go/ast"]
	apiAlias := imports[modulePath+"/internal/emit/api"]
	if astAlias == "" || apiAlias == "" {
		return nil, fmt.Errorf("%s lacks exact ast/api imports", sourcePath)
	}
	var function *ast.FuncDecl
	for _, declaration := range file.Decls {
		candidate, ok := declaration.(*ast.FuncDecl)
		if ok && candidate.Name.Name == functionName {
			function = candidate
			break
		}
	}
	if function == nil {
		return nil, fmt.Errorf(
			"%s lacks root dispatch function %s",
			sourcePath,
			functionName,
		)
	}
	var switches []*ast.TypeSwitchStmt
	ast.Inspect(function.Body, func(node ast.Node) bool {
		if value, ok := node.(*ast.TypeSwitchStmt); ok {
			switches = append(switches, value)
		}
		return true
	})
	if len(switches) != 1 {
		return nil, fmt.Errorf(
			"%s.%s has %d type switches, want one root dispatch",
			sourcePath,
			functionName,
			len(switches),
		)
	}
	names := make(map[string]struct{})
	unsupportedCalls := 0
	for _, statement := range switches[0].Body.List {
		clause, ok := statement.(*ast.CaseClause)
		if !ok {
			return nil, fmt.Errorf(
				"%s has non-case type-switch statement %T",
				sourcePath,
				statement,
			)
		}
		if len(clause.List) == 0 {
			ast.Inspect(clause, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				qualifier, qualifierOK := selector.X.(*ast.Ident)
				if qualifierOK &&
					qualifier.Name == apiAlias &&
					selector.Sel.Name == "Unsupported" {
					unsupportedCalls++
				}
				return true
			})
			continue
		}
		for _, expression := range clause.List {
			pointer, ok := expression.(*ast.StarExpr)
			if !ok {
				return nil, fmt.Errorf(
					"%s has non-pointer dispatch case %T",
					sourcePath,
					expression,
				)
			}
			selector, ok := pointer.X.(*ast.SelectorExpr)
			if !ok {
				return nil, fmt.Errorf(
					"%s has non-go/ast dispatch case %T",
					sourcePath,
					pointer.X,
				)
			}
			qualifier, qualifierOK := selector.X.(*ast.Ident)
			if !qualifierOK ||
				qualifier.Name != astAlias ||
				selector.Sel.Name == "" {
				return nil, fmt.Errorf(
					"%s has non-go/ast dispatch case %T",
					sourcePath,
					pointer.X,
				)
			}
			if _, duplicate := names[selector.Sel.Name]; duplicate {
				return nil, fmt.Errorf(
					"%s dispatches %s more than once",
					sourcePath,
					selector.Sel.Name,
				)
			}
			names[selector.Sel.Name] = struct{}{}
		}
	}
	if unsupportedCalls != 1 {
		return nil, fmt.Errorf(
			"%s has %d exact api.Unsupported default calls, want one",
			sourcePath,
			unsupportedCalls,
		)
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func importAliases(file *ast.File) (map[string]string, error) {
	result := make(map[string]string)
	for _, sourceImport := range file.Imports {
		importPath, err := strconv.Unquote(sourceImport.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid import path %s", sourceImport.Path.Value)
		}
		alias := path.Base(importPath)
		if sourceImport.Name != nil {
			alias = sourceImport.Name.Name
		}
		if alias == "" || alias == "." || alias == "_" {
			return nil, fmt.Errorf("dispatch import %s has invalid alias %s", importPath, alias)
		}
		result[importPath] = alias
	}
	return result, nil
}

func TestDispatchTotalityMutationControls(t *testing.T) {
	for name, source := range map[string]string{
		"missing typed default": `package emit
import (
	"go/ast"
	"github.com/tsoniclang/gotots/internal/emit/api"
)
func dispatch(source ast.Expr) {
	switch source.(type) {
	case *ast.Ident:
	}
}
`,
		"duplicate owner": `package emit
import (
	"go/ast"
	"github.com/tsoniclang/gotots/internal/emit/api"
)
func dispatch(source ast.Expr) {
	switch source.(type) {
	case *ast.Ident:
	case *ast.Ident:
	default:
		api.Unsupported()
	}
}
`,
		"wrong unsupported owner": `package emit
import (
	"go/ast"
	wrong "example.com/wrong"
	"github.com/tsoniclang/gotots/internal/emit/api"
)
func dispatch(source ast.Expr) {
	switch source.(type) {
	case *ast.Ident:
	default:
		wrong.Unsupported()
	}
}
`,
		"wrong ast owner": `package emit
import (
	"go/ast"
	wrong "example.com/wrong"
	"github.com/tsoniclang/gotots/internal/emit/api"
)
func dispatch(source ast.Expr) {
	switch source.(type) {
	case *wrong.Ident:
	default:
		api.Unsupported()
	}
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			sourcePath := filepath.Join(t.TempDir(), "dispatch.go")
			if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := inspectDispatch(sourcePath, "dispatch"); err == nil {
				t.Fatal("dispatch mutation passed its owning gate")
			}
		})
	}

	t.Run("missing selected arm", func(t *testing.T) {
		universe := map[string]selectedASTForm{
			"BinaryExpr": {category: categoryExpression},
			"Ident":      {category: categoryExpression},
		}
		if err := verifyDispatchTotality(
			universe,
			categoryExpression,
			[]string{"Ident"},
		); err == nil {
			t.Fatal("missing selected dispatch arm passed its owning gate")
		}
	})

	t.Run("widened neighboring category", func(t *testing.T) {
		universe := map[string]selectedASTForm{
			"Ident":      {category: categoryExpression},
			"ReturnStmt": {category: categoryStatement},
		}
		if err := verifyDispatchTotality(
			universe,
			categoryExpression,
			[]string{"Ident", "ReturnStmt"},
		); err == nil {
			t.Fatal("widened neighboring dispatch arm passed its owning gate")
		}
	})
}
