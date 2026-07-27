package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type parallelTarget struct {
	source      *ast.Ident
	object      *types.Var
	name        string
	declaration bool
	discard     bool
	requests    []api.RootRequest
	storage     bool
}

func emitParallel(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	if len(source.Lhs) < 2 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if len(source.Rhs) == 1 {
		if tuple, ok := context.TypesInfo().TypeOf(source.Rhs[0]).(*types.Tuple); ok {
			return emitMultipleResults(context, children, source, tuple)
		}
	}
	if len(source.Lhs) != len(source.Rhs) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	targets, err := parallelTargets(context, source)
	if err != nil {
		return api.StatementEmission{}, err
	}

	statements := make([]tsgo.Statement, 0, len(targets)*2)
	requests := make([]api.RootRequest, 0, len(targets)*2)
	temporaryNames := make([]string, len(targets))
	for index, target := range targets {
		sourceValue := source.Rhs[index]
		sourceType := context.TypesInfo().TypeOf(sourceValue)
		if sourceType == nil {
			return api.StatementEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		expectedType := sourceType
		if !target.discard {
			expectedType = target.object.Type()
			if !types.AssignableTo(sourceType, expectedType) {
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
				WithExpectedType(expectedType),
			sourceValue,
		)
		if err != nil {
			return api.StatementEmission{}, err
		}
		if !target.discard {
			value, err = context.Values().Copy(
				context.WithRole(role),
				sourceValue,
				expectedType,
				value,
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
		}
		temporaryName, err := context.Names().Temporary(api.TemporaryAssignmentValue)
		if err != nil {
			return api.StatementEmission{}, err
		}
		temporaryNames[index] = temporaryName
		statements = append(statements, value.Before()...)
		statements = append(
			statements,
			variableStatement(
				context,
				tsgo.NodeFlagsConst,
				temporaryName,
				value.Value(),
			),
		)
		requests = append(requests, value.Requests()...)
	}

	for index, target := range targets {
		if target.discard {
			continue
		}
		temporary := tsgo.Expression(
			context.Factory().Identifier(temporaryNames[index]),
		)
		if target.declaration {
			value := api.DirectExpression(temporary)
			if target.storage {
				value, err = context.AddressableStorage().Cell(
					context,
					children,
					target.source,
					target.object.Type(),
					value,
				)
				if err != nil {
					return api.StatementEmission{}, err
				}
				temporary = value.Value()
				requests = append(requests, value.Requests()...)
			}
			targetType, typeRequests, err := pointerAnnotation(
				context.WithRole(api.RoleLocalType),
				children,
				target.source,
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
					temporary,
				),
			)
			requests = append(requests, typeRequests...)
		} else {
			targetExpression, err := parallelTargetExpression(context, target)
			if err != nil {
				return api.StatementEmission{}, err
			}
			assigned, err := context.Values().Assign(
				context.WithRole(api.RoleAssignmentTarget),
				target.source,
				target.object.Type(),
				targetExpression,
				api.DirectExpression(temporary),
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
		requests = append(requests, target.requests...)
	}
	return api.NewStatementEmission(statements, requests)
}

func parallelTargets(
	context api.Context,
	source *ast.AssignStmt,
) ([]parallelTarget, error) {
	targets := make([]parallelTarget, 0, len(source.Lhs))
	for _, expression := range source.Lhs {
		identifier, ok := expression.(*ast.Ident)
		if !ok {
			return nil, api.Unsupported(context, api.CategoryStatement, source)
		}
		if identifier.Name == "_" {
			targets = append(targets, parallelTarget{
				source:  identifier,
				discard: true,
			})
			continue
		}
		if source.Tok == token.DEFINE {
			if object, ok := context.TypesInfo().Defs[identifier].(*types.Var); ok {
				name, selected := context.AddressableStorage().Name(
					context,
					object,
				)
				var err error
				if !selected {
					name, err = context.Names().Declare(object)
				}
				if err != nil {
					return nil, err
				}
				targets = append(targets, parallelTarget{
					source:      identifier,
					object:      object,
					name:        name,
					declaration: true,
					storage:     selected,
				})
				continue
			}
		}
		object, ok := context.TypesInfo().Uses[identifier].(*types.Var)
		if !ok {
			return nil, api.Unsupported(context, api.CategoryStatement, source)
		}
		name, selected := context.AddressableStorage().Name(context, object)
		var requests []api.RootRequest
		if !selected {
			reference, err := context.Names().Reference(object)
			if err != nil {
				return nil, err
			}
			name = reference.Name()
			requests = reference.Requests()
		}
		targets = append(targets, parallelTarget{
			source:   identifier,
			object:   object,
			name:     name,
			requests: requests,
			storage:  selected,
		})
	}
	return targets, nil
}

func parallelTargetExpression(
	context api.Context,
	target parallelTarget,
) (tsgo.Expression, error) {
	value := tsgo.Expression(context.Factory().Identifier(target.name))
	if target.storage {
		selected, ok, err := context.AddressableStorage().StoreTarget(
			context,
			target.object,
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, &api.InvariantError{
				Role:   context.Role(),
				Reason: "selected parallel target has no addressable storage",
			}
		}
		value = selected.Value()
	}
	return value, nil
}
