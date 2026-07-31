package defined

import (
	"go/ast"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	unaryoperation "github.com/tsoniclang/gotots/internal/emit/expression/unary/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
)

func Emit(
	context api.Context,
	children api.ChildEmitter,
	source *ast.UnaryExpr,
) (api.ExpressionEmission, bool, error) {
	model, ok := definedtype.ResolveBasic(context.TypesInfo().TypeOf(source))
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	operandType := context.TypesInfo().TypeOf(source.X)
	if !types.AssignableTo(operandType, model.Type()) {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	operand, err := children.Expression(
		context.
			WithRole(api.RoleUnaryOperand).
			WithExpectedType(model.Type()),
		source.X,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	operand, err = model.Project(context, operand)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	target, handled, err := unaryoperation.Apply(
		context,
		source.Op,
		model.Underlying(),
		operand,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	if !handled {
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryExpression, source)
	}
	target, err = model.Wrap(context, target)
	return target, true, err
}
