package assignment

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
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	switch source.Tok {
	case token.DEFINE:
		return emitDefinition(context, children, source)
	case token.ASSIGN:
		return emitAssignment(context, children, source)
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func emitDefinition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	if len(source.Lhs) > 1 {
		return emitParallel(context, children, source)
	}
	declarations, before, requests, err := emitDefinitionList(
		context,
		children,
		source,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := append(
		before,
		context.Factory().VariableStatement(nil, declarations),
	)
	return api.NewStatementEmission(statements, requests)
}

func EmitForInitializer(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.ForInitializerEmission, error) {
	if source.Tok != token.DEFINE {
		return api.ForInitializerEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	declarations, before, requests, err := emitDefinitionList(
		context,
		children,
		source,
	)
	if err != nil {
		return api.ForInitializerEmission{}, err
	}
	if len(before) != 0 {
		return api.ForInitializerEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectForInitializer(declarations, requests...), nil
}

func emitDefinitionList(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (
	tsgo.VariableDeclarationList,
	[]tsgo.Statement,
	[]api.PlacementRequest,
	error,
) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Defs[name].(*types.Var)
	if !ok {
		return nil, nil, nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	targetName, err := context.Names().Declare(object)
	if err != nil {
		return nil, nil, nil, err
	}
	targetType, err := children.Type(
		context.WithRole(api.RoleLocalType),
		name,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return nil, nil, nil, err
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType.Value(),
		value.Value(),
	)
	return context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsLet,
		),
		value.Before(),
		api.CombineRequests(targetType.Requests(), value.Requests()),
		nil
}

func emitAssignment(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
	if len(source.Lhs) > 1 {
		return emitParallel(context, children, source)
	}
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Uses[name].(*types.Var)
	if !ok {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	reference, err := context.Names().Reference(object)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	target := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(reference.Name()),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		value.Value(),
	)
	statements := value.Before()
	statements = append(statements, context.Factory().ExpressionStatement(target))
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(reference.Requests(), value.Requests()),
	)
}

type parallelTarget struct {
	source      *ast.Ident
	object      *types.Var
	name        string
	declaration bool
	discard     bool
	requests    []api.PlacementRequest
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
	requests := make([]api.PlacementRequest, 0, len(targets)*2)
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
		temporaryType, err := children.RepresentedType(
			context.WithRole(api.RoleLocalType),
			target.source,
			expectedType,
		)
		if err != nil {
			return api.StatementEmission{}, err
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
				temporaryType.Value(),
				value.Value(),
			),
		)
		requests = append(
			requests,
			value.Requests()...,
		)
		requests = append(requests, temporaryType.Requests()...)
	}

	for index, target := range targets {
		if target.discard {
			continue
		}
		temporary := context.Factory().Identifier(temporaryNames[index])
		if target.declaration {
			targetType, err := children.RepresentedType(
				context.WithRole(api.RoleLocalType),
				target.source,
				target.object.Type(),
			)
			if err != nil {
				return api.StatementEmission{}, err
			}
			statements = append(
				statements,
				variableStatement(
					context,
					tsgo.NodeFlagsLet,
					target.name,
					targetType.Value(),
					temporary,
				),
			)
			requests = append(requests, targetType.Requests()...)
		} else {
			statements = append(
				statements,
				assignmentStatement(context, target.name, temporary),
			)
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
				name, err := context.Names().Declare(object)
				if err != nil {
					return nil, err
				}
				targets = append(targets, parallelTarget{
					source:      identifier,
					object:      object,
					name:        name,
					declaration: true,
				})
				continue
			}
		}
		object, ok := context.TypesInfo().Uses[identifier].(*types.Var)
		if !ok {
			return nil, api.Unsupported(context, api.CategoryStatement, source)
		}
		reference, err := context.Names().Reference(object)
		if err != nil {
			return nil, err
		}
		targets = append(targets, parallelTarget{
			source:   identifier,
			object:   object,
			name:     reference.Name(),
			requests: reference.Requests(),
		})
	}
	return targets, nil
}

func variableStatement(
	context api.Context,
	flags tsgo.NodeFlags,
	name string,
	targetType tsgo.TypeNode,
	value tsgo.Expression,
) tsgo.VariableStatement {
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(name),
		nil,
		targetType,
		value,
	)
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			flags,
		),
	)
}

func assignmentStatement(
	context api.Context,
	name string,
	value tsgo.Expression,
) tsgo.ExpressionStatement {
	target := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(name),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		value,
	)
	return context.Factory().ExpressionStatement(target)
}
