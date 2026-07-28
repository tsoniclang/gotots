package assignment

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func emitMultipleResults(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
	results *types.Tuple,
) (api.StatementEmission, error) {
	if results == nil || results.Len() != len(source.Lhs) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	targets, locationBefore, locationRequests, err := parallelTargets(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	for index, target := range targets {
		if target.discard {
			continue
		}
		targetType := target.target.SourceType()
		if target.declaration {
			targetType = target.object.Type()
		}
		if !types.AssignableTo(results.At(index).Type(), targetType) {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
	}

	role := api.RoleAssignmentValue
	if source.Tok == token.DEFINE {
		role = api.RoleLocalValue
	}
	value, err := children.Expression(
		context.
			WithRole(role).
			WithExpectedResults(results),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	temporaryName, err := context.Names().Temporary(api.TemporaryMultipleResults)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(locationBefore, value.Before()...)
	statements = append(
		statements,
		variableStatement(
			context,
			tsgo.NodeFlagsConst,
			temporaryName,
			value.Value(),
		),
	)
	requests := api.CombineRequests(locationRequests, value.Requests())

	for index, target := range targets {
		if target.discard {
			continue
		}
		element := tsgo.Expression(context.Factory().ElementAccessExpression(
			context.Factory().Identifier(temporaryName),
			nil,
			context.Factory().NumericLiteral(strconv.Itoa(index), tsgo.TokenFlagsNone),
			tsgo.NodeFlagsNone,
		))
		targetType := target.target.SourceType()
		if target.declaration {
			targetType = target.object.Type()
		}
		if target.declaration || !target.target.CopiesValue() {
			copied, err := context.Values().Copy(
				context.WithRole(role),
				source.Rhs[0],
				targetType,
				api.DirectExpression(element),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, copied.Before()...)
			element = copied.Value()
			requests = append(requests, copied.Requests()...)
		}
		if target.declaration {
			if target.storage {
				cell, err := context.AddressableStorage().Cell(
					context,
					children,
					target.identifier,
					target.object.Type(),
					api.DirectExpression(element),
				)
				if err != nil {
					return api.StatementEmission{}, err
				}
				element = cell.Value()
				requests = append(requests, cell.Requests()...)
			}
			targetType, typeRequests, err := pointerAnnotation(
				context.WithRole(api.RoleLocalType),
				children,
				target.identifier,
				target.object.Type(),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			if target.storage {
				targetType = nil
				typeRequests = nil
			}
			statements = append(
				statements,
				typedVariableStatement(
					context,
					tsgo.NodeFlagsLet,
					target.name,
					targetType,
					element,
				),
			)
			requests = append(requests, typeRequests...)
		} else if target.target.IsAccessor() {
			stored, err := target.target.AccessorStore(
				context,
				api.DirectExpression(element),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, stored.Before()...)
			statements = append(
				statements,
				context.Factory().ExpressionStatement(stored.Value()),
			)
			requests = append(requests, stored.Requests()...)
		} else {
			assigned, err := context.Values().Assign(
				context.WithRole(api.RoleAssignmentTarget),
				target.source,
				target.target.SourceType(),
				target.target.Value(),
				api.DirectExpression(element),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(statements, assigned.Before()...)
			statements = append(
				statements,
				context.Factory().ExpressionStatement(assigned.Value()),
			)
			requests = append(requests, assigned.Requests()...)
		}
	}
	return api.NewStatementEmission(statements, requests)
}
