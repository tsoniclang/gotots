package representation

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	definedbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/defined"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (Owner) RequiresCustomUpdate(
	_ api.Context,
	sourceType types.Type,
) bool {
	_, ok := definedtype.Resolve(sourceType)
	return ok
}

func (Owner) BinaryUpdate(
	context api.Context,
	source ast.Node,
	rightSource ast.Expr,
	sourceType types.Type,
	rightRepresentation types.Type,
	operator token.Token,
	left tsgo.Expression,
	right api.ExpressionEmission,
) (api.ExpressionEmission, bool, error) {
	model, ok := definedtype.Resolve(sourceType)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	rightBefore := right.Before()
	rightRequests := right.Requests()
	rightConstant := constantEvidence(context, rightSource)
	if rightConstant != nil {
		var err error
		right, err = constantvalue.EmitValue(
			context.WithRole(api.RoleAssignmentValue),
			rightSource,
			model.Underlying(),
			rightConstant,
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		rightRequests = api.CombineRequests(
			rightRequests,
			right.Requests(),
		)
	} else if rightModel, wrapped := definedtype.Resolve(
		rightRepresentation,
	); wrapped {
		var err error
		right, err = api.NewExpressionEmission(
			right.Before(),
			rightModel.Unwrap(context.Factory(), right.Value()),
			right.Requests(),
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
	}
	right = api.DirectExpression(right.Value(), rightRequests...)
	result, handled, err := definedbinary.ApplyUnderlying(
		context,
		operator,
		model.Underlying(),
		api.DirectExpression(model.Unwrap(context.Factory(), left)),
		right,
		rightConstant,
	)
	if err != nil || !handled {
		return result, handled, err
	}
	result, err = model.Wrap(context, result)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, err = api.NewExpressionEmission(
		rightBefore,
		result.Value(),
		result.Requests(),
	)
	return result, true, err
}

func (Owner) Increment(
	context api.Context,
	source ast.Node,
	sourceType types.Type,
	operator token.Token,
	left tsgo.Expression,
) (api.ExpressionEmission, bool, error) {
	model, ok := definedtype.Resolve(sourceType)
	if !ok {
		return api.ExpressionEmission{}, false, nil
	}
	binaryOperator := token.ILLEGAL
	switch operator {
	case token.INC:
		binaryOperator = token.ADD
	case token.DEC:
		binaryOperator = token.SUB
	default:
		return api.ExpressionEmission{}, true,
			api.Unsupported(context, api.CategoryStatement, source)
	}
	one := constant.MakeInt64(1)
	right, err := constantvalue.EmitValue(
		context.WithRole(api.RoleAssignmentValue),
		source,
		model.Underlying(),
		one,
	)
	if err != nil {
		return api.ExpressionEmission{}, true, err
	}
	result, handled, err := definedbinary.ApplyUnderlying(
		context,
		binaryOperator,
		model.Underlying(),
		api.DirectExpression(model.Unwrap(context.Factory(), left)),
		right,
		one,
	)
	if err != nil || !handled {
		return result, handled, err
	}
	result, err = model.Wrap(context, result)
	return result, true, err
}

func constantEvidence(
	context api.Context,
	source ast.Expr,
) constant.Value {
	facts, ok := context.TypesInfo().Types[source]
	if !ok {
		return nil
	}
	return facts.Value
}
