package localdeclaration

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMultipleResultSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
	results *types.Tuple,
) ([]tsgo.Statement, []api.RootRequest, error) {
	if results == nil || results.Len() != len(source.Names) {
		return nil, nil, api.Unsupported(
			context.WithRole(api.RoleLocalDeclaration),
			api.CategoryStatement,
			source,
		)
	}
	capture, err := resulttuple.Emit(
		context,
		children,
		source.Values[0],
		results,
		api.RoleLocalValue,
	)
	if err != nil {
		return nil, nil, err
	}
	statements := capture.Statements()
	requests := capture.Requests()
	for index, sourceName := range source.Names {
		if sourceName.Name == "_" {
			if object := context.TypesInfo().Defs[sourceName]; object != nil &&
				object.Name() != "_" {
				return nil, nil, api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceName,
				)
			}
			continue
		}
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Var)
		if !ok || !types.AssignableTo(results.At(index).Type(), object.Type()) {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				sourceName,
			)
		}
		element, err := capture.Element(context, index)
		if err != nil {
			return nil, nil, err
		}
		value, err := context.Values().Copy(
			context.WithRole(api.RoleLocalValue),
			source.Values[0],
			object.Type(),
			api.DirectExpression(element),
		)
		if err != nil {
			return nil, nil, err
		}
		declaration, before, declarationRequests, err :=
			localVariableDeclaration(context, children, binding{
				sourceName: sourceName,
				object:     object,
				value:      value,
			})
		if err != nil {
			return nil, nil, err
		}
		statements = append(statements, before...)
		statements = append(
			statements,
			context.Factory().VariableStatement(
				nil,
				context.Factory().VariableDeclarationList(
					[]tsgo.VariableDeclaration{declaration},
					tsgo.NodeFlagsLet,
				),
			),
		)
		requests = append(requests, declarationRequests...)
	}
	return statements, requests, nil
}
