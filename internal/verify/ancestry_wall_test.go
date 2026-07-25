package verify

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestTraversalAncestryUsesOneNonEscapingStack(t *testing.T) {
	targets := []string{
		"internal/language/structure/extract_file.go",
		"internal/stagecheck/derive_structure.go",
	}
	for _, target := range targets {
		path := filepath.Join(repoRoot(t), filepath.FromSlash(target))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		problems, err := ancestryStackProblems(path, raw)
		if err != nil {
			t.Fatal(err)
		}
		for _, problem := range problems {
			t.Errorf("%s: %s", target, problem)
		}
	}

	control := []byte(`package control
type step struct{}
type builder struct{ path []step }
func (builder *builder) scan() {
	copied := append([]step(nil), builder.path...)
	_ = copied
}`)
	problems, err := ancestryStackProblems("control.go", control)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatal("ancestry wall did not detect copied-stack control")
	}
}

func ancestryStackProblems(
	filename string,
	raw []byte,
) ([]string, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, raw, 0)
	if err != nil {
		return nil, err
	}
	parents := map[ast.Node]ast.Node{}
	var stack []ast.Node
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})

	var problems []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Name.Name != "ensurePath" &&
				hasAncestrySliceParameter(node.Type.Params) {
				problems = append(
					problems,
					positionProblem(
						fileSet, node.Pos(),
						"ancestry slice crosses a traversal call",
					),
				)
			}
		case *ast.AssignStmt:
			for _, expression := range node.Rhs {
				if assignmentKeepsPathLocal(expression, node) {
					continue
				}
				if assignmentEscapesPath(expression) {
					problems = append(
						problems,
						positionProblem(
							fileSet, node.Pos(),
							"ancestry stack escapes through assignment",
						),
					)
				}
			}
		case *ast.CallExpr:
			if !callContainsPathSelector(node) {
				return true
			}
			if canonicalPathAppend(
				node, assignmentParent(parents[node]),
			) ||
				callName(node.Fun) == "len" ||
				callName(node.Fun) == "ensurePath" {
				return true
			}
			problems = append(
				problems,
				positionProblem(
					fileSet, node.Pos(),
					"ancestry stack escapes through call",
				),
			)
		}
		return true
	})
	return problems, nil
}

func assignmentKeepsPathLocal(
	expression ast.Expr,
	assignment *ast.AssignStmt,
) bool {
	if len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	if samePathSelector(
		assignment.Lhs[0], pathSelector(expression),
	) {
		return true
	}
	if call, ok := expression.(*ast.CallExpr); ok &&
		canonicalPathAppend(call, assignment) {
		return true
	}
	return canonicalPathTruncate(expression, assignment)
}

func assignmentEscapesPath(expression ast.Expr) bool {
	switch expression := expression.(type) {
	case *ast.SelectorExpr:
		return pathSelector(expression) != nil
	case *ast.SliceExpr:
		return containsPathSelector(expression.X)
	case *ast.CallExpr:
		return callName(expression.Fun) == "append" &&
			callContainsPathSelector(expression)
	default:
		return false
	}
}

func hasAncestrySliceParameter(fields *ast.FieldList) bool {
	if fields == nil {
		return false
	}
	for _, field := range fields.List {
		array, ok := field.Type.(*ast.ArrayType)
		if !ok || array.Len != nil {
			continue
		}
		element, ok := array.Elt.(*ast.Ident)
		if ok &&
			(element.Name == "pathStep" ||
				element.Name == "derivedPathStep") {
			return true
		}
	}
	return false
}

func canonicalPathAppend(
	call *ast.CallExpr,
	assignment *ast.AssignStmt,
) bool {
	if callName(call.Fun) != "append" ||
		call.Ellipsis.IsValid() ||
		len(call.Args) != 2 ||
		assignment == nil ||
		assignment.Tok != token.ASSIGN ||
		len(assignment.Lhs) != 1 ||
		len(assignment.Rhs) != 1 ||
		assignment.Rhs[0] != call {
		return false
	}
	return samePathSelector(
		assignment.Lhs[0], pathSelector(call.Args[0]),
	)
}

func canonicalPathTruncate(
	expression ast.Expr,
	assignment *ast.AssignStmt,
) bool {
	slice, ok := expression.(*ast.SliceExpr)
	if !ok ||
		assignment.Tok != token.ASSIGN ||
		len(assignment.Lhs) != 1 ||
		len(assignment.Rhs) != 1 ||
		assignment.Rhs[0] != expression {
		return false
	}
	return samePathSelector(
		assignment.Lhs[0], pathSelector(slice.X),
	)
}

func assignmentParent(node ast.Node) *ast.AssignStmt {
	assignment, _ := node.(*ast.AssignStmt)
	return assignment
}

func callContainsPathSelector(call *ast.CallExpr) bool {
	for _, argument := range call.Args {
		if containsPathSelector(argument) {
			return true
		}
	}
	return false
}

func containsPathSelector(expression ast.Expr) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if pathSelector(node) != nil {
			found = true
			return false
		}
		return !found
	})
	return found
}

func pathSelector(node ast.Node) *ast.SelectorExpr {
	selector, ok := node.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "path" {
		return nil
	}
	return selector
}

func samePathSelector(
	left ast.Expr,
	right *ast.SelectorExpr,
) bool {
	leftSelector := pathSelector(left)
	if leftSelector == nil || right == nil {
		return false
	}
	leftOwner, leftOK := leftSelector.X.(*ast.Ident)
	rightOwner, rightOK := right.X.(*ast.Ident)
	return leftOK && rightOK && leftOwner.Name == rightOwner.Name
}

func callName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.SelectorExpr:
		return expression.Sel.Name
	default:
		return ""
	}
}

func positionProblem(
	fileSet *token.FileSet,
	position token.Pos,
	message string,
) string {
	location := fileSet.Position(position)
	return fmt.Sprintf("%s:%d: %s", location.Filename, location.Line, message)
}
