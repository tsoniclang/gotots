package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type parallelTarget struct {
	source      ast.Expr
	identifier  *ast.Ident
	object      *types.Var
	name        string
	declaration bool
	discard     bool
	storage     bool
	target      api.StoreTargetEmission
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
	targets, locationBefore, locationRequests, err := parallelTargets(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}

	statements := append(
		make([]tsgo.Statement, 0, len(locationBefore)+len(targets)*2),
		locationBefore...,
	)
	requests := append(
		make([]api.RootRequest, 0, len(locationRequests)+len(targets)*2),
		locationRequests...,
	)
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
			if target.declaration {
				expectedType = target.object.Type()
			} else {
				expectedType = target.target.SourceType()
			}
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
		if !target.discard &&
			(target.declaration || !target.target.CopiesValue()) {
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
					target.identifier,
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
					temporary,
				),
			)
			requests = append(requests, typeRequests...)
		} else {
			stored, err := target.target.StoreValue(
				context.WithRole(api.RoleAssignmentTarget),
				target.source,
				api.DirectExpression(temporary),
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

func parallelTargets(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (
	[]parallelTarget,
	[]tsgo.Statement,
	[]api.RootRequest,
	error,
) {
	targets := make([]parallelTarget, 0, len(source.Lhs))
	var before []tsgo.Statement
	var requests []api.RootRequest
	for _, expression := range source.Lhs {
		identifier, identifierOK := expression.(*ast.Ident)
		if identifierOK && identifier.Name == "_" {
			targets = append(targets, parallelTarget{
				source:  identifier,
				discard: true,
			})
			continue
		}
		if source.Tok == token.DEFINE && identifierOK {
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
					return nil, nil, nil, err
				}
				targets = append(targets, parallelTarget{
					source:      identifier,
					identifier:  identifier,
					object:      object,
					name:        name,
					declaration: true,
					storage:     selected,
				})
				continue
			}
		}
		target, err := children.StoreTarget(
			context.WithRole(api.RoleAssignmentTarget),
			expression,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		target, targetBefore, targetRequests, err := target.PrepareLocation(context)
		if err != nil {
			return nil, nil, nil, err
		}
		before = append(before, targetBefore...)
		requests = append(requests, targetRequests...)
		targets = append(targets, parallelTarget{
			source: expression,
			target: target,
		})
	}
	return targets, before, requests, nil
}
