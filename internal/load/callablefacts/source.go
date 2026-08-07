package callablefacts

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
)

type PointeeReadEvidence uint8

const (
	PointeeReadInvalid PointeeReadEvidence = iota
	PointeeReadEntryStable
)

func SourceCallableIdentity(function *types.Func) (string, error) {
	if function == nil || function.Pkg() == nil || function.Name() == "" {
		return "", &callableabi.Error{Reason: "source callable identity is incomplete"}
	}
	signature := function.Signature()
	if signature.Recv() == nil {
		return callableabi.PackageFunctionIdentity(function.Pkg().Path(), function.Name())
	}
	receiver := types.TypeString(signature.Recv().Type(), func(
		selected *types.Package,
	) string {
		if selected == nil {
			return ""
		}
		return selected.Path()
	})
	return callableabi.MethodIdentity(
		function.Pkg().Path(),
		receiver,
		function.Name(),
	)
}

func PointeeValueEvidence(
	declaration *ast.FuncDecl,
	parameter *types.Var,
	info *types.Info,
) PointeeReadEvidence {
	if declaration == nil || declaration.Body == nil || parameter == nil || info == nil {
		return PointeeReadInvalid
	}
	parents := parentNodes(declaration.Body)
	if !pointeeValueReadAtEntry(declaration, parameter, info) ||
		!parameterUsesArePointeeReads(
			declaration.Body,
			parameter,
			info,
			parents,
		) || bodyMayMutatePointee(declaration.Body, info, parents) {
		return PointeeReadInvalid
	}
	return PointeeReadEntryStable
}

func pointeeValueReadAtEntry(
	declaration *ast.FuncDecl,
	parameter *types.Var,
	info *types.Info,
) bool {
	if len(declaration.Body.List) == 0 {
		return false
	}
	var expression ast.Expr
	switch statement := declaration.Body.List[0].(type) {
	case *ast.ReturnStmt:
		if len(statement.Results) == 0 {
			return false
		}
		expression = statement.Results[0]
	case *ast.AssignStmt:
		if len(statement.Rhs) == 0 || !localAssignments(statement.Lhs, info) {
			return false
		}
		expression = statement.Rhs[0]
	case *ast.DeclStmt:
		declaration, ok := statement.Decl.(*ast.GenDecl)
		if !ok || declaration.Tok != token.VAR || len(declaration.Specs) == 0 {
			return false
		}
		value, ok := declaration.Specs[0].(*ast.ValueSpec)
		if !ok || len(value.Values) == 0 {
			return false
		}
		expression = value.Values[0]
	default:
		return false
	}
	return entryPointeeRead(expression, parameter, info)
}

func entryPointeeRead(
	expression ast.Expr,
	parameter *types.Var,
	info *types.Info,
) bool {
	expression = unparenthesized(expression)
	dereference, ok := expression.(*ast.StarExpr)
	if ok {
		identifier, identifierOK := unparenthesized(dereference.X).(*ast.Ident)
		return identifierOK && info.Uses[identifier] == parameter
	}
	conversion, ok := expression.(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 || conversion.Ellipsis != token.NoPos {
		return false
	}
	callee, ok := info.Types[conversion.Fun]
	if !ok || !callee.IsType() {
		return false
	}
	return entryPointeeRead(conversion.Args[0], parameter, info)
}

func parentNodes(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) != 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func parameterUsesArePointeeReads(
	body *ast.BlockStmt,
	parameter *types.Var,
	info *types.Info,
	parents map[ast.Node]ast.Node,
) bool {
	reads := 0
	for identifier, object := range info.Uses {
		if object != parameter || !insideNode(identifier, body, parents) {
			continue
		}
		current := ast.Node(identifier)
		for {
			parent := parents[current]
			if _, ok := parent.(*ast.ParenExpr); !ok {
				break
			}
			current = parent
		}
		dereference, ok := parents[current].(*ast.StarExpr)
		if !ok || insideFunctionLiteral(dereference, body, parents) ||
			dereferenceIsLocation(dereference, parents) {
			return false
		}
		reads++
	}
	return reads != 0
}

func insideNode(
	node ast.Node,
	ancestor ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	for node != nil {
		if node == ancestor {
			return true
		}
		node = parents[node]
	}
	return false
}

func insideFunctionLiteral(
	node ast.Node,
	body *ast.BlockStmt,
	parents map[ast.Node]ast.Node,
) bool {
	for node != nil && node != body {
		if _, ok := node.(*ast.FuncLit); ok {
			return true
		}
		node = parents[node]
	}
	return false
}

func dereferenceIsLocation(
	dereference *ast.StarExpr,
	parents map[ast.Node]ast.Node,
) bool {
	current := ast.Node(dereference)
	for {
		parent := parents[current]
		if _, ok := parent.(*ast.ParenExpr); !ok {
			break
		}
		current = parent
	}
	switch parent := parents[current].(type) {
	case *ast.AssignStmt:
		return nodeInExpressions(current, parent.Lhs)
	case *ast.IncDecStmt:
		return insideNode(current, parent.X, parents)
	case *ast.RangeStmt:
		return insideNode(current, parent.Key, parents) ||
			insideNode(current, parent.Value, parents)
	case *ast.UnaryExpr:
		return parent.Op == token.AND
	default:
		return false
	}
}

func nodeInExpressions(target ast.Node, expressions []ast.Expr) bool {
	for _, expression := range expressions {
		found := false
		ast.Inspect(expression, func(node ast.Node) bool {
			if node == target {
				found = true
				return false
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

func bodyMayMutatePointee(
	body *ast.BlockStmt,
	info *types.Info,
	parents map[ast.Node]ast.Node,
) bool {
	unsafe := false
	ast.Inspect(body, func(node ast.Node) bool {
		if node == nil || unsafe {
			return !unsafe
		}
		if node != body && insideFunctionLiteral(node, body, parents) {
			return false
		}
		switch selected := node.(type) {
		case *ast.CallExpr:
			unsafe = !pointeeStableCall(selected, info)
		case *ast.AssignStmt:
			unsafe = !localAssignments(selected.Lhs, info)
		case *ast.IncDecStmt:
			unsafe = !localAssignments([]ast.Expr{selected.X}, info)
		case *ast.SendStmt, *ast.GoStmt, *ast.DeferStmt, *ast.SelectStmt:
			unsafe = true
		case *ast.UnaryExpr:
			unsafe = selected.Op == token.ARROW
		case *ast.RangeStmt:
			target := info.TypeOf(selected.X)
			if target == nil {
				unsafe = true
			} else {
				underlying := types.Unalias(target).Underlying()
				switch underlying.(type) {
				case *types.Chan, *types.Signature:
					unsafe = true
				}
			}
		}
		return !unsafe
	})
	return unsafe
}

func pointeeStableCall(source *ast.CallExpr, info *types.Info) bool {
	callee, ok := info.Types[source.Fun]
	if ok && callee.IsType() {
		return true
	}
	var object types.Object
	switch target := unparenthesized(source.Fun).(type) {
	case *ast.Ident:
		object = info.Uses[target]
	case *ast.SelectorExpr:
		object = info.Uses[target.Sel]
	}
	for _, allowed := range []types.Object{
		types.Universe.Lookup("len"),
		types.Universe.Lookup("cap"),
		types.Universe.Lookup("complex"),
		types.Universe.Lookup("real"),
		types.Universe.Lookup("imag"),
		types.Universe.Lookup("min"),
		types.Universe.Lookup("max"),
		types.Universe.Lookup("new"),
		types.Universe.Lookup("make"),
		types.Unsafe.Scope().Lookup("Sizeof"),
		types.Unsafe.Scope().Lookup("Alignof"),
		types.Unsafe.Scope().Lookup("Offsetof"),
	} {
		if object != nil && object == allowed {
			return true
		}
	}
	return false
}

func localAssignments(expressions []ast.Expr, info *types.Info) bool {
	for _, expression := range expressions {
		identifier, ok := unparenthesized(expression).(*ast.Ident)
		if !ok {
			return false
		}
		if identifier.Name == "_" {
			continue
		}
		variable, ok := info.ObjectOf(identifier).(*types.Var)
		if !ok || variable.Parent() == nil ||
			variable.Pkg() != nil && variable.Parent() == variable.Pkg().Scope() {
			return false
		}
	}
	return true
}

func unparenthesized(source ast.Expr) ast.Expr {
	for {
		parenthesized, ok := source.(*ast.ParenExpr)
		if !ok {
			return source
		}
		source = parenthesized.X
	}
}
