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
	if context.Values().RequiresCustomUpdate(
		context,
		target.SourceType(),
	) {
		return emitCustom(context, source, target)
	}
	if !basictype.SupportsInteger(context.TypesSizes(), target.SourceType()) {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if target.IsAccessor() {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
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
	return api.DirectExpression(
		context.Factory().PostfixUnaryExpression(
			target.Value(),
			operator,
		),
		target.Requests()...,
	), nil
}

func emitCustom(
	context api.Context,
	source *ast.IncDecStmt,
	target api.StoreTargetEmission,
) (api.ExpressionEmission, error) {
	left := target.Value()
	if target.IsAccessor() {
		var err error
		target, err = target.CaptureAccessorLocation(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
		left, err = target.AccessorRead(context)
		if err != nil {
			return api.ExpressionEmission{}, err
		}
	}
	result, handled, err := context.Values().Increment(
		context,
		source,
		target.SourceType(),
		source.Tok,
		left,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	if !handled {
		return api.ExpressionEmission{},
			api.Unsupported(context, api.CategoryStatement, source)
	}
	if target.IsAccessor() {
		return target.AccessorStore(context, result)
	}
	assigned, err := context.Values().Assign(
		context.WithRole(api.RoleAssignmentTarget),
		source,
		target.SourceType(),
		target.Value(),
		result,
	)
	if err != nil {
		return api.ExpressionEmission{}, err
	}
	return api.NewExpressionEmission(
		append(target.Before(), assigned.Before()...),
		assigned.Value(),
		api.CombineRequests(target.Requests(), assigned.Requests()),
	)
}
