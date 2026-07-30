package nilcomparison

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
	source *ast.BinaryExpr,
) (api.ExpressionEmission, bool, error) {
	valueSource, role, valueType, negated, selected :=
		SelectSource(context.TypesInfo(), source)
	if !selected {
		return api.ExpressionEmission{}, false, nil
	}
	if _, generic := api.GenericTypeParameter(valueType); generic {
		return api.ExpressionEmission{}, false, nil
	}
	info := context.TypesInfo()
	if info == nil ||
		!types.AssignableTo(info.TypeOf(source), types.Typ[types.Bool]) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	value, err := children.Expression(
		context.WithRole(role).WithExpectedType(valueType),
		valueSource,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, handled, err := Apply(context, source, valueType, value)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !handled {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "nil-comparison has no concrete representation",
		}
	}
	if !negated {
		return target, true, nil
	}
	result, err := api.NewExpressionEmission(
		target.Before(),
		context.Factory().PrefixUnaryExpression(
			tsgo.PrefixUnaryExpressionOperatorKindExclamationToken,
			target.Value(),
		),
		target.Requests(),
	)
	return result, true, err
}

func SelectSource(
	info *types.Info,
	source *ast.BinaryExpr,
) (ast.Expr, api.Role, types.Type, bool, bool) {
	if info == nil ||
		source == nil ||
		(source.Op != token.EQL && source.Op != token.NEQ) {
		return nil, api.Role(""), nil, false, false
	}
	if isNil(info, source.Y) {
		return source.X,
			api.RoleBinaryLeft,
			info.TypeOf(source.X),
			source.Op == token.NEQ,
			true
	}
	if isNil(info, source.X) {
		return source.Y,
			api.RoleBinaryRight,
			info.TypeOf(source.Y),
			source.Op == token.NEQ,
			true
	}
	return nil, api.Role(""), nil, false, false
}

func isNil(info *types.Info, source ast.Expr) bool {
	identifier, ok := source.(*ast.Ident)
	return ok && info.Uses[identifier] == types.Universe.Lookup("nil")
}
