package representation

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantvalue "github.com/tsoniclang/gotots/internal/emit/constant"
	basicbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/basic"
	complexbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/complex"
	definedbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/defined"
	floatbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/float"
	integerbinary "github.com/tsoniclang/gotots/internal/emit/expression/binary/integer"
	genericoperation "github.com/tsoniclang/gotots/internal/emit/generic/operation"
	definedtype "github.com/tsoniclang/gotots/internal/emit/type/defined"
	complexvalue "github.com/tsoniclang/gotots/internal/emit/value/complex"
	floatvalue "github.com/tsoniclang/gotots/internal/emit/value/float"
	integervalue "github.com/tsoniclang/gotots/internal/emit/value/integer"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
	if _, leftGeneric := api.GenericTypeParameter(sourceType); leftGeneric {
		operation, ok := api.BinaryGenericOperation(operator)
		if !ok {
			return api.ExpressionEmission{}, true, &api.InvariantError{
				Role:   context.Role(),
				Reason: "generic binary update operation is invalid",
			}
		}
		target, err := genericoperation.Call(
			context,
			source,
			operation,
			[]types.Type{sourceType, rightRepresentation},
			[]types.Type{sourceType},
			[]api.ExpressionEmission{
				api.DirectExpression(left),
				right,
			},
		)
		if err != nil {
			return api.ExpressionEmission{}, true, err
		}
		return target, true, nil
	}
	model, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		return primitiveBinaryUpdate(
			context,
			sourceType,
			operator,
			left,
			right,
			constantEvidence(context, rightSource),
		)
	}
	underlying, valid := model.Basic()
	if !valid {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined basic update has no basic underlying type",
		}
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
	} else if rightModel, wrapped := definedtype.ResolveBasic(
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
		underlying,
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
	model, ok := definedtype.ResolveBasic(sourceType)
	if !ok {
		binaryOperator, valid := incrementOperator(operator)
		if !valid {
			return api.ExpressionEmission{}, true,
				api.Unsupported(context, api.CategoryStatement, source)
		}
		one := constant.MakeInt64(1)
		right, err := constantvalue.EmitValue(
			context.WithRole(api.RoleAssignmentValue),
			source,
			sourceType,
			one,
		)
		if err != nil {
			return api.ExpressionEmission{}, false, nil
		}
		return primitiveBinaryUpdate(
			context,
			sourceType,
			binaryOperator,
			left,
			right,
			one,
		)
	}
	underlying, valid := model.Basic()
	if !valid {
		return api.ExpressionEmission{}, true, &api.InvariantError{
			Role:   context.Role(),
			Reason: "defined basic increment has no basic underlying type",
		}
	}
	binaryOperator, valid := incrementOperator(operator)
	if !valid {
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
		underlying,
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

func incrementOperator(operator token.Token) (token.Token, bool) {
	switch operator {
	case token.INC:
		return token.ADD, true
	case token.DEC:
		return token.SUB, true
	default:
		return token.ILLEGAL, false
	}
}

func primitiveBinaryUpdate(
	context api.Context,
	sourceType types.Type,
	operator token.Token,
	left tsgo.Expression,
	right api.ExpressionEmission,
	rightConstant constant.Value,
) (api.ExpressionEmission, bool, error) {
	rightValue := api.DirectExpression(
		right.Value(),
		right.Requests()...,
	)
	leftValue := api.DirectExpression(left)
	var (
		result  api.ExpressionEmission
		handled bool
		err     error
	)
	if carrier, ok := integervalue.Describe(
		context.TypesSizes(),
		sourceType,
	); ok {
		switch {
		case (operator == token.SHL || operator == token.SHR) &&
			rightConstant == nil &&
			integervalue.SupportsVariableShift(
				context.IntegerRepresentation(),
				carrier,
				operator,
			):
			result, handled, err = integerbinary.ApplyVariableShift(
				context,
				operator,
				carrier,
				leftValue,
				rightValue,
			)
		case integervalue.SupportsArithmetic(
			context.IntegerRepresentation(),
			operator,
		),
			integervalue.SupportsBitwise(
				context.IntegerRepresentation(),
				carrier,
				operator,
			),
			integervalue.SupportsShift(
				context.IntegerRepresentation(),
				carrier,
				operator,
				rightConstant,
			):
			result, handled, err = integerbinary.Apply(
				context,
				operator,
				carrier,
				leftValue,
				rightValue,
			)
		default:
			return api.ExpressionEmission{}, false, nil
		}
	} else if carrier, ok := floatvalue.Describe(sourceType); ok {
		result, handled, err = floatbinary.Apply(
			context,
			operator,
			carrier,
			leftValue,
			rightValue,
		)
	} else if carrier, ok := complexvalue.Describe(sourceType); ok {
		result, handled, err = complexbinary.Apply(
			context,
			operator,
			carrier,
			leftValue,
			rightValue,
		)
	} else {
		result, handled = basicbinary.Apply(
			context,
			sourceType,
			operator,
			leftValue,
			rightValue,
		)
	}
	if err != nil || !handled {
		return result, handled, err
	}
	result, err = api.NewExpressionEmission(
		right.Before(),
		result.Value(),
		result.Requests(),
	)
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
