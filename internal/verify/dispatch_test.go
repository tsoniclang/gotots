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
		categoryExpression:  23,
		categoryParentOwned: 10,
		categoryStatement:   21,
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
			expected: "e6b7c7c1534cbbe58eed5c8589be26f0c6bafdeb8b2e18e1a1206fe441f903e5",
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
	for category, fileName := range map[nodeCategory]string{
		categoryDeclaration: "dispatch_declaration.go",
		categoryExpression:  "dispatch_expression.go",
		categoryStatement:   "dispatch_statement.go",
	} {
		cases := dispatchCases(t, filepath.Join(root, "internal", "emit", fileName))
		for _, name := range cases {
			if actual, exists := universe[name]; !exists || actual.category != category {
				t.Errorf(
					"%s dispatches %s, selected go/ast category is %s",
					fileName,
					name,
					actual.category,
				)
			}
		}
		for _, bad := range []string{"BadDecl", "BadExpr", "BadStmt"} {
			if contains(cases, bad) {
				t.Errorf("%s handles parser recovery form %s", fileName, bad)
			}
		}
	}
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
		result[name] = selectedASTForm{
			category: category,
			shape:    types.TypeString(named.Underlying(), packagePath),
		}
	}
	return result
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

func dispatchCases(t *testing.T, sourcePath string) []string {
	t.Helper()
	result, err := inspectDispatch(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func inspectDispatch(sourcePath string) ([]string, error) {
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
	var switches []*ast.TypeSwitchStmt
	ast.Inspect(file, func(node ast.Node) bool {
		if value, ok := node.(*ast.TypeSwitchStmt); ok {
			switches = append(switches, value)
		}
		return true
	})
	if len(switches) != 1 {
		return nil, fmt.Errorf(
			"%s has %d type switches, want one root dispatch",
			sourcePath,
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
			if _, err := inspectDispatch(sourcePath); err == nil {
				t.Fatal("dispatch mutation passed its owning gate")
			}
		})
	}
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
