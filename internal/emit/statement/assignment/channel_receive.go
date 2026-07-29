package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func EmitSelectedReceive(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
	elementType types.Type,
	value api.ExpressionEmission,
	okValue api.ExpressionEmission,
) (api.StatementEmission, error) {
	if source == nil ||
		(source.Tok != token.DEFINE && source.Tok != token.ASSIGN) ||
		len(source.Rhs) != 1 ||
		len(source.Lhs) < 1 ||
		len(source.Lhs) > 2 ||
		elementType == nil {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	values := []selectedReceiveValue{
		{sourceType: elementType, emission: value},
	}
	if len(source.Lhs) == 2 {
		values = append(values, selectedReceiveValue{
			sourceType: types.Typ[types.Bool],
			emission:   okValue,
		})
	}
	targets, before, requests, err := parallelTargets(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if len(targets) != len(values) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	for index, target := range targets {
		if target.discard {
			continue
		}
		targetType := target.target.SourceType()
		if target.declaration {
			targetType = target.object.Type()
		}
		if !types.AssignableTo(values[index].sourceType, targetType) {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source.Lhs[index])
		}
	}
	for index, target := range targets {
		if target.discard {
			continue
		}
		selected := values[index].emission
		before = append(before, selected.Before()...)
		requests = append(requests, selected.Requests()...)
		if target.declaration {
			targetValue := selected.Value()
			if target.storage {
				cell, err := context.AddressableStorage().Cell(
					context,
					children,
					target.identifier,
					target.object.Type(),
					api.DirectExpression(targetValue),
				)
				if err != nil {
					return api.StatementEmission{}, err
				}
				targetValue = cell.Value()
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
			before = append(before, typedVariableStatement(
				context,
				tsgo.NodeFlagsLet,
				target.name,
				targetType,
				targetValue,
			))
			requests = append(requests, typeRequests...)
			continue
		}
		stored, err := target.target.StoreValue(
			context.WithRole(api.RoleAssignmentTarget),
			target.source,
			selected,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		before = append(before, stored.Before()...)
		before = append(
			before,
			context.Factory().ExpressionStatement(stored.Value()),
		)
		requests = append(requests, stored.Requests()...)
	}
	return api.NewStatementEmission(before, requests)
}

type selectedReceiveValue struct {
	sourceType types.Type
	emission   api.ExpressionEmission
}
