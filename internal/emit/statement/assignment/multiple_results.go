package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/resulttuple"
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
			targetType = target.sourceType
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
	capture, err := resulttuple.Emit(
		context,
		children,
		source.Rhs[0],
		results,
		role,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(locationBefore, capture.Statements()...)
	requests := api.CombineRequests(locationRequests, capture.Requests())

	for index, target := range targets {
		if target.discard {
			continue
		}
		element, err := capture.Element(context, index)
		if err != nil {
			return api.StatementEmission{}, err
		}
		targetType := target.target.SourceType()
		if target.declaration {
			targetType = target.sourceType
		}
		mode := api.ValueTransferCopy
		if !target.declaration {
			mode = storeTransferMode(target.target)
		}
		value, err := context.Values().Transfer(
			context.WithRole(role),
			source.Rhs[0],
			results.At(index).Type(),
			targetType,
			mode,
			api.DirectExpression(element),
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		statements = append(statements, value.Before()...)
		element = value.Value()
		requests = append(requests, value.Requests()...)
		if target.declaration {
			if context.IsGotoLocal(target.object) {
				statements = append(
					statements,
					assignmentStatement(context, target.name, element),
				)
				continue
			}
			targetType, typeRequests, err := pointerAnnotation(
				context.WithRole(api.RoleLocalType),
				children,
				target.identifier,
				target.sourceType,
			)
			if err != nil {
				return api.StatementEmission{}, err
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
		} else {
			stored, err := target.target.StoreValue(
				context.WithRole(api.RoleAssignmentTarget),
				target.source,
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
		}
	}
	return api.NewStatementEmission(statements, requests)
}
