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
) (tsgo.Statement, error) {
	switch source.Tok {
	case token.DEFINE:
		return emitDefinition(context, children, source)
	case token.ASSIGN:
		return emitAssignment(context, children, source)
	default:
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
}

func emitDefinition(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (tsgo.Statement, error) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Defs[name].(*types.Var)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	targetName, err := context.Names().Declare(object)
	if err != nil {
		return nil, err
	}
	targetType, err := children.Type(
		context.WithRole(api.RoleLocalType),
		name,
	)
	if err != nil {
		return nil, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleLocalValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return nil, err
	}
	declaration := context.Factory().VariableDeclaration(
		context.Factory().Identifier(targetName),
		nil,
		targetType,
		value,
	)
	return context.Factory().VariableStatement(
		nil,
		context.Factory().VariableDeclarationList(
			[]tsgo.VariableDeclaration{declaration},
			tsgo.NodeFlagsLet,
		),
	), nil
}

func emitAssignment(
	context api.Context,
	children api.ChildEmitter,
	source *ast.AssignStmt,
) (tsgo.Statement, error) {
	if len(source.Lhs) != 1 || len(source.Rhs) != 1 {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	name, ok := source.Lhs[0].(*ast.Ident)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	object, ok := context.TypesInfo().Uses[name].(*types.Var)
	if !ok {
		return nil, api.Unsupported(context, api.CategoryStatement, source)
	}
	targetName, err := context.Names().Reference(object)
	if err != nil {
		return nil, err
	}
	value, err := children.Expression(
		context.
			WithRole(api.RoleAssignmentValue).
			WithExpectedType(object.Type()),
		source.Rhs[0],
	)
	if err != nil {
		return nil, err
	}
	target := context.Factory().BinaryExpression(
		nil,
		context.Factory().Identifier(targetName),
		nil,
		context.Factory().BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		value,
	)
	return context.Factory().ExpressionStatement(target), nil
}
