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
	var requests []api.RootRequest
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
) (tsgo.VariableStatement, []api.RootRequest, error) {
	if source.Doc != nil || source.Comment != nil ||
		len(source.Names) == 0 ||
		(len(source.Values) != 0 && len(source.Names) != len(source.Values)) ||
		(len(source.Values) == 0 && source.Type == nil) {
		return nil, nil,
			api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				source,
			)
	}

	declarations := make([]tsgo.VariableDeclaration, 0, len(source.Names))
	var requests []api.RootRequest
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

		value, err := localValue(context, children, source, sourceName, index, object)
		if err != nil {
			return nil, nil, err
		}
		if len(value.Before()) != 0 {
			return nil, nil,
				api.Unsupported(
					context.WithRole(api.RoleLocalValue),
					api.CategoryExpression,
					sourceName,
				)
		}
		targetName, selected := context.AddressableStorage().Name(context, object)
		if !selected {
			targetName, err = context.Names().Declare(object)
		} else {
			value, err = context.AddressableStorage().Cell(
				context,
				children,
				sourceName,
				object.Type(),
				value,
			)
		}
		if err != nil {
			return nil, nil, err
		}
		var targetType tsgo.TypeNode
		if !selected &&
			context.Values().RequiresExplicitType(context, object.Type()) {
			represented, err := children.RepresentedType(
				context.WithRole(api.RoleLocalType),
				sourceName,
				object.Type(),
			)
			if err != nil {
				return nil, nil, err
			}
			targetType = represented.Value()
			requests = append(requests, represented.Requests()...)
		}
		declarations = append(
			declarations,
			context.Factory().VariableDeclaration(
				context.Factory().Identifier(targetName),
				nil,
				targetType,
				value.Value(),
			),
		)
		requests = append(
			requests,
			value.Requests()...,
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

func localValue(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
	sourceName *ast.Ident,
	index int,
	object *types.Var,
) (api.ExpressionEmission, error) {
	valueContext := context.
		WithRole(api.RoleLocalValue).
		WithExpectedType(object.Type())
	if len(source.Values) == 0 {
		return context.Values().Zero(valueContext, sourceName, object.Type())
	}
	sourceValue := source.Values[index]
	valueType := context.TypesInfo().TypeOf(sourceValue)
	if valueType == nil || !types.AssignableTo(valueType, object.Type()) {
		return api.ExpressionEmission{},
			api.Unsupported(
				valueContext,
				api.CategoryExpression,
				sourceValue,
			)
	}
	value, err := children.Expression(valueContext, sourceValue)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return context.Values().Copy(
		valueContext,
		sourceValue,
		object.Type(),
		value,
	)
}
