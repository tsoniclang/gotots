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
	switch source.Lhs[0].(type) {
	case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr:
	default:
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
	if target.IsSetter() ||
		!basictype.SupportsInteger(context.TypesSizes(), target.SourceType()) ||
		!types.AssignableTo(
			context.TypesInfo().TypeOf(source.Rhs[0]),
			target.SourceType(),
		) {
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
	if len(value.Before()) != 0 {
		return api.StatementEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	expression := context.Factory().BinaryExpression(
		nil,
		target.Value(),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorPlusEqualsToken),
		value.Value(),
	)
	statements := target.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(expression),
	)
	return api.NewStatementEmission(
		statements,
		api.CombineRequests(target.Requests(), value.Requests()),
	)
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
	[]api.RootRequest,
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
	targetName, selected := context.AddressableStorage().Name(context, object)
	var err error
	if !selected {
		targetName, err = context.Names().Declare(object)
	}
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
	if selected {
		value, err = context.AddressableStorage().Cell(
			context,
			children,
			name,
			object.Type(),
			value,
		)
		if err != nil {
			return nil, nil, nil, err
		}
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
	if selected {
		targetType = nil
		typeRequests = nil
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
) (tsgo.TypeNode, []api.RootRequest, error) {
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
