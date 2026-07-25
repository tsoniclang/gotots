package localdeclaration

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.DeclStmt,
) (api.StatementEmission, error) {
	declaration, ok := source.Decl.(*ast.GenDecl)
	if !ok || declaration.Doc != nil || declaration.Tok != token.VAR ||
		len(declaration.Specs) == 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}

	statements := make([]tsgo.Statement, 0, len(declaration.Specs))
	var requests []api.PlacementRequest
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			return api.StatementEmission{},
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceSpec,
				)
		}
		target, targetRequests, err := emitSpec(context, children, spec)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, target)
		requests = append(requests, targetRequests...)
	}
	return api.NewStatementEmission(statements, requests)
}

func emitSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
) (tsgo.VariableStatement, []api.PlacementRequest, error) {
	if source.Doc != nil || source.Comment != nil ||
		len(source.Names) == 0 ||
		len(source.Names) != len(source.Values) {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				source,
			)
	}

	declarations := make([]tsgo.VariableDeclaration, 0, len(source.Names))
	var requests []api.PlacementRequest
	for index, sourceName := range source.Names {
		if sourceName.Name == "_" {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceName,
				)
		}
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Var)
		if !ok {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalDeclaration),
					api.CategoryStatement,
					sourceName,
				)
		}
		if source.Type != nil &&
			!types.Identical(context.TypesInfo().TypeOf(source.Type), object.Type()) {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalType),
					api.CategoryType,
					source.Type,
				)
		}

		sourceValue := source.Values[index]
		valueType := context.TypesInfo().TypeOf(sourceValue)
		if valueType == nil || !types.AssignableTo(valueType, object.Type()) {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalValue),
					api.CategoryExpression,
					sourceValue,
				)
		}
		value, err := children.Expression(
			context.
				WithRole(api.RoleLocalValue).
				WithExpectedType(object.Type()),
			sourceValue,
		)
		if err != nil {
			return nil, nil, err
		}
		if len(value.Before()) != 0 {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalValue),
					api.CategoryExpression,
					sourceValue,
				)
		}
		targetType, err := children.RepresentedType(
			context.WithRole(api.RoleLocalType),
			sourceName,
			object.Type(),
		)
		if err != nil {
			return nil, nil, err
		}
		targetName, err := context.Names().Declare(object)
		if err != nil {
			return nil, nil, err
		}
		declarations = append(
			declarations,
			context.Factory().VariableDeclaration(
				context.Factory().Identifier(targetName),
				nil,
				targetType.Value(),
				value.Value(),
			),
		)
		requests = append(
			requests,
			api.CombineRequests(value.Requests(), targetType.Requests())...,
		)
	}
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			declarations,
			tsgo.NodeFlagsLet,
		),
	), requests, nil
}
