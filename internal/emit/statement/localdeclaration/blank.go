package localdeclaration

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitBlankAwareSpec(
	context api.Context,
	children api.ChildEmitter,
	source *ast.ValueSpec,
) ([]tsgo.Statement, []api.RootRequest, error) {
	var statements []tsgo.Statement
	var requests []api.RootRequest
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
			if len(source.Values) == 0 {
				continue
			}
			discarded, err := discardedLocalValue(
				context,
				children,
				source.Values[index],
			)
			if err != nil {
				return nil, nil, err
			}
			statements = append(statements, discarded.Statements()...)
			requests = append(requests, discarded.Requests()...)
			continue
		}
		object, ok := context.TypesInfo().Defs[sourceName].(*types.Var)
		if !ok {
			return nil, nil, api.Unsupported(
				context.WithRole(api.RoleLocalDeclaration),
				api.CategoryStatement,
				sourceName,
			)
		}
		_, callableZero := callable.Signature(object.Type())
		callableZero = callableZero && len(source.Values) == 0
		value, err := localValue(
			context,
			children,
			source,
			sourceName,
			index,
			object,
		)
		if err != nil {
			return nil, nil, err
		}
		declaration, before, declarationRequests, err :=
			localVariableDeclaration(context, children, binding{
				sourceName:      sourceName,
				object:          object,
				value:           value,
				omitInitializer: callableZero,
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

func discardedLocalValue(
	context api.Context,
	children api.ChildEmitter,
	source ast.Expr,
) (api.StatementEmission, error) {
	sourceType := context.TypesInfo().TypeOf(source)
	if sourceType == nil {
		return api.StatementEmission{}, api.Unsupported(
			context.WithRole(api.RoleLocalValue),
			api.CategoryExpression,
			source,
		)
	}
	if facts, ok := context.TypesInfo().Types[source]; ok && facts.Value != nil {
		return api.NewStatementEmission(nil, nil)
	}
	expectedType := sourceType
	if basic, ok := types.Unalias(sourceType).(*types.Basic); ok &&
		basic.Info()&types.IsUntyped != 0 {
		expectedType = types.Default(sourceType)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(expectedType),
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := value.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(value.Value()),
	)
	return api.NewStatementEmission(statements, value.Requests())
}
