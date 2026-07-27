package assignment

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
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
	case token.ADD_ASSIGN:
		return emitCompoundAddition(context, children, source)
	default:
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
}

func emitCompoundAddition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.StatementEmission, error) {
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
	if !ok ||
		!basictype.SupportsInteger(context.TypesSizes(), object.Type()) ||
		!types.AssignableTo(context.TypesInfo().TypeOf(source.Rhs[0]), object.Type()) {
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
	if len(value.Before()) != 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	target := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(reference.Name()),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorPlusEqualsToken),
		value.Value(),
	)
	return api.DirectStatement(
		context.Factory().ExpressionStatement(target),
		api.CombineRequests(reference.Requests(), value.Requests())...,
	), nil
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
		expression, err := EmitExpression(context, children, source)
		if err != nil {
			return api.ForInitializerEmission{}, err
		}
		if len(expression.Before()) != 0 {
			return api.ForInitializerEmission{},
				api.Unsupported(context, api.CategoryStatement, source)
		}
		return api.ExpressionForInitializer(
			expression.Value(),
			expression.Requests()...,
		)
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

func EmitExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (api.ExpressionEmission, error) {
	target, err := Emit(context, children, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	statements := target.Statements()
	if len(statements) != 1 {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	statement, ok := statements[0].(tsgo.ExpressionStatement)
	if !ok {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	return api.DirectExpression(
		statement.Expression(),
		target.Requests()...,
	), nil
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
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return nil, nil, nil, err
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleLocalValue),
		source.Rhs[0],
		object.Type(),
		value,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	targetType, typeRequests, err := pointerAnnotation(
		context.WithRole(api.RoleLocalType),
		children,
		name,
		object.Type(),
	)
	if err != nil {
		return nil, nil, nil, err
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType,
		value.Value(),
	)
	return context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsLet,
		),
		value.Before(),
		api.CombineRequests(value.Requests(), typeRequests),
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
	target, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.Lhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	sourceType := context.TypesInfo().TypeOf(source.Rhs[0])
	if sourceType == nil || !types.AssignableTo(sourceType, target.SourceType()) {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(target.SourceType()),
		source.Rhs[0],
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	value, err = context.Values().Copy(
		context.WithRole(api.RoleAssignmentValue),
		source.Rhs[0],
		target.SourceType(),
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	if target.IsSetter() {
		return emitSetter(context, target, value)
	}
	assigned, err := context.Values().Assign(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		target.SourceType(),
		target.Value(),
		value,
	)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := target.Before()
	statements = append(statements, assigned.Before()...)
	statements = append(
		statements,
		context.Factory().ExpressionStatement(assigned.Value()),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(target.Requests(), assigned.Requests()),
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
		requests = append(
			requests,
			value.Requests()...,
		)
	}

	for index, target := range targets {
		if target.discard {
			continue
		}
		temporary := context.Factory().Identifier(temporaryNames[index])
		if target.declaration {
			targetType, typeRequests, err := pointerAnnotation(
				context.WithRole(api.RoleLocalType),
				children,
				target.source,
				target.object.Type(),
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
					temporary,
				),
			)
			requests = append(requests, typeRequests...)
		} else {
			assigned, err := context.Values().Assign(
				context.WithRole(api.RoleAssignmentTarget),
				target.source,
				target.object.Type(),
				context.Factory().Identifier(target.name),
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
	value tsgo.Expression,
) tsgo.VariableStatement {
	return typedVariableStatement(context, flags, name, nil, value)
}

func typedVariableStatement(
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

func pointerAnnotation(
	context api.Context,
	children api.ChildEmitter,
	source ast.Node,
	sourceType types.Type,
) (tsgo.TypeNode, []api.PlacementRequest, error) {
	if !context.Values().RequiresExplicitType(context, sourceType) {
		return nil, nil, nil
	}
	target, err := children.RepresentedType(context, source, sourceType)
	if err != nil {
		return nil, nil, err
	}
	return target.Value(), target.Requests(), nil
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
