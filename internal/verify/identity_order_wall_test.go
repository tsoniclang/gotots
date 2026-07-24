package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

type renderedIdentityUse struct {
	position token.Position
	class    string
}

func TestRenderedIdentityDoesNotDriveComputation(t *testing.T) {
	fset := token.NewFileSet()
	for _, path := range productionGoFiles(t) {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, violation := range renderedIdentityUses(
			fset, file, path,
		) {
			t.Errorf(
				"%s: rendered String() drives %s; use typed identity equality or Compare",
				violation.position,
				violation.class,
			)
		}
		for _, violation := range formattedSemanticSortKeys(
			fset, file, path,
		) {
			t.Errorf(
				"%s: %s; compare typed structural components",
				violation.position,
				violation.class,
			)
		}
	}
}

func TestRenderedIdentityComputationDetectorRejectsMutation(
	t *testing.T,
) {
	const mutated = `package mutation
func compare(left, right Identity, indexed map[string]bool) bool {
	indexed[left.String()] = true
	return left.String() < right.String()
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(
		fset, "identity_mutation.go", mutated, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	uses := renderedIdentityUses(fset, file, "identity_mutation.go")
	if len(uses) != 2 {
		t.Fatalf(
			"rendered-identity wall found %d mutation sites, want 2",
			len(uses),
		)
	}
	if uses[0].class != "lookup" ||
		uses[1].class != "comparison" {
		t.Fatalf(
			"rendered-identity wall classified mutation as %q/%q",
			uses[0].class,
			uses[1].class,
		)
	}
}

func TestFormattedSemanticSortKeyDetectorRejectsMutation(
	t *testing.T,
) {
	const mutated = `package frontend
import (
	"fmt"
	"sort"
)
func bindingOrder(id Identity) string {
	return fmt.Sprintf("%s", id)
}
func order(ids []Identity) {
	sort.Slice(ids, func(left, right int) bool {
		return bindingOrder(ids[left]) < bindingOrder(ids[right])
	})
}`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(
		fset, "formatted_order_mutation.go", mutated, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	uses := formattedSemanticSortKeys(
		fset,
		file,
		filepath.Join(
			"internal", "language", "frontend",
			"formatted_order_mutation.go",
		),
	)
	if len(uses) != 2 {
		t.Fatalf(
			"formatted-sort wall found %d mutation sites, want 2",
			len(uses),
		)
	}
}

func renderedIdentityUses(
	fset *token.FileSet,
	file *ast.File,
	path string,
) []renderedIdentityUse {
	var out []renderedIdentityUse
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.IndexExpr:
			if containsStringCall(node.Index) {
				out = append(out, renderedIdentityUse{
					position: fset.Position(node.Index.Pos()),
					class:    "lookup",
				})
			}
		case *ast.BinaryExpr:
			if comparisonOperator(node.Op) &&
				(directStringCall(node.X) ||
					directStringCall(node.Y)) &&
				!artifactCanonicalComparison(path, node) &&
				!digestStringComparison(node) {
				out = append(out, renderedIdentityUse{
					position: fset.Position(node.Pos()),
					class:    "comparison",
				})
			}
		}
		return true
	})
	return out
}

func formattedSemanticSortKeys(
	fset *token.FileSet,
	file *ast.File,
	path string,
) []renderedIdentityUse {
	slashPath := filepath.ToSlash(path)
	if !strings.Contains(
		slashPath,
		"internal/language/frontend/",
	) && !strings.Contains(
		slashPath,
		"internal/language/semantic/",
	) {
		return nil
	}
	renderingHelpers := map[string]bool{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok ||
			function.Type.Results == nil ||
			len(function.Type.Results.List) != 1 {
			continue
		}
		result, ok := function.Type.Results.List[0].Type.(*ast.Ident)
		if !ok || result.Name != "string" {
			continue
		}
		if containsFormattingCall(function.Body) {
			renderingHelpers[function.Name.Name] = true
		}
	}
	var out []renderedIdentityUse
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 2 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, packageOK := selector.X.(*ast.Ident)
		if !packageOK ||
			pkg.Name != "sort" ||
			(selector.Sel.Name != "Slice" &&
				selector.Sel.Name != "SliceStable") {
			return true
		}
		comparator, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			return true
		}
		ast.Inspect(comparator.Body, func(node ast.Node) bool {
			candidate, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if formattingCall(candidate) {
				out = append(out, renderedIdentityUse{
					position: fset.Position(candidate.Pos()),
					class:    "formatted semantic sort key",
				})
				return false
			}
			helper, ok := candidate.Fun.(*ast.Ident)
			if ok && renderingHelpers[helper.Name] {
				out = append(out, renderedIdentityUse{
					position: fset.Position(candidate.Pos()),
					class:    "rendered semantic sort helper",
				})
				return false
			}
			return true
		})
		return true
	})
	return out
}

func containsFormattingCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if formattingCall(call) || directStringCall(call) {
			found = true
			return false
		}
		return true
	})
	return found
}

func formattingCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Sprintf" {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "fmt"
}

func directStringCall(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "String"
}

func containsStringCall(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == "String" &&
			len(call.Args) == 0 {
			found = true
			return false
		}
		return true
	})
	return found
}

func artifactCanonicalComparison(
	path string,
	expression *ast.BinaryExpr,
) bool {
	if expression.Op != token.EQL && expression.Op != token.NEQ {
		return false
	}
	base := filepath.Base(path)
	slashPath := filepath.ToSlash(path)
	return strings.HasPrefix(base, "artifact_") ||
		base == "artifact.go" ||
		(strings.Contains(slashPath, "/internal/identity/") &&
			strings.HasSuffix(base, "parse.go")) ||
		strings.HasSuffix(
			slashPath,
			"/internal/scope/contract/artifact.go",
		)
}

func digestStringComparison(expression *ast.BinaryExpr) bool {
	return isDigestStringCall(expression.X) ||
		isDigestStringCall(expression.Y)
}

func isDigestStringCall(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "String" {
		return false
	}
	receiver, ok := selector.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	digest, ok := receiver.Fun.(*ast.SelectorExpr)
	return ok && strings.HasSuffix(digest.Sel.Name, "Digest")
}

func comparisonOperator(operator token.Token) bool {
	switch operator {
	case token.EQL, token.NEQ, token.LSS,
		token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}
