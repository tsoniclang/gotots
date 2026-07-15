package census

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// classifyCall assigns every call expression exactly one semantic class
// using go/types evidence. The classification is exhaustive: a call shape
// outside the reviewed set is an error, never a bucket.
//
// Classes:
//
//	conversion          - the operand in call position is a type
//	builtin:<name>      - predeclared operation (recorded in builtins)
//	funcCall            - direct call of a declared function
//	methodCall          - method call with a concrete receiver
//	interfaceMethodCall - method call dispatched through an interface
//	funcValueCall       - call of a function-typed value (variable, field,
//	                      parameter, result of another call, func literal)
//	methodExprCall      - call of a method expression (T.Method)(recv, ...)
func classifyCall(info *types.Info, call *ast.CallExpr, stats *fileStats, rareOnly func(string, token.Pos)) error {
	if tv, ok := info.Types[call.Fun]; ok && tv.IsType() {
		stats.constructs["conversion"]++
		return nil
	}

	callee := ast.Unparen(call.Fun)

	// Builtins are identified by object identity, never spelling.
	if ident := calleeIdent(callee); ident != nil {
		if builtin, ok := info.Uses[ident].(*types.Builtin); ok {
			stats.builtins[builtin.Name()]++
			rareOnly(builtin.Name(), call.Pos())
			return nil
		}
	}

	// Method and field selections carry explicit selection evidence.
	if selector, ok := callee.(*ast.SelectorExpr); ok {
		if selection, ok := info.Selections[selector]; ok {
			switch selection.Kind() {
			case types.MethodVal:
				if _, isInterface := selection.Recv().Underlying().(*types.Interface); isInterface {
					stats.constructs["interfaceMethodCall"]++
				} else {
					stats.constructs["methodCall"]++
				}
				return nil
			case types.MethodExpr:
				stats.constructs["methodExprCall"]++
				return nil
			case types.FieldVal:
				stats.constructs["funcValueCall"]++
				return nil
			default:
				return fmt.Errorf("unreviewed selection kind %v in call at %v", selection.Kind(), call.Pos())
			}
		}
		// Package-qualified selector: resolved through Uses below.
	}

	if ident := calleeIdent(callee); ident != nil {
		switch object := info.Uses[ident].(type) {
		case *types.Func:
			stats.constructs["funcCall"]++
			return nil
		case *types.Var:
			stats.constructs["funcValueCall"]++
			return nil
		case nil:
			// Defs (e.g. recursive reference)? fall through to type check.
		default:
			_ = object
		}
	}

	// Everything else must at least be a value of function type: a func
	// literal called immediately, the result of another call, an indexed
	// function value, or an instantiated generic function.
	tv, ok := info.Types[call.Fun]
	if !ok || tv.Type == nil {
		return fmt.Errorf("call at %v has no type evidence", call.Pos())
	}
	if _, isSignature := tv.Type.Underlying().(*types.Signature); !isSignature {
		return fmt.Errorf("unreviewed call variant at %v: callee type %s", call.Pos(), tv.Type)
	}
	stats.constructs["funcValueCall"]++
	return nil
}

func calleeIdent(callee ast.Expr) *ast.Ident {
	switch f := callee.(type) {
	case *ast.Ident:
		return f
	case *ast.SelectorExpr:
		return f.Sel
	case *ast.IndexExpr:
		return calleeIdent(ast.Unparen(f.X))
	case *ast.IndexListExpr:
		return calleeIdent(ast.Unparen(f.X))
	}
	return nil
}
