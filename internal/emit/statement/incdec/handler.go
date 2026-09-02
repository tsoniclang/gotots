package incdec

import (
	"go/ast"
	"go/token"

	"github.com/tsoniclang/gotots/internal/emit/api"
	basictype "github.com/tsoniclang/gotots/internal/emit/type/basic"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IncDecStmt,
) (api.StatementEmission, error) {
	expression, err := EmitExpression(context, children, source)
	if err != nil {
		return api.StatementEmission{}, err
	}
	statements := expression.Before()
	statements = append(
		statements,
		context.Factory().ExpressionStatement(expression.Value()),
	)
	return api.NewStatementEmission(statements, expression.Requests())
}

func EmitExpression(
	context api.Context,
	children api.ChildEmitter,
	source *ast.IncDecStmt,
) (api.ExpressionEmission, error) {
	target, err := children.StoreTarget(
		context.WithRole(api.RoleAssignmentTarget),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	requiresStorageProjection, err := context.Values().RequiresStorageProjection(
		context,
		target.SourceType(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	var operator tsgo.PostfixUnaryExpressionOperatorKind
	switch source.Tok {
	case token.INC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindPlusPlusToken
	case token.DEC:
		operator = tsgo.PostfixUnaryExpressionOperatorKindMinusMinusToken
	default:
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	integerAlias, supportsInteger := basictype.IntegerAlias(
		context.TypesSizes(),
		target.SourceType(),
	)
	if supportsInteger &&
		!context.ScalarABI().RequiresExactIntegerResult(integerAlias) &&
		!target.IsAccessor() &&
		!target.IsProperty() &&
		(!target.UsesCanonicalStorage() || !requiresStorageProjection) {
		return api.DirectExpression(
			context.Factory().PostfixUnaryExpression(
				target.Value(),
				operator,
			),
			target.Requests()...,
		), nil
	}
	return emitCustom(context, source, target)
}

func emitCustom(
	context api.Context,
	source *ast.IncDecStmt,
	target api.StoreTargetEmission,
) (api.ExpressionEmission, error) {
	target, err := target.CaptureLocation(context)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	left, err := target.ReadValue(context, source)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	result, handled, err := context.Values().Increment(
		context,
		source,
		target.SourceType(),
		source.Tok,
		left.Value(),
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	stored, err := target.StoreValue(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		result,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return stored, nil
}
